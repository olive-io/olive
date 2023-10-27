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
	"sync"
	"sync/atomic"
	"time"

	"github.com/lni/dragonboat/v4"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"github.com/oliveio/olive/api"
	"github.com/oliveio/olive/pkg/cindex"
	"github.com/oliveio/olive/pkg/config"
	errs "github.com/oliveio/olive/pkg/errors"
	"github.com/oliveio/olive/pkg/mvcc"
	"github.com/oliveio/olive/pkg/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

type OliveServer struct {

	// inflightSnapshots holds count the number of snapshots currently inflight.
	inflightSnapshots int64  // must use atomic operations to access; keep 64-bit aligned.
	appliedIndex      uint64 // must use atomic operations to access; keep 64-bit aligned.
	committedIndex    uint64 // must use atomic operations to access; keep 64-bit aligned.
	term              uint64 // must use atomic operations to access; keep 64-bit aligned.
	lead              uint64 // must use atomic operations to access; keep 64-bit aligned.

	consistIndex cindex.IConsistentIndexer

	clusterID uint64

	nh *dragonboat.NodeHost

	Cfg config.ServerConfig

	lgMu *sync.RWMutex
	lg   *zap.Logger

	w wait.Wait

	// stop signals the run goroutine should shutdown.
	stop chan struct{}
	// stopping is closed by run goroutine on shutdown.
	stopping chan struct{}
	// done is closed when all goroutines from start() complete.
	done chan struct{}

	kv mvcc.IKV

	ctx    context.Context
	cancel context.CancelFunc

	bemu sync.Mutex
	be   backend.IBackend

	reqIDGen *idutil.Generator

	apply applier
	// applyV3Base is the core applier without auth or quotas
	applyBase applier
	// applyV3Internal is the applier for internal request
	//applyInternal applierV3Internal

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the goroutines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup
}

func (s *OliveServer) Logger() *zap.Logger {
	s.lgMu.RLock()
	l := s.lg
	s.lgMu.RUnlock()
	return l
}

func (s *OliveServer) parseProposeCtxErr(err error, start time.Time) error {
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

func (s *OliveServer) KV() mvcc.IKV { return s.kv }
func (s *OliveServer) Backend() backend.IBackend {
	s.bemu.Lock()
	defer s.bemu.Unlock()
	return s.be
}

// applyEntryNormal apples an EntryNormal type raftpb request to the OliveServer
func (s *OliveServer) applyEntryNormal(ent *sm.Entry) {
	var ar *applyResult
	index := s.consistIndex.ConsistentIndex()
	if ent.Index > index {
		// set the consistent index of current executing entry
		s.consistIndex.SetConsistentApplyingIndex(ent.Index)
		defer func() {
			// The txPostLockInsideApplyHook will not get called in some cases,
			// in which we should move the consistent index forward directly.
			newIndex := s.consistIndex.ConsistentIndex()
			if newIndex < ent.Index {
				s.consistIndex.SetConsistentIndex(ent.Index)
			}
		}()
	}
	s.lg.Debug("apply entry normal",
		zap.Uint64("consistent-index", index),
		zap.Uint64("entry-index", ent.Index))

	raftReq := api.InternalRaftRequest{}
	if err := raftReq.Unmarshal(ent.Cmd); err != nil {
		s.lg.Error("unmarshal raft entry",
			zap.Uint64("entry-index", ent.Index))
		return
	}
	s.lg.Debug("applyEntryNormal", zap.Stringer("raftReq", &raftReq))

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

func (s *OliveServer) setCommittedIndex(v uint64) {
	atomic.StoreUint64(&s.committedIndex, v)
}

func (s *OliveServer) getCommittedIndex() uint64 {
	return atomic.LoadUint64(&s.committedIndex)
}

func (s *OliveServer) setAppliedIndex(v uint64) {
	atomic.StoreUint64(&s.appliedIndex, v)
}

func (s *OliveServer) getAppliedIndex() uint64 {
	return atomic.LoadUint64(&s.appliedIndex)
}

func (s *OliveServer) setTerm(v uint64) {
	atomic.StoreUint64(&s.term, v)
}

func (s *OliveServer) getTerm() uint64 {
	return atomic.LoadUint64(&s.term)
}

func (s *OliveServer) setLead(v uint64) {
	atomic.StoreUint64(&s.lead, v)
}

func (s *OliveServer) getLead() uint64 {
	return atomic.LoadUint64(&s.lead)
}

// GoAttach creates a goroutine on a given function and tracks it using
// the etcdserver waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (s *OliveServer) GoAttach(f func()) {
	s.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer s.wgMu.RUnlock()
	select {
	case <-s.stopping:
		lg := s.Logger()
		lg.Warn("server has stopped; skipping GoAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		f()
	}()
}
