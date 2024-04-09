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
	"github.com/olive-io/olive/meta/leader"
	"github.com/olive-io/olive/meta/schedule"
	genericserver "github.com/olive-io/olive/pkg/server"
)

type Server struct {
	genericserver.IEmbedServer
	pb.UnsafeClusterServer
	pb.UnsafeMetaRPCServer
	pb.UnsafeBpmnRPCServer

	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	etcd  *embed.Etcd
	v3cli *clientv3.Client
	idReq *idutil.Generator

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

	ec.ServiceRegister = func(gs *grpc.Server) {
		pb.RegisterClusterServer(gs, s)
		pb.RegisterMetaRPCServer(gs, s)
		pb.RegisterBpmnRPCServer(gs, s)
	}

	var err error
	s.etcd, err = embed.StartEtcd(ec)
	if err != nil {
		return errors.Wrap(err, "start embed etcd")
	}

	<-s.etcd.Server.ReadyNotify()
	s.v3cli = v3client.New(s.etcd.Server)
	s.idReq = idutil.NewGenerator(uint16(s.etcd.Server.ID()), time.Now())
	s.notifier = leader.NewNotify(s.etcd.Server)

	sLimit := schedule.Limit{
		RegionLimit:     s.cfg.RegionLimit,
		DefinitionLimit: s.cfg.RegionDefinitionsLimit,
	}
	s.scheduler = schedule.New(s.ctx, s.lg, s.v3cli, s.notifier, sLimit, s.StoppingNotify())
	if err = s.scheduler.Start(); err != nil {
		return err
	}

	s.IEmbedServer.Destroy(s.destroy)

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
