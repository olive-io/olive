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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lni/dragonboat/v4"
	dbc "github.com/lni/dragonboat/v4/config"
	errs "github.com/olive-io/olive/pkg/errors"
	"github.com/olive-io/olive/server/config"
	"github.com/olive-io/olive/server/mvcc"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

const (
	DefaultSnapshotCount = 100000

	recommendedMaxRequestBytes = 10
)

var (
	recommendedMaxRequestBytesString = humanize.Bytes(uint64(recommendedMaxRequestBytes))
)

type OliveServer struct {

	// inflightSnapshots holds count the number of snapshots currently inflight.
	inflightSnapshots int64 // must use atomic operations to access; keep 64-bit aligned.

	nh *dragonboat.NodeHost

	Cfg config.ServerConfig

	shardMu   sync.RWMutex
	ShardCfgs map[uint64]config.SharedConfig

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

	smu sync.RWMutex
	sms map[uint64]*stateMachine

	apply applier

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the goroutines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup
}

func NewServer(cfg config.ServerConfig) (*OliveServer, error) {

	lg := cfg.Logger.GetLogger()
	if cfg.MaxRequestBytes > recommendedMaxRequestBytes {
		lg.Warn(
			"exceeded recommended request limit",
			zap.Uint("max-request-bytes", cfg.MaxRequestBytes),
			zap.String("max-request-size", humanize.Bytes(uint64(cfg.MaxRequestBytes))),
			zap.Int("recommended-request-bytes", recommendedMaxRequestBytes),
			zap.String("recommended-request-size", recommendedMaxRequestBytesString),
		)
	}

	bepath := cfg.BackendPath()
	be := openBackend(cfg, nil)

	nhc := dbc.NodeHostConfig{
		WALDir:              cfg.WALPath(),
		NodeHostDir:         bepath,
		RTTMillisecond:      cfg.RTTMillisecond,
		EnableMetrics:       true,
		RaftEventListener:   nil,
		SystemEventListener: nil,
	}

	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		return nil, err
	}

	s := &OliveServer{
		inflightSnapshots: 0,
		nh:                nh,
		Cfg:               cfg,
		ShardCfgs:         map[uint64]config.SharedConfig{},
		lgMu:              nil,
		lg:                nil,
		w:                 nil,
		stop:              nil,
		stopping:          nil,
		done:              nil,
		kv:                nil,
		ctx:               nil,
		cancel:            nil,
		bemu:              sync.Mutex{},
		be:                be,
		reqIDGen:          nil,
		sms:               map[uint64]*stateMachine{},
		apply:             nil,
		wgMu:              sync.RWMutex{},
		wg:                sync.WaitGroup{},
	}

	return s, nil
}

func (s *OliveServer) StartReplica(sharedCfg config.SharedConfig) error {

	rc := dbc.Config{
		ReplicaID:          sharedCfg.ClusterID,
		ShardID:            sharedCfg.SharedID,
		CheckQuorum:        true,
		PreVote:            s.Cfg.PreVote,
		ElectionRTT:        s.Cfg.ElectionTicks,
		HeartbeatRTT:       s.Cfg.TickMs,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
		WaitReady:          true,
	}

	join := sharedCfg.NewCluster

	s.shardMu.Lock()
	s.ShardCfgs[sharedCfg.SharedID] = sharedCfg
	s.shardMu.Unlock()

	members := map[uint64]string{}
	err := s.nh.StartOnDiskReplica(members, join, s.NewDiskKV, rc)
	if err != nil {
		return err
	}

	return nil
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

func (s *OliveServer) getShard(shardID uint64) (*stateMachine, bool) {
	s.shardMu.RLock()
	defer s.shardMu.RUnlock()
	ssm, ok := s.sms[shardID]
	return ssm, ok
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
