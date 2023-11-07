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
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	sm "github.com/lni/dragonboat/v4/statemachine"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/pkg/bytesutil"
	errs "github.com/olive-io/olive/pkg/errors"
	"github.com/olive-io/olive/server/cindex"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

var (
	lastAppliedIndex = []byte("last_applied_index")
)

type shard struct {
	lg *zap.Logger

	shardID uint64
	nodeID  uint64

	ctx    context.Context
	cancel context.CancelFunc

	apply applier
	be    backend.IBackend
	w     wait.Wait

	consistIndex cindex.IConsistentIndexer
	reqIDGen     *idutil.Generator

	stopping chan struct{}
	done     chan struct{}
	changec  chan struct{}

	appliedIndex   uint64 // must use atomic operations to access; keep 64-bit aligned.
	committedIndex uint64 // must use atomic operations to access; keep 64-bit aligned.
	term           uint64 // must use atomic operations to access; keep 64-bit aligned.
	lead           uint64 // must use atomic operations to access; keep 64-bit aligned.
}

func (s *KVServer) NewDiskKV(shardID, nodeID uint64) sm.IOnDiskStateMachine {

	ci := cindex.NewConsistentIndex(s.Backend(), shardID, nodeID)
	cindex.CreateMetaBucket(s.Backend().BatchTx())

	ctx, cancel := context.WithCancel(s.ctx)
	ss := &shard{
		lg: s.Logger(),

		shardID: shardID,
		nodeID:  nodeID,

		ctx:    ctx,
		cancel: cancel,

		apply: s.apply,
		be:    s.Backend(),
		w:     s.w,

		consistIndex: ci,

		reqIDGen: idutil.NewGenerator(uint16(nodeID), time.Now()),

		stopping: make(chan struct{}, 1),
		done:     make(chan struct{}),
		changec:  make(chan struct{}, 5),
	}

	s.smu.Lock()
	s.sms[shardID] = ss
	s.smu.Unlock()

	s.done = make(chan struct{}, 1)

	go s.run()

	go func() {
		select {
		case <-ss.stopping:
			s.smu.Lock()
			delete(s.sms, shardID)
			s.smu.Unlock()
		}
	}()

	return ss
}

func (s *shard) run() {
	defer close(s.done)
	for {
		select {
		case <-s.stopping:
			return
		case <-s.changec:
			// TODO: trigger raft group change
		default:

		}
	}
}

func (s *shard) Open(done <-chan struct{}) (uint64, error) {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	r := &pb.RangeRequest{
		Key:   bytesutil.PathJoin(s.prefix(), lastAppliedIndex),
		Limit: 1,
	}
	rsp, err := s.apply.Range(ctx, nil, r)
	if err != nil {
		return 0, err
	}
	if len(rsp.Kvs) == 0 {
		return 0, err
	}

	index := binary.LittleEndian.Uint64(rsp.Kvs[0].Value)
	s.setAppliedIndex(index)
	return index, nil
}

func (s *shard) Update(entries []sm.Entry) ([]sm.Entry, error) {
	if entries[0].Index < s.getAppliedIndex() {
		return entries, nil
	}

	last := 0
	for i := range entries {
		entry := entries[i]
		s.applyEntryNormal(&entry)
		s.setAppliedIndex(entry.Index)
		entry.Result = sm.Result{Value: uint64(len(entry.Cmd))}
		last = i
	}

	lastIndex := entries[last].Index
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	r := &pb.PutRequest{Key: bytesutil.PathJoin(s.prefix(), lastAppliedIndex)}
	r.Value = make([]byte, 8)
	binary.LittleEndian.PutUint64(r.Value, lastIndex)
	_, _, err := s.apply.Put(ctx, nil, r)
	if err != nil {
		return entries, err
	}

	return entries, nil
}

func (s *shard) Lookup(query interface{}) (interface{}, error) {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	r := query.(*pb.RangeRequest)
	rsp, err := s.apply.Range(ctx, nil, r)
	if err != nil {
		return nil, err
	}

	return rsp, nil
}

func (s *shard) Sync() error {
	return nil
}

func (s *shard) PrepareSnapshot() (interface{}, error) {
	snapshot := s.be.Snapshot()
	return snapshot, nil
}

func (s *shard) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	snapshot := ctx.(backend.ISnapshot)
	_, err := snapshot.WriteTo(s.prefix(), writer)
	if err != nil {
		return err
	}

	return nil
}

func (s *shard) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	err := s.be.Recover(reader)
	if err != nil {
		return err
	}

	return nil
}

func (s *shard) prefix() []byte {
	return []byte(fmt.Sprintf("/%d/%d", s.shardID, s.nodeID))
}

