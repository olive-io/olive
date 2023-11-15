package server

import (
	"context"
	"sync"

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

type OliveServer struct {
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

	errorc chan error

	ctx    context.Context
	cancel context.CancelFunc

	raftEventCh chan raftio.LeaderInfo

	rmu             sync.RWMutex
	replicas        map[uint64]*Replica
	replicaRequestC chan *replicaRequest

	// wgMu blocks concurrent waitgroup mutation while server stopping
	wgMu sync.RWMutex
	// wg is used to wait for the goroutines that depends on the server state
	// to exit when stopping the server.
	wg sync.WaitGroup
}

func NewServer(lg *zap.Logger, cfg config.ServerConfig) (*OliveServer, error) {
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

	if err := nhc.Validate(); err != nil {
		return nil, err
	}

	nh, err := dragonboat.NewNodeHost(nhc)
	if err != nil {
		return nil, err
	}

	s := &OliveServer{
		ServerConfig:    &cfg,
		nh:              nh,
		lgMu:            new(sync.RWMutex),
		lg:              lg,
		raftEventCh:     raftChannel,
		replicas:        map[uint64]*Replica{},
		replicaRequestC: make(chan *replicaRequest, 128),
	}

	return s, nil
}

func (s *OliveServer) Start() {
	s.start()
	s.GoAttach(s.processReplicaEvent)
	s.GoAttach(s.processReplicaRequest)
}

func (s *OliveServer) start() {
	s.done = make(chan struct{})
	s.stop = make(chan struct{})
	s.stopping = make(chan struct{}, 1)
	s.errorc = make(chan error, 1)
	s.ctx, s.cancel = context.WithCancel(context.Background())

	go s.run()
}

func (s *OliveServer) Logger() *zap.Logger {
	s.lgMu.RLock()
	l := s.lg
	s.lgMu.RUnlock()
	return l
}

// HardStop stops the server without coordination with other raft group in the server.
func (s *OliveServer) HardStop() {
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
func (s *OliveServer) Stop() {
	lg := s.Logger()
	if err := s.TransferLeadership(); err != nil {
		lg.Warn("leadership transfer failed", zap.Error(err))
	}
	s.HardStop()
}

// StopNotify returns a channel that receives an empty struct
// when the server is stopped.
func (s *OliveServer) StopNotify() <-chan struct{} { return s.done }

// StoppingNotify returns a channel that receives an empty struct
// when the server is being stopped.
func (s *OliveServer) StoppingNotify() <-chan struct{} { return s.stopping }

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
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

func (s *OliveServer) run() {
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
