// Copyright 2023 Lack (xingyys@gmail.com).
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

package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/olive-io/olive/api"
	errs "github.com/olive-io/olive/pkg/errors"
	"github.com/olive-io/olive/server/mvcc"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.uber.org/zap"
)

type applyResult struct {
	resp proto.Message
	err  error
	// physc signals the physical effect of the request has completed in addition
	// to being logically reflected by the node. Currently only used for
	// Compaction requests.
	physc <-chan struct{}
	trace *traceutil.Trace
}

// applier is the interface for processing rafe message
type applier interface {
	Apply(r *api.InternalRaftRequest) *applyResult

	Put(ctx context.Context, txn mvcc.ITxnWrite, p *api.PutRequest) (*api.PutResponse, *traceutil.Trace, error)
	Range(ctx context.Context, txn mvcc.ITxnRead, r *api.RangeRequest) (*api.RangeResponse, error)
	DeleteRange(txn mvcc.ITxnWrite, dr *api.DeleteRangeRequest) (*api.DeleteRangeResponse, error)
	Txn(ctx context.Context, rt *api.TxnRequest) (*api.TxnResponse, *traceutil.Trace, error)
	Compaction(compaction *api.CompactionRequest) (*api.CompactionResponse, <-chan struct{}, *traceutil.Trace, error)
}

type checkReqFunc func(mvcc.IReadView, *api.RequestOp) error

type applierBackend struct {
	s *KVServer

	checkPut   checkReqFunc
	checkRange checkReqFunc
}

func (s *KVServer) newApplierBackend() applier {
	base := &applierBackend{s: s}
	base.checkPut = func(rv mvcc.IReadView, req *api.RequestOp) error {
		return base.checkRequestPut(rv, req)
	}
	base.checkRange = func(rv mvcc.IReadView, req *api.RequestOp) error {
		return base.checkRequestRange(rv, req)
	}
	return base
}

