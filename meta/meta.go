package meta

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/olive-io/olive/api/olivepb"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

var (
	ErrAlreadyStarted = errors.New("olive-meta: already started")
	ErrServerStart    = errors.New("olive-meta: start server")
	ErrServerStop     = errors.New("olive-meta: start stop")
)

const (
	metaShard = uint64(128)
)

type Server struct {
	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	gs       *grpc.Server
	listener net.Listener

	etcd *embed.Etcd

	shardID uint64

	wgMu sync.RWMutex
	wg   sync.WaitGroup

	stopping chan struct{}
	done     chan struct{}
	stop     chan struct{}
}

func NewServer(cfg Config) (*Server, error) {

	lg := cfg.Config.GetLogger()

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg: cfg,

		ctx:    ctx,
		cancel: cancel,

		lg: lg,

		stopping: make(chan struct{}, 1),
		stop:     make(chan struct{}, 1),
		done:     make(chan struct{}, 1),
	}

	return s, nil
}

func (s *Server) Start() error {
	ec := s.cfg.Config
	ec.EnableGRPCGateway = true
	ec.ServiceRegister = func(gs *grpc.Server) {
		olivepb.RegisterBpmnRPCServer(gs, s)
		olivepb.RegisterRunnerRPCServer(gs, s)
	}

	//mux := newHttpMux()
	//ec.UserHandlers = map[string]http.Handler{
	//	"/": mux,
	//}

	etcd, err := embed.StartEtcd(ec)
	if err != nil {
		return err
	}
	s.etcd = etcd

	<-s.etcd.Server.ReadyNotify()

	//mux := s.newHttpMux()
	//
	//var err error
	//s.gs, err = s.newGRPCServe()
	//if err != nil {
	//	return err
	//}
	//
	//handler := grpcHandlerFunc(s.gs, mux)
	//srv := &http.Server{
	//	Handler: handler,
	//}
	//
	//s.wg.Add(1)
	//go func() {
	//	defer s.wg.Done()
	//	if err := srv.Serve(s.listener); err != nil {
	//		s.errorc <- fmt.Errorf("%w: %v", ErrServerStart, err)
	//	}
	//}()
	//
	//s.wg.Add(1)
	//go func() {
	//	defer s.wg.Done()
	//
	//	select {
	//	case <-s.stopc:
	//	}
	//
	//	ctx := context.Background()
	//	if err := srv.Shutdown(ctx); err != nil {
	//		s.errorc <- fmt.Errorf("%w: %v", ErrServerStop, err)
	//	}
	//}()

	return nil
}

func (s *Server) Stop() error {
	return nil
}

func (s *Server) GracefulStop() error {
	if err := s.Stop(); err != nil {
		return err
	}

	s.etcd.Server.HardStop()

	<-s.etcd.Server.StopNotify()

	s.wg.Wait()
	return nil
}

//func newHttpMux() http.Handler {
//	mux := http.NewServeMux()
//	mux.Handle("/metrics", promhttp.Handler())
//
//	return mux
//}

// grpcHandlerFunc returns a http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Given in gRPC docs.
func grpcHandlerFunc(gh *grpc.Server, hh http.Handler) http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			gh.ServeHTTP(w, r)
		} else {
			hh.ServeHTTP(w, r)
		}
	}), h2s)
}

// StopNotify returns a channel that receives an empty struct
// when the server is stopped.
func (s *Server) StopNotify() <-chan struct{} { return s.done }

// StoppingNotify returns a channel that receives an empty struct
// when the server is being stopped.
func (s *Server) StoppingNotify() <-chan struct{} { return s.stopping }

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (s *Server) GoAttach(f func()) {
	s.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer s.wgMu.RUnlock()
	select {
	case <-s.stopping:
		s.lg.Warn("server has stopped; skipping GoAttach")
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
