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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrAlreadyStarted = errors.New("olive-meta: already started")
	ErrServerStart    = errors.New("olive-meta: start server")
	ErrServerStop     = errors.New("olive-meta: start stop")
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

func NewServer(cfg Config) (*Server, error) {

	lg := cfg.Server.Logger.GetLogger()

	kvs, err := server.NewServer(cfg.Server)
	if err != nil {
		return nil, err
	}

	var ts net.Listener
	tc, isTls, err := cfg.Server.TLSConfig()
	if err != nil {
		return nil, err
	}

	if isTls {
		ts, err = tls.Listen("tcp", cfg.ListenerAddress, tc)
	} else {
		ts, err = net.Listen("tcp", cfg.ListenerAddress)
	}
	if err != nil {
		return nil, err
	}

	cred := insecure.NewCredentials()
	gs := grpc.NewServer(grpc.Creds(cred))

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

	name := s.cfg.Server.Name
	scfg := config.ShardConfig{
		Name:       name,
		PeerURLs:   s.cfg.PeerURLs,
		NewCluster: false,
	}

	if s.cfg.ShardTimeout != 0 {
		scfg.Timeout = s.cfg.ShardTimeout
	}

	shardID, err := s.kvs.StartReplica(scfg)
	if err != nil {
		return err
	}
	s.setShardID(shardID)

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
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	if otherHandler == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			grpcServer.ServeHTTP(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