func (a *applierBackend) Apply(r *api.InternalRaftRequest) *applyResult {
	op := "unknown"
	ar := &applyResult{}
	defer func(start time.Time) {
		success := ar.err == nil || errors.Is(ar.err, mvcc.ErrCompacted)
		applySec.WithLabelValues("v1", op, strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
		warnOfExpensiveRequest(a.s.Logger(), a.s.Cfg.WarningApplyDuration, start, &api.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		if !success {
			warnOfFailedRequest(a.s.Logger(), start, &api.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		}
	}(time.Now())

	// call into a.s.applyV3.F instead of a.F so upper appliers can check individual calls
	switch {
	case r.Range != nil:
		op = "Range"
		ar.resp, ar.err = a.s.apply.Range(context.TODO(), nil, r.Range)
	case r.Put != nil:
		op = "Put"
		ar.resp, ar.trace, ar.err = a.s.apply.Put(context.TODO(), nil, r.Put)
	case r.DeleteRange != nil:
		op = "DeleteRange"
		ar.resp, ar.err = a.s.apply.DeleteRange(nil, r.DeleteRange)
	default:
		a.s.lg.Panic("not implemented apply", zap.Stringer("raft-request", r))
	}
	return ar
}

func (a *applierBackend) Put(ctx context.Context, txn mvcc.ITxnWrite, p *api.PutRequest) (resp *api.PutResponse, trace *traceutil.Trace, err error) {
	resp = &api.PutResponse{}
	resp.Header = &api.ResponseHeader{}
	trace = traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("put",
			a.s.Logger(),
			traceutil.Field{Key: "key", Value: string(p.Key)},
			traceutil.Field{Key: "req_size", Value: p.XSize()},
		)
	}
	val := p.Value
	if txn == nil {
		txn = a.s.KV().Write(trace)
		defer txn.End()
	}

	var rr *mvcc.RangeResult
	if p.IgnoreValue || p.PrevKv {
		trace.StepWithFunction(func() {
			rr, err = txn.Range(context.TODO(), p.Key, nil, mvcc.RangeOptions{})
		}, "get previous kv pair")

		if err != nil {
			return nil, nil, err
		}
	}
	if p.IgnoreValue {
		if rr == nil || len(rr.KVs) == 0 {
			// ignore_{lease,value} flag expects previous key-value pair
			return nil, nil, errs.ErrKeyNotFound
		}
	}
	if p.IgnoreValue {
		val = rr.KVs[0].Value
	}
	if p.PrevKv {
		if rr != nil && len(rr.KVs) != 0 {
			resp.PrevKv = &rr.KVs[0]
		}
	}

	resp.Header.Revision = txn.Put(p.Key, val)
	trace.AddField(traceutil.Field{Key: "response_revision", Value: resp.Header.Revision})
	return resp, trace, nil
}

func (a *applierBackend) Range(ctx context.Context, txn mvcc.ITxnRead, r *api.RangeRequest) (*api.RangeResponse, error) {
	trace := traceutil.Get(ctx)

	resp := &api.RangeResponse{}
	resp.Header = &api.ResponseHeader{}

	if txn == nil {
		txn = a.s.kv.Read(mvcc.ConcurrentReadTxMode, trace)
		defer txn.End()
	}

	limit := r.Limit
	if r.SortOrder != api.RangeRequest_NONE ||
		r.MinModRevision != 0 || r.MaxModRevision != 0 ||
		r.MinCreateRevision != 0 || r.MaxCreateRevision != 0 {
		// fetch everything; sort and truncate afterwards
		limit = 0
	}
	if limit > 0 {
		// fetch one extra for 'more' flag
		limit = limit + 1
	}

	ro := mvcc.RangeOptions{
		Limit: limit,
		Rev:   r.Revision,
		Count: r.CountOnly,
	}

	rr, err := txn.Range(ctx, r.Key, mkGteRange(r.RangeEnd), ro)
	if err != nil {
		return nil, err
	}

	if r.MaxModRevision != 0 {
		f := func(kv *api.KeyValue) bool { return kv.ModRevision > r.MaxModRevision }
		pruneKVs(rr, f)
	}
	if r.MinModRevision != 0 {
		f := func(kv *api.KeyValue) bool { return kv.ModRevision < r.MinModRevision }
		pruneKVs(rr, f)
	}
	if r.MaxCreateRevision != 0 {
		f := func(kv *api.KeyValue) bool { return kv.CreateRevision > r.MaxCreateRevision }
		pruneKVs(rr, f)
	}
	if r.MinCreateRevision != 0 {
		f := func(kv *api.KeyValue) bool { return kv.CreateRevision < r.MinCreateRevision }
		pruneKVs(rr, f)
	}

	sortOrder := r.SortOrder
	if r.SortTarget != api.RangeRequest_KEY && sortOrder == api.RangeRequest_NONE {
		// Since current mvcc.Range implementation returns results
		// sorted by keys in lexiographically ascending order,
		// sort ASCEND by default only when target is not 'KEY'
		sortOrder = api.RangeRequest_ASCEND
	}
	if sortOrder != api.RangeRequest_NONE {
		var sorter sort.Interface
		switch {
		case r.SortTarget == api.RangeRequest_KEY:
			sorter = &kvSortByKey{&kvSort{rr.KVs}}
		case r.SortTarget == api.RangeRequest_VERSION:
			sorter = &kvSortByVersion{&kvSort{rr.KVs}}
		case r.SortTarget == api.RangeRequest_CREATE:
			sorter = &kvSortByCreate{&kvSort{rr.KVs}}
		case r.SortTarget == api.RangeRequest_MOD:
			sorter = &kvSortByMod{&kvSort{rr.KVs}}
		case r.SortTarget == api.RangeRequest_VALUE:
			sorter = &kvSortByValue{&kvSort{rr.KVs}}
		}
		switch {
		case sortOrder == api.RangeRequest_ASCEND:
			sort.Sort(sorter)
		case sortOrder == api.RangeRequest_DESCEND:
			sort.Sort(sort.Reverse(sorter))
		}
	}

	if r.Limit > 0 && len(rr.KVs) > int(r.Limit) {
		rr.KVs = rr.KVs[:r.Limit]
		resp.More = true
	}
	trace.Step("filter and sort the key-value pairs")
	resp.Header.Revision = rr.Rev
	resp.Count = int64(rr.Count)
	resp.Kvs = make([]*api.KeyValue, len(rr.KVs))
	for i := range rr.KVs {
		if r.KeysOnly {
			rr.KVs[i].Value = nil
		}
		resp.Kvs[i] = &rr.KVs[i]
	}
	trace.Step("assemble the response")
	return resp, nil
}

func (a *applierBackend) DeleteRange(txn mvcc.ITxnWrite, dr *api.DeleteRangeRequest) (*api.DeleteRangeResponse, error) {
	resp := &api.DeleteRangeResponse{}
	resp.Header = &api.ResponseHeader{}
	end := mkGteRange(dr.RangeEnd)

	if txn == nil {
		txn = a.s.kv.Write(traceutil.TODO())
		defer txn.End()
	}

	if dr.PrevKv {
		rr, err := txn.Range(context.TODO(), dr.Key, end, mvcc.RangeOptions{})
		if err != nil {
			return nil, err
		}
		if rr != nil {
			resp.PrevKvs = make([]*api.KeyValue, len(rr.KVs))
			for i := range rr.KVs {
				resp.PrevKvs[i] = &rr.KVs[i]
			}
		}
	}

	resp.Deleted, resp.Header.Revision = txn.DeleteRange(dr.Key, end)
	return resp, nil
}

func (a *applierBackend) Txn(ctx context.Context, rt *api.TxnRequest) (*api.TxnResponse, *traceutil.Trace, error) {
	lg := a.s.Logger()
	trace := traceutil.Get(ctx)
	if trace.IsEmpty() {
		trace = traceutil.New("transaction", a.s.Logger())
		ctx = context.WithValue(ctx, traceutil.TraceKey, trace)
	}
	isWrite := !isTxnReadonly(rt)

	// When the transaction contains write operations, we use ReadTx instead of
	// ConcurrentReadTx to avoid extra overhead of copying buffer.
	var txn mvcc.ITxnWrite
	if isWrite && a.s.Cfg.ExperimentalTxnModeWriteWithSharedBuffer {
		txn = mvcc.NewReadOnlyTxnWrite(a.s.KV().Read(mvcc.SharedBufReadTxMode, trace))
	} else {
		txn = mvcc.NewReadOnlyTxnWrite(a.s.KV().Read(mvcc.ConcurrentReadTxMode, trace))
	}

	var txnPath []bool
	trace.StepWithFunction(
		func() {
			txnPath = compareToPath(txn, rt)
		},
		"compare",
	)

	if isWrite {
		trace.AddField(traceutil.Field{Key: "read_only", Value: false})
		if _, err := checkRequests(txn, rt, txnPath, a.checkPut); err != nil {
			txn.End()
			return nil, nil, err
		}
	}
	if _, err := checkRequests(txn, rt, txnPath, a.checkRange); err != nil {
		txn.End()
		return nil, nil, err
	}
	trace.Step("check requests")
	txnResp, _ := newTxnResp(rt, txnPath)

	// When executing mutable txn ops, etcd must hold the txn lock so
	// readers do not see any intermediate results. Since writes are
	// serialized on the raft loop, the revision in the read view will
	// be the revision of the write txn.
	if isWrite {
		txn.End()
		txn = a.s.KV().Write(trace)
	}
	_, err := a.applyTxn(ctx, txn, rt, txnPath, txnResp)
	if err != nil {
		if isWrite {
			// end txn to release locks before panic
			txn.End()
			// When txn with write operations starts it has to be successful
			// We don't have a way to recover state in case of write failure
			lg.Panic("unexpected error during txn with writes", zap.Error(err))
		} else {
			lg.Error("unexpected error during readonly txn", zap.Error(err))
		}
	}
	rev := txn.Rev()
	if len(txn.Changes()) != 0 {
		rev++
	}
	txn.End()

	txnResp.Header.Revision = rev
	trace.AddField(
		traceutil.Field{Key: "number_of_response", Value: len(txnResp.Responses)},
		traceutil.Field{Key: "response_revision", Value: txnResp.Header.Revision},
	)
	return txnResp, trace, err
}

// newTxnResp allocates a txn response for a txn request given a path.
func newTxnResp(rt *api.TxnRequest, txnPath []bool) (txnResp *api.TxnResponse, txnCount int) {
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}
	resps := make([]*api.ResponseOp, len(reqs))
	txnResp = &api.TxnResponse{
		Responses: resps,
		Succeeded: txnPath[0],
		Header:    &api.ResponseHeader{},
	}
	for i, req := range reqs {
		switch tv := req.Request.(type) {
		case *api.RequestOp_RequestRange:
			resps[i] = &api.ResponseOp{Response: &api.ResponseOp_ResponseRange{}}
		case *api.RequestOp_RequestPut:
			resps[i] = &api.ResponseOp{Response: &api.ResponseOp_ResponsePut{}}
		case *api.RequestOp_RequestDeleteRange:
			resps[i] = &api.ResponseOp{Response: &api.ResponseOp_ResponseDeleteRange{}}
		case *api.RequestOp_RequestTxn:
			resp, txns := newTxnResp(tv.RequestTxn, txnPath[1:])
			resps[i] = &api.ResponseOp{Response: &api.ResponseOp_ResponseTxn{ResponseTxn: resp}}
			txnPath = txnPath[1+txns:]
			txnCount += txns + 1
		default:
		}
	}
	return txnResp, txnCount
}

func compareToPath(rv mvcc.IReadView, rt *api.TxnRequest) []bool {
	txnPath := make([]bool, 1)
	ops := rt.Success
	if txnPath[0] = applyCompares(rv, rt.Compare); !txnPath[0] {
		ops = rt.Failure
	}
	for _, op := range ops {
		tv, ok := op.Request.(*api.RequestOp_RequestTxn)
		if !ok || tv.RequestTxn == nil {
			continue
		}
		txnPath = append(txnPath, compareToPath(rv, tv.RequestTxn)...)
	}
	return txnPath
}

func applyCompares(rv mvcc.IReadView, cmps []*api.Compare) bool {
	for _, c := range cmps {
		if !applyCompare(rv, c) {
			return false
		}
	}
	return true
}

// applyCompare applies the compare request.
// If the comparison succeeds, it returns true. Otherwise, returns false.
func applyCompare(rv mvcc.IReadView, c *api.Compare) bool {
	// TODO: possible optimizations
	// * chunk reads for large ranges to conserve memory
	// * rewrite rules for common patterns:
	//	ex. "[a, b) createrev > 0" => "limit 1 /\ kvs > 0"
	// * caching
	rr, err := rv.Range(context.TODO(), c.Key, mkGteRange(c.RangeEnd), mvcc.RangeOptions{})
	if err != nil {
		return false
	}
	if len(rr.KVs) == 0 {
		if c.Target == api.Compare_VALUE {
			// Always fail if comparing a value on a key/keys that doesn't exist;
			// nil == empty string in grpc; no way to represent missing value
			return false
		}
		return compareKV(c, api.KeyValue{})
	}
	for _, kv := range rr.KVs {
		if !compareKV(c, kv) {
			return false
		}
	}
	return true
}

func compareKV(c *api.Compare, ckv api.KeyValue) bool {
	var result int
	rev := int64(0)
	switch c.Target {
	case api.Compare_VALUE:
		v := []byte{}
		if tv, _ := c.TargetUnion.(*api.Compare_Value); tv != nil {
			v = tv.Value
		}
		result = bytes.Compare(ckv.Value, v)
	case api.Compare_CREATE:
		if tv, _ := c.TargetUnion.(*api.Compare_CreateRevision); tv != nil {
			rev = tv.CreateRevision
		}
		result = compareInt64(ckv.CreateRevision, rev)
	case api.Compare_MOD:
		if tv, _ := c.TargetUnion.(*api.Compare_ModRevision); tv != nil {
			rev = tv.ModRevision
		}
		result = compareInt64(ckv.ModRevision, rev)
	case api.Compare_VERSION:
		if tv, _ := c.TargetUnion.(*api.Compare_Version); tv != nil {
			rev = tv.Version
		}
		result = compareInt64(ckv.Version, rev)
	}
	switch c.Result {
	case api.Compare_EQUAL:
		return result == 0
	case api.Compare_NOT_EQUAL:
		return result != 0
	case api.Compare_GREATER:
		return result > 0
	case api.Compare_LESS:
		return result < 0
	}
	return true
}

func (a *applierBackend) applyTxn(ctx context.Context, txn mvcc.ITxnWrite, rt *api.TxnRequest, txnPath []bool, tresp *api.TxnResponse) (txns int, err error) {
	trace := traceutil.Get(ctx)
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}

	for i, req := range reqs {
		respi := tresp.Responses[i].Response
		switch tv := req.Request.(type) {
		case *api.RequestOp_RequestRange:
			trace.StartSubTrace(
				traceutil.Field{Key: "req_type", Value: "range"},
				traceutil.Field{Key: "range_begin", Value: string(tv.RequestRange.Key)},
				traceutil.Field{Key: "range_end", Value: string(tv.RequestRange.RangeEnd)})
			resp, err := a.Range(ctx, txn, tv.RequestRange)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed Range: %w", err)
			}
			respi.(*api.ResponseOp_ResponseRange).ResponseRange = resp
			trace.StopSubTrace()
		case *api.RequestOp_RequestPut:
			trace.StartSubTrace(
				traceutil.Field{Key: "req_type", Value: "put"},
				traceutil.Field{Key: "key", Value: string(tv.RequestPut.Key)},
				traceutil.Field{Key: "req_size", Value: tv.RequestPut.XSize()})
			resp, _, err := a.Put(ctx, txn, tv.RequestPut)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed Put: %w", err)
			}
			respi.(*api.ResponseOp_ResponsePut).ResponsePut = resp
			trace.StopSubTrace()
		case *api.RequestOp_RequestDeleteRange:
			resp, err := a.DeleteRange(txn, tv.RequestDeleteRange)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed DeleteRange: %w", err)
			}
			respi.(*api.ResponseOp_ResponseDeleteRange).ResponseDeleteRange = resp
		case *api.RequestOp_RequestTxn:
			resp := respi.(*api.ResponseOp_ResponseTxn).ResponseTxn
			applyTxns, err := a.applyTxn(ctx, txn, tv.RequestTxn, txnPath[1:], resp)
			if err != nil {
				// don't wrap the error. It's a recursive call and err should be already wrapped
				return 0, err
			}
			txns += applyTxns + 1
			txnPath = txnPath[applyTxns+1:]
		default:
			// empty union
		}
	}
	return txns, nil
}

