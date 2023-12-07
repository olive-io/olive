// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
	"go.etcd.io/etcd/pkg/v3/traceutil"
)

const v1Version = "v1"

type applyResult struct {
	resp proto.Message
	err  error
	// physc signals the physical effect of the request has completed in addition
	// to being logically reflected by the node. Currently only used for
	// Compaction requests.
	physc <-chan struct{}
	trace *traceutil.Trace
}

type Applier interface {
	Apply(ctx context.Context, r *pb.RaftInternalRequest) *applyResult
	Range(ctx context.Context, r *pb.RegionRangeRequest) (*pb.RegionRangeResponse, error)
	Put(ctx context.Context, r *pb.RegionPutRequest) (*pb.RegionPutResponse, *traceutil.Trace, error)
	Delete(ctx context.Context, r *pb.RegionDeleteRequest) (*pb.RegionDeleteResponse, *traceutil.Trace, error)

	DeployDefinition(ctx context.Context, r *pb.RegionDeployDefinitionRequest) (*pb.RegionDeployDefinitionResponse, *traceutil.Trace, error)
	ExecuteDefinition(ctx context.Context, r *pb.RegionExecuteDefinitionRequest) (*pb.RegionExecuteDefinitionResponse, *traceutil.Trace, error)
}

type applier struct {
	r *Region
}

func (r *Region) newApplier() *applier {
	return &applier{r: r}
}

func (a *applier) Apply(ctx context.Context, r *pb.RaftInternalRequest) *applyResult {
	op := "unknown"
	ar := &applyResult{}
	defer func(start time.Time) {
		//success := ar.err == nil || ar.err == mvcc.ErrCompacted
		success := ar.err == nil
		a.r.metric.applySec.WithLabelValues(v1Version, op, strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
		warnOfExpensiveRequest(a.r.lg, a.r.metric.slowApplies, a.r.WarningApplyDuration, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		if !success {
			warnOfFailedRequest(a.r.lg, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		}
	}(time.Now())

	switch {
	case r.Range != nil:
		op = "Range"
		ar.resp, ar.err = a.Range(ctx, r.Range)
	case r.Put != nil:
		op = "Put"
		ar.resp, ar.trace, ar.err = a.Put(ctx, r.Put)
	case r.Delete != nil:
		op = "Delete"
		ar.resp, ar.trace, ar.err = a.Delete(ctx, r.Delete)
	case r.DeployDefinition != nil:
		op = "DeployDefinition"
		ar.resp, ar.trace, ar.err = a.DeployDefinition(ctx, r.DeployDefinition)
	case r.ExecuteDefinition != nil:
		op = "ExecuteDefinition"
		ar.resp, ar.trace, ar.err = a.ExecuteDefinition(ctx, r.ExecuteDefinition)
	}

	return ar
}

func (a *applier) Range(ctx context.Context, r *pb.RegionRangeRequest) (*pb.RegionRangeResponse, error) {
	trace := traceutil.Get(ctx)

	resp := &pb.RegionRangeResponse{}
	resp.Header = &pb.RaftResponseHeader{}

	limit := r.Limit
	kvs, err := a.r.getRange(r.Key, mkGteRange(r.RangeEnd), limit)
	if err != nil {
		return nil, err
	}
	resp.Kvs = kvs
	trace.Step("assemble the response")
	return resp, nil
}

func (a *applier) Put(ctx context.Context, r *pb.RegionPutRequest) (*pb.RegionPutResponse, *traceutil.Trace, error) {
	resp := &pb.RegionPutResponse{}
	resp.Header = &pb.RaftResponseHeader{}
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("put",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "key", Value: string(r.Key)},
			traceutil.Field{Key: "req_size", Value: r.XSize()},
		)
	}

	err := a.r.put(r.Key, r.Value, r.IsSync)
	if err != nil {
		return nil, nil, err
	}
	return resp, trace, nil
}

func (a *applier) Delete(ctx context.Context, r *pb.RegionDeleteRequest) (*pb.RegionDeleteResponse, *traceutil.Trace, error) {
	resp := &pb.RegionDeleteResponse{}
	resp.Header = &pb.RaftResponseHeader{}
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("delete",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "key", Value: string(r.Key)},
		)
	}

	err := a.r.del(r.Key, r.IsSync)
	if err != nil {
		return nil, trace, err
	}
	return resp, trace, nil
}

func (a *applier) DeployDefinition(ctx context.Context, r *pb.RegionDeployDefinitionRequest) (*pb.RegionDeployDefinitionResponse, *traceutil.Trace, error) {
	resp := &pb.RegionDeployDefinitionResponse{}
	resp.Header = &pb.RaftResponseHeader{}
	definition := r.Definition
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("deploy_definition",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "definition_id", Value: definition.Id},
			traceutil.Field{Key: "definition_version", Value: definition.Version},
		)
	}

	prefix := bytesutil.PathJoin(definitionPrefix, []byte(definition.Id))
	key := bytesutil.PathJoin(prefix, []byte(fmt.Sprintf("%d", definition.Version)))

	kvs, _ := a.r.getRange(prefix, getPrefix(prefix), 0)
	if len(kvs) > 0 {
		a.r.metric.definition.Add(1)
	}

	trace.Step("save definition", traceutil.Field{Key: "key", Value: key})
	data, _ := definition.Marshal()
	if err := a.r.put(key, data, true); err != nil {
		return nil, trace, err
	}
	resp.Definition = definition

	return resp, trace, nil
}

func (a *applier) ExecuteDefinition(ctx context.Context, r *pb.RegionExecuteDefinitionRequest) (*pb.RegionExecuteDefinitionResponse, *traceutil.Trace, error) {
	resp := &pb.RegionExecuteDefinitionResponse{}
	resp.Header = &pb.RaftResponseHeader{}
	process := r.ProcessInstance
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("deploy_definition",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "process_instance", Value: process.Id},
			traceutil.Field{Key: "definition_id", Value: process.DefinitionId},
			traceutil.Field{Key: "definition_version", Value: process.DefinitionVersion},
		)
	}

	prefix := bytesutil.PathJoin(processPrefix,
		[]byte(process.DefinitionId),
		[]byte(fmt.Sprintf("%d", process.DefinitionVersion)))
	key := bytesutil.PathJoin(prefix, []byte(fmt.Sprintf("%d", process.Id)))

	if kv, _ := a.r.get(key); kv != nil {
		return nil, trace, ErrProcessExecuted
	}

	trace.Step("save process", traceutil.Field{Key: "key", Value: key})
	process.Status = pb.ProcessInstance_Prepare
	data, _ := process.Marshal()
	if err := a.r.put(key, data, true); err != nil {
		return nil, trace, err
	}
	resp.ProcessInstance = process
	a.r.processQ.Set(process)

	return resp, trace, nil
}

// mkGteRange determines if the range end is a >= range. This works around grpc
// sending empty byte strings as nil; >= is encoded in the range end as '\0'.
// If it is a GTE range, then []byte{} is returned to indicate the empty byte
// string (vs nil being no byte string).
func mkGteRange(rangeEnd []byte) []byte {
	if len(rangeEnd) == 1 && rangeEnd[0] == 0 {
		return []byte{}
	}
	return rangeEnd
}
