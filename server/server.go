package server

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lni/dragonboat/v4"
	dbc "github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	"github.com/olive-io/olive/server/config"
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

	// stop signals the run goroutine should shutdown.
	stop chan struct{}
	// stopping is closed by run goroutine on shutdown.
	stopping chan struct{}
	// done is closed when all goroutines from start() complete.
	done chan struct{}

	// internalShard sets value when interval raft StartInternalReplica() completed
	internalShard uint64
	// internalWaitC is closed when internal raft StartInternalReplica() completed
	internalWaitC chan struct{}

	errorc chan error

	ctx    context.Context
	cancel context.CancelFunc

	raftEventCh chan raftio.LeaderInfo

	rmu      sync.RWMutex
	replicas map[uint64]*Replica

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

	raftChannel := make(chan raftio.LeaderInfo, 1)
	rel := newRaftEventListener(raftChannel)

	sel := newSystemEventListener()

	hostDir := cfg.BackendPath()
	nhc := dbc.NodeHostConfig{
		WALDir:              cfg.WALPath(),
		NodeHostDir:         hostDir,
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

	s := &KVServer{
		ServerConfig: &cfg,
		nh:           nh,
		lgMu:         new(sync.RWMutex),
		lg:           lg,
		raftEventCh:  raftChannel,
		replicas:     map[uint64]*Replica{},
	}

	return s, nil
}

func (s *KVServer) Start() {
	s.start()
	s.GoAttach(s.processRaftEvent)
}

func (s *KVServer) start() {
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	s.stopping = make(chan struct{}, 1)
	s.errorc = make(chan error, 1)
	s.internalWaitC = make(chan struct{}, 1)
	s.ctx, s.cancel = context.WithCancel(context.Background())

	go s.run()
}

func (s *KVServer) StartInternalReplica(cfg config.ShardConfig) error {
	if _, ok := s.isReadyInternalReplica(); ok {
		return nil
	}

	err := s.StartReplica(cfg)
	if err != nil {
		return err
	}

	s.readyInternalReplica(cfg.ShardID)
	return nil
}

func (s *KVServer) isReadyInternalReplica() (uint64, bool) {
	select {
	case <-s.internalWaitC:
		return atomic.LoadUint64(&s.internalShard), true
	default:
		return 0, false
	}
}

func (s *KVServer) readyInternalReplica(shardID uint64) {
	select {
	case <-s.internalWaitC:
	default:
		close(s.internalWaitC)
	}

	atomic.StoreUint64(&s.internalShard, shardID)
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
			return ErrStopped
		case <-after.C:
			return fmt.Errorf("wait shard ready: %w", ErrTimeout)
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
		return ErrCanceled

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
		return ErrTimeout

	default:
		return err
	}
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
			ra, ok := s.getReplica(ch.ShardID)
			if ok {
				ra.raetLead(ch.LeaderID)
				ra.raetTerm(ch.Term)
				ra.ChangeNotify()
			}
		}
	}
}

func (s *KVServer) getReplica(shardID uint64) (*Replica, bool) {
	s.rmu.RLock()
	defer s.rmu.RUnlock()
	ssm, ok := s.replicas[shardID]
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

// TransferLeadership transfers the leader to the chosen transferee.
func (s *KVServer) TransferLeadership() error {
	// TODO: TransferLeadership
	return nil
}

func (s *KVServer) InternalCluster() (RaftStatusGetter, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}

	ra, ok := s.getReplica(shardID)
	if !ok {
		panic("invalid shardID from readyInternalReplica()")
	}
	return ra, nil
}