func (a *applierBackend) Compaction(compaction *api.CompactionRequest) (*api.CompactionResponse, <-chan struct{}, *traceutil.Trace, error) {
	resp := &api.CompactionResponse{}
	resp.Header = &api.ResponseHeader{}
	trace := traceutil.New("compact",
		a.s.Logger(),
		traceutil.Field{Key: "revision", Value: compaction.Revision},
	)

	ch, err := a.s.KV().Compact(trace, compaction.Revision)
	if err != nil {
		return nil, ch, nil, err
	}
	// get the current revision. which key to get is not important.
	rr, _ := a.s.KV().Range(context.TODO(), []byte("compaction"), nil, mvcc.RangeOptions{})
	resp.Header.Revision = rr.Rev
	return resp, ch, trace, err
}

type kvSort struct{ kvs []api.KeyValue }

func (s *kvSort) Swap(i, j int) {
	t := s.kvs[i]
	s.kvs[i] = s.kvs[j]
	s.kvs[j] = t
}
func (s *kvSort) Len() int { return len(s.kvs) }

type kvSortByKey struct{ *kvSort }

func (s *kvSortByKey) Less(i, j int) bool {
	return bytes.Compare(s.kvs[i].Key, s.kvs[j].Key) < 0
}

type kvSortByVersion struct{ *kvSort }