func (s *shard) parseProposeCtxErr(err error, start time.Time) error {
	switch err {
	case context.Canceled:
		return errs.ErrCanceled

	case context.DeadlineExceeded:
		//s.leadTimeMu.RLock()
		//curLeadElected := s.leadElectedTime
		//s.leadTimeMu.RUnlock()
		//prevLeadLost := curLeadElected.Add(-2 * time.Duration(s.Cfg.ElectionTicks) * time.Duration(s.Cfg.TickMs) * time.Millisecond)
		//if start.After(prevLeadLost) && start.Before(curLeadElected) {
		//	return ErrTimeoutDueToLeaderFail
		//}
		//lead := types.ID(s.getLead())
		//switch lead {
		//case types.ID(raft.None):
		//	// TODO: return error to specify it happens because the cluster does not have leader now
		//case s.ID():
		//	if !isConnectedToQuorumSince(s.r.transport, start, s.ID(), s.cluster.Members()) {
		//		return errs.ErrTimeoutDueToConnectionLost
		//	}
		//default:
		//	if !isConnectedSince(s.r.transport, start, lead) {
		//		return errs.ErrTimeoutDueToConnectionLost
		//	}
		//}
		return errs.ErrTimeout

	default:
		return err
	}
}

// applyEntryNormal apples an EntryNormal type raftpb request to the KVServer
func (s *shard) applyEntryNormal(ent *sm.Entry) {
	var ar *applyResult
	index := s.consistIndex.ConsistentIndex()
	if ent.Index > index {
		// set the consistent index of current executing entry
		s.consistIndex.SetConsistentApplyingIndex(ent.Index, s.term)
		defer func() {
			// The txPostLockInsideApplyHook will not get called in some cases,
			// in which we should move the consistent index forward directly.
			newIndex := s.consistIndex.ConsistentIndex()
			if newIndex < ent.Index {
				s.consistIndex.SetConsistentIndex(ent.Index, s.term)
			}
		}()
	}
	s.lg.Debug("apply entry normal",
		zap.Uint64("consistent-index", index),
		zap.Uint64("entry-term", s.term),
		zap.Uint64("entry-index", ent.Index))

	raftReq := pb.InternalRaftRequest{}
	if err := raftReq.Unmarshal(ent.Cmd); err != nil {
		s.lg.Error("unmarshal raft entry",
			zap.Uint64("entry-term", s.term),
			zap.Uint64("entry-index", ent.Index))
		return
	}
	s.lg.Debug("applyEntryNormal", zap.Stringer("raftReq", &raftReq))

	if raftReq.Execute != nil {
		lr := raftReq.Execute
		lr.Index = ent.Index
		lr.Leader = s.getLead()
		lr.Shard = s.shardID
		lr.Node = s.nodeID
	}

	id := raftReq.Header.ID
	needResult := s.w.IsRegistered(id)
	if needResult || !noSideEffect(&raftReq) {
		ar = s.apply.Apply(&raftReq)
	}

	if ar == nil {
		return
	}

	s.w.Trigger(id, ar)

	//if !errors.Is(ar.err, errs.ErrNoSpace) || len(s.alarmStore.Get(pb.AlarmType_NOSPACE)) > 0 {
	//	s.w.Trigger(id, ar)
	//	return
	//}
	//
	//lg := sm.s.Logger()
	//lg.Warn(
	//	"message exceeded backend quota; raising alarm",
	//	zap.Int64("quota-size-bytes", sm.s.Cfg.QuotaBackendBytes),
	//	zap.String("quota-size", humanize.Bytes(uint64(sm.s.Cfg.QuotaBackendBytes))),
	//	zap.Error(ar.err),
	//)
	//
	//s.GoAttach(func() {
	//	//a := &pb.AlarmRequest{
	//	//	MemberID: uint64(s.ID()),
	//	//	Action:   pb.AlarmRequest_ACTIVATE,
	//	//	Alarm:    pb.AlarmType_NOSPACE,
	//	//}
	//	//s.raftRequest(s.ctx, pb.InternalRaftRequest{Alarm: a})
	//	s.w.Trigger(id, ar)
	//})
}

func (s *shard) Close() error {
	select {
	case <-s.stopping:
		return nil
	default:
		close(s.stopping)
	}

	<-s.done

	return nil
}

func (s *shard) isLeader() bool {
	leaderID := s.getLead()
	return leaderID != 0 && leaderID == s.nodeID
}

func (s *shard) isReady() bool {
	return s.getLead() > 0 && s.getTerm() > 0
}

func (s *shard) ChangeNotify() {
	s.changec <- struct{}{}
}

func (s *shard) setCommittedIndex(v uint64) {
	atomic.StoreUint64(&s.committedIndex, v)
}

func (s *shard) getCommittedIndex() uint64 {
	return atomic.LoadUint64(&s.committedIndex)
}

func (s *shard) setAppliedIndex(v uint64) {
	atomic.StoreUint64(&s.appliedIndex, v)
}

func (s *shard) getAppliedIndex() uint64 {
	return atomic.LoadUint64(&s.appliedIndex)
}

func (s *shard) setTerm(v uint64) {
	atomic.StoreUint64(&s.term, v)
}

func (s *shard) getTerm() uint64 {
	return atomic.LoadUint64(&s.term)
}

func (s *shard) setLead(v uint64) {
	atomic.StoreUint64(&s.lead, v)
}

func (s *shard) getLead() uint64 {
	return atomic.LoadUint64(&s.lead)
}
