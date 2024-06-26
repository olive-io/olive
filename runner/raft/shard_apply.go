/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package raft

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/gogo/protobuf/proto"
	"go.etcd.io/etcd/pkg/v3/traceutil"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/pkg/bytesutil"
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
	Range(ctx context.Context, r *pb.ShardRangeRequest) (*pb.ShardRangeResponse, error)
	Put(ctx context.Context, r *pb.ShardPutRequest) (*pb.ShardPutResponse, *traceutil.Trace, error)
	Delete(ctx context.Context, r *pb.ShardDeleteRequest) (*pb.ShardDeleteResponse, *traceutil.Trace, error)

	DeployDefinition(ctx context.Context, r *pb.ShardDeployDefinitionRequest) (*pb.ShardDeployDefinitionResponse, *traceutil.Trace, error)
	RunBpmnProcess(ctx context.Context, r *pb.ShardRunBpmnProcessRequest) (*pb.ShardRunBpmnProcessResponse, *traceutil.Trace, error)
}

type applier struct {
	r *Shard
}

func (s *Shard) newApplier() *applier {
	return &applier{r: s}
}

func (a *applier) Apply(ctx context.Context, r *pb.RaftInternalRequest) *applyResult {
	op := "unknown"
	ar := &applyResult{}
	defer func(start time.Time) {
		//success := ar.err == nil || ar.err == mvcc.ErrCompacted
		success := ar.err == nil
		a.r.metric.applySec.WithLabelValues(v1Version, op, strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
		warnOfExpensiveRequest(a.r.lg, a.r.metric.slowApplies, a.r.cfg.WarningApplyDuration, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
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
	case r.RunBpmnProcess != nil:
		op = "RunBpmnProcess"
		ar.resp, ar.trace, ar.err = a.RunBpmnProcess(ctx, r.RunBpmnProcess)
	}

	return ar
}

func (a *applier) Range(ctx context.Context, r *pb.ShardRangeRequest) (*pb.ShardRangeResponse, error) {
	trace := traceutil.Get(ctx)

	limit := r.Limit
	kvs, err := a.r.getRange(r.Key, mkGteRange(r.RangeEnd), limit)
	if err != nil {
		return nil, err
	}

	resp := &pb.ShardRangeResponse{
		Header: a.toHeader(),
		Kvs:    kvs,
	}
	trace.Step("assemble the response")
	return resp, nil
}

func (a *applier) Put(ctx context.Context, r *pb.ShardPutRequest) (*pb.ShardPutResponse, *traceutil.Trace, error) {
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("put",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "key", Value: string(r.Key)},
			traceutil.Field{Key: "req_size", Value: r.XXX_Size()},
		)
	}

	err := a.r.put(r.Key, r.Value, r.IsSync)
	if err != nil {
		return nil, nil, err
	}

	resp := &pb.ShardPutResponse{
		Header: a.toHeader(),
	}

	return resp, trace, nil
}

func (a *applier) Delete(ctx context.Context, r *pb.ShardDeleteRequest) (*pb.ShardDeleteResponse, *traceutil.Trace, error) {
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

	resp := &pb.ShardDeleteResponse{
		Header: a.toHeader(),
	}

	return resp, trace, nil
}

func (a *applier) DeployDefinition(ctx context.Context, r *pb.ShardDeployDefinitionRequest) (*pb.ShardDeployDefinitionResponse, *traceutil.Trace, error) {
	definition := r.Definition
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("deploy_definition",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "namespace", Value: definition.Namespace},
			traceutil.Field{Key: "name", Value: definition.Name},
			traceutil.Field{Key: "version", Value: definition.Spec.Version},
		)
	}

	prefix := bytesutil.PathJoin(definitionPrefix, []byte(definition.Namespace), []byte(definition.Name))
	key := bytesutil.PathJoin(prefix, []byte(fmt.Sprintf("%d", definition.Spec.Version)))

	kvs, _ := a.r.getRange(prefix, getPrefix(prefix), 0)
	if len(kvs) == 0 {
		a.r.metric.definition.Add(1)
	}

	trace.Step("save definition", traceutil.Field{Key: "key", Value: key})
	data, _ := definition.Marshal()
	if err := a.r.put(key, data, true); err != nil {
		return nil, trace, err
	}

	resp := &pb.ShardDeployDefinitionResponse{
		Header:     a.toHeader(),
		Definition: definition,
	}

	return resp, trace, nil
}

func (a *applier) RunBpmnProcess(ctx context.Context, r *pb.ShardRunBpmnProcessRequest) (*pb.ShardRunBpmnProcessResponse, *traceutil.Trace, error) {
	process := r.Process
	trace := traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("run_bpmn_process",
			a.r.lg,
			traceutil.Field{Key: "region", Value: a.r.getID()},
			traceutil.Field{Key: "process", Value: process.Name},
			traceutil.Field{Key: "definition", Value: process.Spec.Definition},
			traceutil.Field{Key: "definition_version", Value: process.Spec.Version},
		)
	}

	definitionKey := bytesutil.PathJoin(definitionPrefix,
		[]byte(process.Namespace),
		[]byte(process.Spec.Definition),
		[]byte(fmt.Sprintf("%d", process.Spec.Version)))
	kv, err := a.r.get(definitionKey)
	if err != nil {
		return nil, trace, errors.Wrapf(err, "Definition not found")
	}
	definition := new(corev1.Definition)
	if err = definition.Unmarshal(kv.Value); err != nil {
		return nil, trace, errors.Wrapf(err, "Definition unmarshal failed")
	}

	prefix := bytesutil.PathJoin(processPrefix,
		[]byte(process.Namespace),
		[]byte(process.Spec.Definition),
		[]byte(fmt.Sprintf("%d", process.Spec.Version)))
	key := bytesutil.PathJoin(prefix, []byte(process.Name))
	if kv, _ = a.r.get(key); kv != nil {
		//return nil, trace, ErrProcessExecuted
	}

	stat := &corev1.ProcessStat{
		Spec: corev1.ProcessStatSpec{
			DefinitionName:    definition.Name,
			DefinitionVersion: definition.Spec.Version,

			ProcessName:   process.Name,
			BpmnProcessId: process.Spec.BpmnProcess,
			InitContext:   corev1.ProcessContext{},
		},
		Status: corev1.ProcessStatStatus{
			DefinitionContent: definition.Spec.Content,
			FlowNodes:         []*corev1.FlowNodeStat{},
			Phase:             corev1.ProcessPrepare,
		},
	}
	stat.Name = process.Name
	stat.Namespace = process.Namespace

	stat.Spec.InitContext.Headers = process.Spec.Headers
	stat.Spec.InitContext.Properties = make(map[string]string)
	stat.Spec.InitContext.DataObjects = process.Spec.DataObjects
	for name, value := range process.Spec.Properties {
		stat.Spec.InitContext.Properties[name] = value.String()
	}

	stat.Status.Context = *stat.Spec.InitContext.DeepCopy()

	trace.Step("save process stat", traceutil.Field{Key: "key", Value: key})
	data, _ := stat.Marshal()
	if err := a.r.put(key, data, true); err != nil {
		return nil, trace, err
	}

	resp := &pb.ShardRunBpmnProcessResponse{
		Header: a.toHeader(),
		Stat:   stat,
	}
	a.r.processQ.Set(NewProcessInfo(stat))

	return resp, trace, nil
}

func (a *applier) toHeader() *pb.RaftResponseHeader {
	return &pb.RaftResponseHeader{
		RegionId:  a.r.getID(),
		ReplicaId: a.r.getMember(),
		RaftTerm:  a.r.getTerm(),
	}
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