func (s *kvSortByVersion) Less(i, j int) bool {
	return (s.kvs[i].Version - s.kvs[j].Version) < 0
}

type kvSortByCreate struct{ *kvSort }

func (s *kvSortByCreate) Less(i, j int) bool {
	return (s.kvs[i].CreateRevision - s.kvs[j].CreateRevision) < 0
}

type kvSortByMod struct{ *kvSort }

func (s *kvSortByMod) Less(i, j int) bool {
	return (s.kvs[i].ModRevision - s.kvs[j].ModRevision) < 0
}

type kvSortByValue struct{ *kvSort }

func (s *kvSortByValue) Less(i, j int) bool {
	return bytes.Compare(s.kvs[i].Value, s.kvs[j].Value) < 0
}

func checkRequests(rv mvcc.IReadView, rt *api.TxnRequest, txnPath []bool, f checkReqFunc) (int, error) {
	txnCount := 0
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}
	for _, req := range reqs {
		if tv, ok := req.Request.(*api.RequestOp_RequestTxn); ok && tv.RequestTxn != nil {
			txns, err := checkRequests(rv, tv.RequestTxn, txnPath[1:], f)
			if err != nil {
				return 0, err
			}
			txnCount += txns + 1
			txnPath = txnPath[txns+1:]
			continue
		}
		if err := f(rv, req); err != nil {
			return 0, err
		}
	}
	return txnCount, nil
}

