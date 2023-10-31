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
	"net"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	srv      *grpc.Server
	listener net.Listener

	kvs *server.KVServer

	stop chan struct{}
}

func NewServer(cfg Config) (*Server, error) {

	lg := cfg.ServerConfig.Logger.GetLogger()

	kvs, err := server.NewServer(cfg.ServerConfig)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", cfg.ListenerAddress)
	if err != nil {
		return nil, err
	}

	cred := insecure.NewCredentials()
	srv := grpc.NewServer(grpc.Creds(cred))

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		cfg: cfg,

		ctx:    ctx,
		cancel: cancel,

		lg: lg,

		srv:      srv,
		listener: listener,

		kvs: kvs,

		stop: make(chan struct{}),
	}

	api.RegisterDefinitionRPCServer(srv, s)

	return s, nil
}

func (s *Server) Start() error {
	s.kvs.Start()

	scfg := s.cfg.ShardConfig
	if err := s.kvs.StartReplica(scfg); err != nil {
		return err
	}

	go func() {
		if err := s.srv.Serve(s.listener); err != nil {

		}
	}()

	return nil
}
