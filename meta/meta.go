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

package meta

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/server"
	"github.com/olive-io/olive/server/config"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	grpccredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

	kvs *server.KVServer

	shardID uint64

	wg sync.WaitGroup

	errorc chan error
	startc chan struct{}
	stopc  chan struct{}
}

func NewServer(lg *zap.Logger, cfg Config) (*Server, error) {
	if lg == nil {
		lg = zap.NewExample()
	}

	kvs, err := server.NewServer(lg, cfg.ServerConfig)
	if err != nil {
		return nil, err
	}

	var ts net.Listener
	tc, isTLS, err := cfg.ServerConfig.TLSConfig()
	if err != nil {
		return nil, err
	}

	opts := make([]grpc.ServerOption, 0)
	cred := insecure.NewCredentials()
	if isTLS {
		cred = grpccredentials.NewTLS(tc)
		ts, err = tls.Listen("tcp", cfg.ListenerClientAddress, tc)
	} else {
		ts, err = net.Listen("tcp", cfg.ListenerClientAddress)
	}
	if err != nil {
		return nil, err
	}
	opts = append(opts, grpc.Creds(cred))

	if cfg.MaxGRPCSendMessageSize != 0 {
		opts = append(opts, grpc.MaxSendMsgSize(int(cfg.MaxGRPCSendMessageSize)))
	}
	if cfg.MaxGRPCReceiveMessageSize != 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(int(cfg.MaxGRPCReceiveMessageSize)))
	}

	gs := grpc.NewServer(opts...)

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg: cfg,

		ctx:    ctx,
		cancel: cancel,

		lg: lg,

		gs:       gs,
		listener: ts,

		kvs: kvs,

		errorc: make(chan error, 1),
		startc: make(chan struct{}, 1),
		stopc:  make(chan struct{}, 1),
	}

	api.RegisterDefinitionRPCServer(gs, s)

	return s, nil
}

func (s *Server) Start() error {
	select {
	case <-s.startc:
		return ErrAlreadyStarted
	default:
	}

	defer close(s.startc)

	s.kvs.Start()

	name := s.cfg.Name
	scfg := config.ShardConfig{
		Name:       name,
		ShardID:    metaShard,
		PeerURLs:   s.cfg.InitialCluster,
		NewCluster: false,
	}

	if s.cfg.ElectionTimeout != 0 {
		scfg.ElectionTimeout = s.cfg.ElectionTimeout
	}

	err := s.kvs.StartReplica(scfg)
	if err != nil {
		return err
	}
	s.setShardID(scfg.ShardID)

	mux := http.NewServeMux()

	handler := grpcHandlerFunc(s.gs, mux)
	srv := &http.Server{
		Handler: handler,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := srv.Serve(s.listener); err != nil {
			s.errorc <- fmt.Errorf("%w: %v", ErrServerStart, err)
		}
	}()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		select {
		case <-s.stopc:
		}

		ctx := context.Background()
		if err := srv.Shutdown(ctx); err != nil {
			s.errorc <- fmt.Errorf("%w: %v", ErrServerStop, err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	select {
	case <-s.stopc:
		return nil
	default:
	}

	close(s.stopc)
	s.cancel()
	return nil
}

func (s *Server) GracefulStop() error {
	if err := s.Stop(); err != nil {
		return err
	}

	s.wg.Wait()
	return nil
}

func (s *Server) setShardID(id uint64) {
	atomic.StoreUint64(&s.shardID, id)
}

func (s *Server) getShardID() uint64 {
	return atomic.LoadUint64(&s.shardID)
}

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
