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
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/olive-io/olive/api/olivepb"
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
	Apply(r *pb.RaftInternalRequest) *applyResult
	Range(ctx context.Context, r *pb.RegionRangeRequest) (*pb.RegionRangeResponse, error)
	Put(ctx context.Context, r *pb.RegionPutRequest) (*pb.RegionPutResponse, *traceutil.Trace, error)
	Delete(ctx context.Context, r *pb.RegionDeleteRequest) (*pb.RegionDeleteResponse, *traceutil.Trace, error)
}

type applier struct {
	r *Region
}

func (r *Region) newApplier() *applier {
	return &applier{r: r}
}

func (a *applier) Apply(r *pb.RaftInternalRequest) *applyResult {
	op := "unknown"
	ar := &applyResult{}
	defer func(start time.Time) {
		//success := ar.err == nil || ar.err == mvcc.ErrCompacted
		success := ar.err == nil
		a.r.metric.applySec.WithLabelValues(v1Version, op, strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
		warnOfExpensiveRequest(a.r.lg, a.r.metric.slowApplies, a.r.warningApplyDuration, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		if !success {
			warnOfFailedRequest(a.r.lg, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		}
	}(time.Now())

	switch {
	case r.Range != nil:
		op = "Range"
		ar.resp, ar.err = a.Range(context.TODO(), r.Range)
	case r.Put != nil:
		op = "Put"
		ar.resp, ar.trace, ar.err = a.Put(context.TODO(), r.Put)
	case r.Delete != nil:
		op = "Delete"
		ar.resp, ar.trace, ar.err = a.Delete(context.TODO(), r.Delete)
	case r.DeployDefinition != nil:
		op = "DeployDefinition"
	case r.ExecuteDefinition != nil:
		op = "ExecuteDefinition"
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

func (a *applier) Put(ctx context.Context, r *pb.RegionPutRequest) (resp *pb.RegionPutResponse, trace *traceutil.Trace, err error) {
	resp = &pb.RegionPutResponse{}
	resp.Header = &pb.RaftResponseHeader{}
	trace = traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("put",
			a.r.lg,
			traceutil.Field{Key: "key", Value: string(r.Key)},
			traceutil.Field{Key: "req_size", Value: r.XSize()},
		)
	}

	err = a.r.put(r.Key, r.Value, r.IsSync)
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
			traceutil.Field{Key: "key", Value: string(r.Key)},
		)
	}

	err := a.r.del(r.Key, r.IsSync)
	if err != nil {
		return nil, trace, err
	}
	return resp, trace, nil
}

// mkGteRange determines if the range end is a >= range. This works around grpc
// sending empty byte strings as nil; >= is encoded in the range end as '\0'.
// If it is a GTE range, then []byte{} is returned to indicate the empty byte
// string (vs nil being no byte string).
func mkGteRange(rangeEnd []byte) []byte {
	if len(rangeEnd) == 0 {
		return defaultEndKey
	}
	if len(rangeEnd) == 1 && rangeEnd[0] == 0 {
		return []byte{}
	}
	return rangeEnd
}
