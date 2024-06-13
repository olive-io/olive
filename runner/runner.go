/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package runner

import (
	"context"
	"net"
	"net/http"
	urlpkg "net/url"
	"sync"

	"github.com/go-logr/zapr"
	"github.com/gofrs/flock"
	gw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	"github.com/olive-io/olive/apis"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/client-go"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/olive-io/olive/runner/raft"
)

type Runner struct {
	genericdaemon.IDaemon

	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	gLock *flock.Flock

	scheme *krt.Scheme

	client *clientgo.Client
	be     backend.IBackend
	serve  *http.Server

	controller *raft.Controller

	// rw locker for local *monv1.Runner
	prMu sync.RWMutex
	pr   *corev1.Runner
}

func NewRunner(cfg *Config) (*Runner, error) {
	lg := cfg.GetLogger()
	// set klog by zap Logger
	klog.SetLogger(zapr.NewLogger(lg))

	gLock, err := cfg.LockDataDir()
	if err != nil {
		return nil, err
	}
	lg.Debug("protected directory: " + cfg.DataDir)

	scheme := apis.Scheme
	client, err := clientgo.New(cfg.clientConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	be := newBackend(cfg)

	embedDaemon := genericdaemon.NewEmbedDaemon(lg)
	runner := &Runner{
		IDaemon: embedDaemon,
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		gLock:   gLock,
		scheme:  scheme,
		client:  client,
		be:      be,
	}

	return runner, nil
}

func (r *Runner) Logger() *zap.Logger {
	return r.cfg.GetLogger()
}

func (r *Runner) Start(stopc <-chan struct{}) error {
	if err := r.start(); err != nil {
		return err
	}
	if err := r.startGRPCServer(); err != nil {
		return err
	}

	r.OnDestroy(r.destroy)
	r.GoAttach(r.process)

	<-stopc

	return r.stop()
}

func (r *Runner) start() error {
	lg := r.Logger()

	var err error

	defer r.be.ForceCommit()
	tx := r.be.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Meta)
	tx.UnsafeCreateBucket(buckets.Region)
	tx.UnsafeCreateBucket(buckets.Key)
	tx.Unlock()
	if err = tx.Commit(); err != nil {
		lg.Error("pebble commit", zap.Error(err))
	}

	runner, err := r.register()
	if err != nil {
		return err
	}
	r.persistRunner(runner)

	r.controller, err = r.startRaftController()
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) stop() error {
	r.IDaemon.Shutdown()
	return nil
}

func (r *Runner) startGRPCServer() error {
	lg := r.Logger()

	scheme, ts, err := r.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	r.cfg.ListenClientURL = scheme + ts.Addr().String()

	gs, gwmux, err := r.buildGRPCServer()
	if err != nil {
		return err
	}
	handler := r.buildUserHandler()

	mux := r.createMux(gwmux, handler)
	r.serve = &http.Server{
		Handler:        genericdaemon.GRPCHandlerFunc(gs, mux),
		MaxHeaderBytes: 1024 * 1024 * 20,
	}

	r.GoAttach(func() {
		_ = r.serve.Serve(ts)
	})

	return nil
}

func (r *Runner) createListener() (string, net.Listener, error) {
	cfg := r.cfg
	lg := r.Logger()
	url, err := urlpkg.Parse(cfg.ListenClientURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return url.Scheme + "://", listener, nil
}

func (r *Runner) buildGRPCServer() (*grpc.Server, *gw.ServeMux, error) {
	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)
	mux := gw.NewServeMux()

	runnerGRPC := &runnerImpl{controller: r.controller}
	pb.RegisterRunnerRPCServer(gs, runnerGRPC)

	return gs, mux, nil
}

func (r *Runner) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())

	return handler
}

func (r *Runner) createMux(gwmux *gw.ServeMux, handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	if gwmux != nil {
		mux.Handle(
			"/v1/",
			wsproxy.WebsocketProxy(
				gwmux,
				wsproxy.WithRequestMutator(
					// Default to the POST method for streams
					func(_ *http.Request, outgoing *http.Request) *http.Request {
						outgoing.Method = "POST"
						return outgoing
					},
				),
				wsproxy.WithMaxRespBodyBufferSize(0x7fffffff),
			),
		)
	}
	if handler != nil {
		mux.Handle("/", handler)
	}
	return mux
}

func (r *Runner) startRaftController() (*raft.Controller, error) {
	cfg := r.cfg
	lg := r.Logger()

	listenPeerURL, err := urlpkg.Parse(cfg.ListenPeerURL)
	if err != nil {
		return nil, err
	}
	raftAddr := listenPeerURL.Host

	raftConfig := raft.NewConfig()
	raftConfig.DataDir = cfg.RegionDir()
	raftConfig.RaftAddress = raftAddr
	raftConfig.HeartbeatMs = cfg.HeartbeatMs
	raftConfig.RaftRTTMillisecond = cfg.RaftRTTMillisecond
	raftConfig.Logger = lg

	pr := r.getRunner()
	controller, err := raft.NewController(r.ctx, raftConfig, r.be, r.client, pr)
	if err != nil {
		return nil, err
	}

	lg.Info("start raft container")
	if err = controller.Start(r.StoppingNotify()); err != nil {
		return nil, err
	}

	return controller, nil
}

func (r *Runner) destroy() {
	r.cancel()
	if err := r.gLock.Unlock(); err != nil {
		r.Logger().Error("released "+r.cfg.DataDir, zap.Error(err))
	}
}
