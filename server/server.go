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
	"fmt"
	"math"
	"net/url"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lni/dragonboat/v4"
	dbc "github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	errs "github.com/olive-io/olive/pkg/errors"
	"github.com/olive-io/olive/server/config"
	"github.com/olive-io/olive/server/mvcc"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

const (
	recommendedMaxRequestBytes = 10 * 1024 * 1024
)

var (
	recommendedMaxRequestBytesString = humanize.Bytes(uint64(recommendedMaxRequestBytes))
)

type KVServer struct {
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

	raftEventCh chan raftio.LeaderInfo

	smu sync.RWMutex
	sms map[uint64]*shard

	apply applier

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the goroutines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup
}

func NewServer(cfg config.ServerConfig) (*KVServer, error) {

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

	relCh := make(chan raftio.LeaderInfo, 10)
	rel := newRaftEventListener(relCh)

	sel := newSystemEventListener()

	nhc := dbc.NodeHostConfig{
		WALDir:              cfg.WALPath(),
		NodeHostDir:         bepath,
		RTTMillisecond:      cfg.RTTMillisecond,
		RaftAddress:         cfg.RaftAddress,
		EnableMetrics:       true,
		RaftEventListener:   rel,
		SystemEventListener: sel,
	}

	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		return nil, err
	}

	kv := mvcc.NewStore(lg, be, mvcc.StoreConfig{CompactionBatchLimit: cfg.CompactionBatchLimit})

	ctx, cancel := context.WithCancel(context.Background())
	s := &KVServer{
		nh:          nh,
		Cfg:         cfg,
		lgMu:        new(sync.RWMutex),
		lg:          lg,
		w:           wait.New(),
		stop:        make(chan struct{}),
		stopping:    make(chan struct{}, 1),
		done:        make(chan struct{}),
		kv:          kv,
		ctx:         ctx,
		cancel:      cancel,
		bemu:        sync.Mutex{},
		be:          be,
		raftEventCh: relCh,
		sms:         map[uint64]*shard{},
		wgMu:        sync.RWMutex{},
		wg:          sync.WaitGroup{},
	}
	s.apply = s.newApplierBackend()

	return s, nil
}

func (s *KVServer) Start() {
	s.GoAttach(s.processRaftEvent)
}

func (s *KVServer) StartReplica(cfg config.ShardConfig) error {

	shardID := GenHash([]byte(cfg.Name))
	rc := dbc.Config{
		ShardID:            shardID,
		CheckQuorum:        true,
		PreVote:            s.Cfg.PreVote,
		ElectionRTT:        s.Cfg.ElectionTicks,
		HeartbeatRTT:       s.Cfg.TickMs,
		SnapshotEntries:    10,
		CompactionOverhead: 5,
		WaitReady:          true,
	}

	join := cfg.NewCluster

	members := map[uint64]string{}
	for key, urlText := range cfg.PeerURLs {
		URL, err := url.Parse(urlText.String())
		if err != nil {
			return fmt.Errorf("invalid url: %v", urlText.String())
		}
		replicaID := GenHash([]byte(key))
		raftAddress := URL.Host
		if raftAddress == s.Cfg.RaftAddress {
			rc.ReplicaID = replicaID
		}
		members[replicaID] = raftAddress
	}

	start := time.Now()
	err := s.nh.StartOnDiskReplica(members, join, s.NewDiskKV, rc)
	if err != nil {
		return err
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = time.Duration(math.MaxInt64)
	}
	after := time.NewTimer(timeout)
	defer after.Stop()
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return errs.ErrStopped
		case <-after.C:
			return fmt.Errorf("wait shard ready: %w", errs.ErrTimeout)
		case <-ticker.C:
		}

		leaderID, term, ok, e1 := s.nh.GetLeaderID(shardID)
		if ok {
			s.lg.Info("start new shard",
				zap.Uint64("leader", leaderID),
				zap.Uint64("term", term),
				zap.Duration("duration", time.Now().Sub(start)))
			break
		}
		if e1 != nil {
			return fmt.Errorf("get leader %v", e1)
		}
	}

	return nil
}

func (s *KVServer) Logger() *zap.Logger {
	s.lgMu.RLock()
	l := s.lg
	s.lgMu.RUnlock()
	return l
}

func (s *KVServer) parseProposeCtxErr(err error, start time.Time) error {
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

func (s *KVServer) KV() mvcc.IKV { return s.kv }
func (s *KVServer) Backend() backend.IBackend {
	s.bemu.Lock()
	defer s.bemu.Unlock()
	return s.be
}

// HardStop stops the server without coordination with other raft group in the server.
func (s *KVServer) HardStop() {
	select {
	case s.stop <- struct{}{}:
	case <-s.done:
		return
	}
	<-s.done
}

// StopNotify returns a channel that receives an empty struct
// when the server is stopped.
func (s *KVServer) StopNotify() <-chan struct{} { return s.done }

// StoppingNotify returns a channel that receives an empty struct
// when the server is being stopped.
func (s *KVServer) StoppingNotify() <-chan struct{} { return s.stopping }

func (s *KVServer) processRaftEvent() {
	for {
		select {
		case <-s.stop:
			return
		case ch := <-s.raftEventCh:
			sm, ok := s.getShard(ch.ShardID)
			if ok {
				sm.setLead(ch.LeaderID)
				sm.setTerm(ch.Term)
				sm.ChangeNotify()
			}
		}
	}
}

func (s *KVServer) getShard(shardID uint64) (*shard, bool) {
	s.smu.RLock()
	defer s.smu.RUnlock()
	ssm, ok := s.sms[shardID]
	return ssm, ok
}

// GoAttach creates a goroutine on a given function and tracks it using
// the etcdserver waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (s *KVServer) GoAttach(f func()) {
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
