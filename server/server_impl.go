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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v4"
	"github.com/olive-io/olive/api"
	errs "github.com/olive-io/olive/pkg/errors"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.uber.org/zap"
)

const (
	// In the health case, there might be a small gap (10s of entries) between
	// the applied index and committed index.
	// However, if the committed entries are very heavy to apply, the gap might grow.
	// We should stop accepting new proposals if the gap growing to a certain point.
	maxGapBetweenApplyAndCommitIndex = 5000
	traceThreshold                   = 100 * time.Millisecond

	// The timeout for the node to catch up its applied index, and is used in
	// lease related operations, such as LeaseRenew and LeaseTimeToLive.
	applyTimeout = time.Second
)

//func (s *KVServer) DeployDefinition(ctx context.Context, req *api.DeployDefinitionRequest) (resp *api.DeployDefinitionResponse, err error) {
//	definitions := &api.Definition{
//		Id:      req.Id,
//		Name:    req.Name,
//		Version: 0,
//		Content: req.Content,
//	}
//
//	data, _ := definitions.Marshal()
//
//	key := path.Join("/definitions", req.Id)
//	putReq := &api.PutRequest{
//		Key:         []byte(key),
//		Value:       data,
//		PrevKv:      false,
//		IgnoreValue: false,
//	}
//
//	var rsp *api.PutResponse
//	rsp, err = s.Put(ctx, 0, putReq)
//	if err != nil {
//		return
//	}
//
//	resp.Version = rsp.Header.Revision
//	return
//}
//
//func (s *KVServer) ListDefinition(ctx context.Context, req *api.ListDefinitionRequest) (resp *api.ListDefinitionResponse, err error) {
//	key := path.Join("/definitions")
//	rangeReq := &api.RangeRequest{
//		Key:          []byte(key),
//		Serializable: true,
//	}
//
//	var rsp *api.RangeResponse
//	rsp, err = s.Range(ctx, 0, rangeReq)
//	if err != nil {
//		return
//	}
//
//	resp.Definition = make([]*api.Definition, 0)
//	for _, kv := range rsp.Kvs {
//		definitions := &api.Definition{}
//		if err = definitions.Unmarshal(kv.Value); err == nil {
//			resp.Definition = append(resp.Definition, definitions)
//		}
//	}
//
//	return
//}
//
//func (s *KVServer) GetDefinition(ctx context.Context, req *api.GetDefinitionRequest) (*api.GetDefinitionResponse, error) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (s *KVServer) RemoveDefinition(ctx context.Context, req *api.RemoveDefinitionRequest) (resp *api.RemoveDefinitionResponse, err error) {
//	key := path.Join("/definitions", req.Id)
//	deleteReq := &api.DeleteRangeRequest{
//		Key: []byte(key),
//	}
//
//	var rsp *api.DeleteRangeResponse
//	rsp, err = s.DeleteRange(ctx, 0, deleteReq)
//	if err != nil {
//		return
//	}
//
//	_ = rsp
//	return
//}
//
//func (s *KVServer) ExecuteDefinition(ctx context.Context, req *api.ExecuteDefinitionRequest) (*api.ExecuteDefinitionResponse, error) {
//	//TODO implement me
//	panic("implement me")
//}

func (s *KVServer) Range(ctx context.Context, shardID uint64, r *api.RangeRequest) (*api.RangeResponse, error) {
	_, exists := s.getShard(shardID)
	if !exists {
		return nil, errs.ErrShardNotFound
	}

	trace := traceutil.New("range",
		s.Logger(),
		traceutil.Field{Key: "range_begin", Value: string(r.Key)},
		traceutil.Field{Key: "range_end", Value: string(r.RangeEnd)},
		traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
	)
	ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

	var resp *api.RangeResponse
	var err error
	defer func(start time.Time) {
		warnOfExpensiveReadOnlyRangeRequest(s.Logger(), s.Cfg.WarningApplyDuration, start, r, resp, err)
		if resp != nil {
			trace.AddField(
				traceutil.Field{Key: "response_count", Value: len(resp.Kvs)},
				traceutil.Field{Key: "response_revision", Value: resp.Header.Revision},
				traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
			)
		}
		trace.LogIfLong(traceThreshold)
	}(time.Now())

	var get func()

	if !r.Serializable {
		get = func() {
			var result any
			var ok bool
			result, err = s.nh.StaleRead(shardID, r)
			resp, ok = result.(*api.RangeResponse)
			if !ok {
				s.Logger().Panic("not match raft read", zap.Stringer("request", r))
			}
		}
		trace.Step("agreement among raft nodes before linearized reading")
	} else {
		get = func() {
			var result any
			var ok bool
			result, err = s.nh.SyncRead(ctx, shardID, r)
			resp, ok = result.(*api.RangeResponse)
			if !ok {
				s.Logger().Panic("not match raft read", zap.Stringer("request", r))
			}
		}
	}
	//chk := func(ai *auth.AuthInfo) error {
	//	return s.authStore.IsRangePermitted(ai, r.Key, r.RangeEnd)
	//}

	if serr := s.doSerialize(ctx /*,chk*/, get); serr != nil {
		err = serr
		return nil, err
	}
	return resp, err
}

