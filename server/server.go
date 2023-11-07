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
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/mvcc"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

const (
	recommendedMaxRequestBytes = 100 * 1024 * 1024
)

var (
	recommendedMaxRequestBytesString = humanize.Bytes(uint64(recommendedMaxRequestBytes))
)

type KVServer struct {
	*config.ServerConfig

	nh *dragonboat.NodeHost

	lgMu *sync.RWMutex
	lg   *zap.Logger

	w wait.Wait

	// stop signals the run goroutine should shutdown.
	stop chan struct{}
	// stopping is closed by run goroutine on shutdown.
	stopping chan struct{}
	// done is closed when all goroutines from start() complete.
	done chan struct{}

	errorc chan error

	kv     mvcc.IKV
	lessor lease.ILessor

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

func NewServer(lg *zap.Logger, cfg config.ServerConfig) (*KVServer, error) {
	if lg == nil {
		lg = zap.NewExample()
	}

	if cfg.MaxRequestBytes > recommendedMaxRequestBytes {
		lg.Warn(
			"exceeded recommended request limit",
			zap.Uint64("max-request-bytes", cfg.MaxRequestBytes),
			zap.String("max-request-size", humanize.Bytes(cfg.MaxRequestBytes)),
			zap.Int("recommended-request-bytes", recommendedMaxRequestBytes),
			zap.String("recommended-request-size", recommendedMaxRequestBytesString),
		)
	}

	bepath := cfg.BackendPath()
	be := openBackend(lg, cfg, nil)

	raftChannel := make(chan raftio.LeaderInfo, 1)
	rel := newRaftEventListener(raftChannel)

	sel := newSystemEventListener()

	nhc := dbc.NodeHostConfig{
		WALDir:              cfg.WALPath(),
		NodeHostDir:         bepath,
		RTTMillisecond:      cfg.RTTMillisecond,
		RaftAddress:         cfg.ListenerPeerAddress,
		EnableMetrics:       true,
		RaftEventListener:   rel,
		SystemEventListener: sel,
		MutualTLS:           cfg.MutualTLS,
		CAFile:              cfg.CAFile,
		CertFile:            cfg.CertFile,
		KeyFile:             cfg.KeyFile,
	}

	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		return nil, err
	}

	heartbeat := time.Millisecond * time.Duration(cfg.RTTMillisecond) * time.Duration(cfg.HeartBeatTTL)
	minTTL := (3 * time.Millisecond * time.Duration(cfg.RTTMillisecond) * time.Duration(cfg.ElectionTTL)) / 2 * heartbeat

	// always recover lessor before kv. When we recover the mvcc.KV it will reattach keys to its leases.
	// If we recover mvcc.KV first, it will attach the keys to the wrong lessor before it recovers.
	lessor := lease.NewLessor(lg, be, lease.LessorConfig{
		MinLeaseTTL:                int64(math.Ceil(minTTL.Seconds())),
		CheckpointInterval:         cfg.LeaseCheckpointInterval,
		CheckpointPersist:          cfg.LeaseCheckpointPersist,
		ExpiredLeasesRetryInterval: cfg.ReqTimeout(),
	})

	kv := mvcc.New(lg, be, lessor, mvcc.StoreConfig{CompactionBatchLimit: cfg.CompactionBatchLimit})

	s := &KVServer{
		ServerConfig: &cfg,
		nh:           nh,
		lgMu:         new(sync.RWMutex),
		lg:           lg,
		kv:           kv,
		lessor:       lessor,
		bemu:         sync.Mutex{},
		be:           be,
		raftEventCh:  raftChannel,
		sms:          map[uint64]*shard{},
	}
	s.apply = s.newApplierBackend()

	return s, nil
}

func (s *KVServer) Start() {
	s.start()
	s.GoAttach(s.processRaftEvent)
}

func (s *KVServer) start() {
	s.w = wait.New()
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	s.stopping = make(chan struct{}, 1)
	s.errorc = make(chan error, 1)
	s.ctx, s.cancel = context.WithCancel(context.Background())

	go s.run()
}

func (s *KVServer) StartReplica(cfg config.ShardConfig) error {

	rc := dbc.Config{
		ShardID:             cfg.ShardID,
		CheckQuorum:         true,
		PreVote:             s.PreVote,
		ElectionRTT:         s.ElectionTTL,
		HeartbeatRTT:        s.HeartBeatTTL,
		SnapshotEntries:     10,
		CompactionOverhead:  5,
		OrderedConfigChange: true,
		WaitReady:           true,
	}

	join := cfg.NewCluster

	members := map[uint64]string{}
	for key, urlText := range cfg.PeerURLs {
		URL, err := url.Parse(urlText.String())
		if err != nil {
			return fmt.Errorf("invalid url: %v", urlText.String())
		}
		replicaID := GenHash([]byte(key))
		peerAddress := URL.Host
		if peerAddress == s.ListenerPeerAddress {
			rc.ReplicaID = replicaID
		}
		members[replicaID] = peerAddress
	}

	start := time.Now()
	err := s.nh.StartOnDiskReplica(members, join, s.NewDiskKV, rc)
	if err != nil {
		return err
	}

	electionTimeout := cfg.ElectionTimeout
	if electionTimeout == 0 {
		electionTimeout = time.Duration(math.MaxInt64)
	}
	after := time.NewTimer(electionTimeout)
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

		leaderID, term, ok, e1 := s.nh.GetLeaderID(cfg.ShardID)
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

// Stop stops the server gracefully, and shuts down the running goroutine.
// Stop should be called after a Start(s), otherwise it will block forever.
// When stopping leader, Stop transfers its leadership to one of its peers
// before stopping the server.
// Stop terminates the Server and performs any necessary finalization.
// Do and Process cannot be called after Stop has been invoked.
func (s *KVServer) Stop() {
	lg := s.Logger()
	if err := s.TransferLeadership(); err != nil {
		lg.Warn("leadership transfer failed", zap.Error(err))
	}
	s.HardStop()
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
		case <-s.stopping:
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

func (s *KVServer) ShardView(shard uint64) {
	lg := s.Logger()
	opt := dragonboat.DefaultNodeHostInfoOption
	opt.SkipLogInfo = true
	nhi := s.nh.GetNodeHostInfo(opt)
	lg.Sugar().Infof("%+v", nhi)
}

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
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

func (s *KVServer) run() {
	lg := s.Logger()

	defer func() {
		s.wgMu.Lock() // block concurrent waitgroup adds in GoAttach while stopping
		close(s.stopping)
		s.wgMu.Unlock()
		s.cancel()

		// wait for gouroutines before closing raft so wal stays open
		s.wg.Wait()

		s.CleanUp()

		close(s.done)
	}()

	for {
		select {
		case err := <-s.errorc:
			lg.Warn("server error", zap.Error(err))
			lg.Warn("data-dir used by this member must be removed")
			return
		case <-s.stop:
			return
		}
	}
}

func (s *KVServer) CleanUp() {
	lg := s.Logger()

	s.bemu.Lock()
	defer s.bemu.Unlock()

	s.be.ForceCommit()
	if err := s.be.Close(); err != nil {
		lg.Error("close backend", zap.Error(err))
	}
}

// TransferLeadership transfers the leader to the chosen transferee.
func (s *KVServer) TransferLeadership() error {
	// TODO: TransferLeadership
	return nil
}
