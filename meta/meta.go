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

package meta

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
	genericserver "github.com/olive-io/olive/pkg/server"

	"github.com/olive-io/olive/meta/leader"
	"github.com/olive-io/olive/meta/schedule"
)

type Server struct {
	genericserver.IEmbedServer

	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	etcd  *embed.Etcd
	v3cli *clientv3.Client
	idGen *idutil.Generator

	notifier leader.Notifier

	scheduler *schedule.Scheduler
}

func NewServer(cfg Config) (*Server, error) {

	lg := cfg.Config.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		IEmbedServer: embedServer,
		cfg:          cfg,

		ctx:    ctx,
		cancel: cancel,

		lg: lg,
	}

	return s, nil
}

func (s *Server) Start(stopc <-chan struct{}) error {
	ec := s.cfg.Config
	ec.EnableGRPCGateway = true

	ec.UserHandlers = map[string]http.Handler{
		"/metrics": promhttp.Handler(),
	}

	clusterRPC, err := newClusterServer(s)
	if err != nil {
		return err
	}
	runnerRPC, err := newRunnerServer(s)
	if err != nil {
		return err
	}
	authRPC, err := newAuthServer(s)
	if err != nil {
		return err
	}
	bpmnRPC, err := newBpmnServer(s)
	if err != nil {
		return err
	}
	ec.ServiceRegister = func(gs *grpc.Server) {
		pb.RegisterMetaClusterRPCServer(gs, clusterRPC)
		pb.RegisterMetaRunnerRPCServer(gs, runnerRPC)
		pb.RegisterMetaRegionRPCServer(gs, runnerRPC)
		pb.RegisterAuthRPCServer(gs, authRPC)
		pb.RegisterRbacRPCServer(gs, authRPC)
		pb.RegisterBpmnRPCServer(gs, bpmnRPC)
	}

	s.etcd, err = embed.StartEtcd(ec)
	if err != nil {
		return errors.Wrap(err, "start embed etcd")
	}

	select {
	case <-s.etcd.Server.ReadyNotify():
	case <-stopc:
		return errors.New("etcd server not stops")
	}

	s.v3cli = v3client.New(s.etcd.Server)
	s.idGen = idutil.NewGenerator(uint16(s.etcd.Server.ID()), time.Now())
	s.notifier = leader.NewNotify(s.etcd.Server)

	sLimit := schedule.Limit{
		RegionLimit:     s.cfg.RegionLimit,
		DefinitionLimit: s.cfg.RegionDefinitionsLimit,
	}
	s.scheduler = schedule.New(s.ctx, s.lg, s.v3cli, s.notifier, sLimit, s.StoppingNotify())
	if err = s.scheduler.Start(); err != nil {
		return err
	}

	if err = authRPC.Prepare(s.ctx); err != nil {
		return err
	}

	s.IEmbedServer.OnDestroy(s.destroy)

	<-stopc

	return s.stop()
}

func (s *Server) stop() error {
	s.IEmbedServer.Shutdown()
	return nil
}

func (s *Server) destroy() {
	s.etcd.Server.HardStop()
	<-s.etcd.Server.StopNotify()
	s.cancel()
}