func (s *KVServer) Put(ctx context.Context, shardID uint64, r *api.PutRequest) (*api.PutResponse, error) {
	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := s.raftRequest(ctx, shardID, api.InternalRaftRequest{Put: r})
	if err != nil {
		return nil, err
	}
	return resp.(*api.PutResponse), nil
}

func (s *KVServer) DeleteRange(ctx context.Context, shardID uint64, r *api.DeleteRangeRequest) (*api.DeleteRangeResponse, error) {
	resp, err := s.raftRequest(ctx, shardID, api.InternalRaftRequest{DeleteRange: r})
	if err != nil {
		return nil, err
	}
	return resp.(*api.DeleteRangeResponse), nil
}

func (s *KVServer) Txn(ctx context.Context, shardID uint64, r *api.TxnRequest) (*api.TxnResponse, error) {
	if isTxnReadonly(r) {
		trace := traceutil.New("transaction",
			s.Logger(),
			traceutil.Field{Key: "read_only", Value: true},
			traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
		)
		ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

		var get func()
		var resp *api.TxnResponse
		var err error

		if !isTxnSerializable(r) {
			var result any
			var ok bool
			result, err = s.nh.StaleRead(shardID, r)
			resp, ok = result.(*api.TxnResponse)
			if !ok {
				s.Logger().Panic("not match raft read", zap.Stringer("request", r))
			}
			trace.Step("agreement among raft nodes before linearized reading")

		} else {
			get = func() {
				var result any
				var ok bool
				result, err = s.nh.SyncRead(ctx, shardID, r)
				resp, ok = result.(*api.TxnResponse)
				if !ok {
					s.Logger().Panic("not match raft read", zap.Stringer("request", r))
				}
			}
		}
		//chk := func(ai *auth.AuthInfo) error {
		//	return checkTxnAuth(s.authStore, ai, r)
		//}

		defer func(start time.Time) {
			warnOfExpensiveReadOnlyTxnRequest(s.Logger(), s.Cfg.WarningApplyDuration, start, r, resp, err)
			trace.LogIfLong(traceThreshold)
		}(time.Now())

		if serr := s.doSerialize(ctx /*, chk*/, get); serr != nil {
			return nil, serr
		}
		return resp, err
	}

	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := s.raftRequest(ctx, shardID, api.InternalRaftRequest{Txn: r})
	if err != nil {
		return nil, err
	}
	return resp.(*api.TxnResponse), nil
}

func isTxnSerializable(rt *api.TxnRequest) bool {
	for _, u := range rt.Success {
		if r := u.GetRequestRange(); r == nil || !r.Serializable {
			return false
		}
	}
	for _, u := range rt.Failure {
		if r := u.GetRequestRange(); r == nil || !r.Serializable {
			return false
		}
	}
	return true
}

func isTxnReadonly(tr *api.TxnRequest) bool {
	for _, u := range tr.Success {
		if r := u.GetRequestRange(); r == nil {
			return false
		}
	}
	for _, u := range tr.Failure {
		if r := u.GetRequestRange(); r == nil {
			return false
		}
	}
	return true
}

func (s *KVServer) Compact(ctx context.Context, shardID uint64, r *api.CompactionRequest) (*api.CompactionResponse, error) {
	startTime := time.Now()
	result, err := s.processInternalRaftRequestOnce(ctx, shardID, api.InternalRaftRequest{Compaction: r})
	trace := traceutil.TODO()
	if result != nil && result.trace != nil {
		trace = result.trace
		defer func() {
			trace.LogIfLong(traceThreshold)
		}()
		applyStart := result.trace.GetStartTime()
		result.trace.SetStartTime(startTime)
		trace.InsertStep(0, applyStart, "process raft request")
	}
	if r.Physical && result != nil && result.physc != nil {
		<-result.physc
		// The compaction is done deleting keys; the hash is now settled
		// but the data is not necessarily committed. If there's a crash,
		// the hash may revert to a hash prior to compaction completing
		// if the compaction resumes. Force the finished compaction to
		// commit so it won't resume following a crash.
		s.be.ForceCommit()
		trace.Step("physically apply compaction")
	}
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	resp := result.resp.(*api.CompactionResponse)
	if resp == nil {
		resp = &api.CompactionResponse{}
	}
	if resp.Header == nil {
		resp.Header = &api.ResponseHeader{}
	}
	resp.Header.Revision = s.kv.Rev()
	trace.AddField(traceutil.Field{Key: "response_revision", Value: resp.Header.Revision})
	trace.AddField(traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)})
	return resp, nil
}