func (a *applierBackend) checkRequestPut(rv mvcc.IReadView, reqOp *api.RequestOp) error {
	tv, ok := reqOp.Request.(*api.RequestOp_RequestPut)
	if !ok || tv.RequestPut == nil {
		return nil
	}
	req := tv.RequestPut
	if req.IgnoreValue {
		// expects previous key-value, error if not exist
		rr, err := rv.Range(context.TODO(), req.Key, nil, mvcc.RangeOptions{})
		if err != nil {
			return err
		}
		if rr == nil || len(rr.KVs) == 0 {
			return errs.ErrKeyNotFound
		}
	}
	return nil
}

func (a *applierBackend) checkRequestRange(rv mvcc.IReadView, reqOp *api.RequestOp) error {
	tv, ok := reqOp.Request.(*api.RequestOp_RequestRange)
	if !ok || tv.RequestRange == nil {
		return nil
	}
	req := tv.RequestRange
	switch {
	case req.Revision == 0:
		return nil
	case req.Revision > rv.Rev():
		return mvcc.ErrFutureRev
	case req.Revision < rv.FirstRev():
		return mvcc.ErrCompacted
	}
	return nil
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
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

func noSideEffect(r *api.InternalRaftRequest) bool {
	return r.Range != nil
}

func pruneKVs(rr *mvcc.RangeResult, isPrunable func(*api.KeyValue) bool) {
	j := 0
	for i := range rr.KVs {
		rr.KVs[j] = rr.KVs[i]
		if !isPrunable(&rr.KVs[i]) {
			j++
		}
	}
	rr.KVs = rr.KVs[:j]
}