func (s *KVServer) raftRequestOnce(ctx context.Context, shardID uint64, r api.InternalRaftRequest) (proto.Message, error) {
	result, err := s.processInternalRaftRequestOnce(ctx, shardID, r)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	if startTime, ok := ctx.Value(traceutil.StartTimeKey).(time.Time); ok && result.trace != nil {
		applyStart := result.trace.GetStartTime()
		// The trace object is created in apply. Here reset the start time to trace
		// the raft request time by the difference between the request start time
		// and apply start time
		result.trace.SetStartTime(startTime)
		result.trace.InsertStep(0, applyStart, "process raft request")
		result.trace.LogIfLong(traceThreshold)
	}
	return result.resp, nil
}

func (s *KVServer) raftRequest(ctx context.Context, shardID uint64, r api.InternalRaftRequest) (proto.Message, error) {
	return s.raftRequestOnce(ctx, shardID, r)
}

// doSerialize handles the auth logic, with permissions checked by "chk", for a serialized request "get". Returns a non-nil error on authentication failure.
func (s *KVServer) doSerialize(
	ctx context.Context,
	//chk func(*auth.AuthInfo) error,
	get func(),
) error {
	trace := traceutil.Get(ctx)
	//ai, err := s.AuthInfoFromCtx(ctx)
	//if err != nil {
	//	return err
	//}
	//if ai == nil {
	//	// chk expects non-nil AuthInfo; use empty credentials
	//	ai = &auth.AuthInfo{}
	//}
	//if err = chk(ai); err != nil {
	//	return err
	//}
	trace.Step("get authentication metadata")
	// fetch response for serialized request
	get()
	// check for stale token revision in case the auth store was updated while
	// the request has been handled.
	//if ai.Revision != 0 && ai.Revision != s.authStore.Revision() {
	//	return auth.ErrAuthOldRevision
	//}
	return nil
}

func (s *KVServer) processInternalRaftRequestOnce(ctx context.Context, shardID uint64, r api.InternalRaftRequest) (*applyResult, error) {
	ssm, exists := s.getShard(shardID)
	if !exists {
		return nil, errs.ErrShardNotFound
	}

	ai := ssm.getAppliedIndex()
	ci := ssm.getCommittedIndex()
	if ci > ai+maxGapBetweenApplyAndCommitIndex {
		return nil, errs.ErrTooManyRequests
	}

	r.Header = &api.RequestHeader{
		ID: ssm.reqIDGen.Next(),
	}

	// check authinfo if it is not InternalAuthenticateRequest
	//if r.Authenticate == nil {
	//	authInfo, err := s.AuthInfoFromCtx(ctx)
	//	if err != nil {
	//		return nil, err
	//	}
	//	if authInfo != nil {
	//		r.Header.Username = authInfo.Username
	//		r.Header.AuthRevision = authInfo.Revision
	//	}
	//}

	data, err := r.Marshal()
	if err != nil {
		return nil, err
	}

	if len(data) > int(s.Cfg.MaxRequestBytes) {
		return nil, errs.ErrRequestTooLarge
	}

	id := r.Header.ID
	ch := s.w.Register(id)

	cctx, cancel := context.WithTimeout(ctx, s.Cfg.ReqTimeout())
	defer cancel()

	start := time.Now()

	session := s.nh.GetNoOPSession(shardID)
	_, err = s.nh.SyncPropose(cctx, session, data)
	if err != nil {
		proposalsFailed.Inc()
		s.w.Trigger(id, nil) // GC wait
		return nil, err
	}
	proposalsPending.Inc()
	defer proposalsPending.Dec()

	select {
	case x := <-ch:
		return x.(*applyResult), nil
	case <-cctx.Done():
		proposalsFailed.Inc()
		s.w.Trigger(id, nil) // GC wait
		return nil, s.parseProposeCtxErr(cctx.Err(), start)
	case <-s.done:
		return nil, errs.ErrStopped
	}
}

func isStopped(err error) bool {
	return errors.Is(err, dragonboat.ErrClosed) || errors.Is(err, errs.ErrStopped)
}
