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

package plane

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

	pb "github.com/olive-io/olive/api/rpc/planepb"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/plane/leader"
	"github.com/olive-io/olive/plane/scheduler"
	"github.com/olive-io/olive/plane/server"
)

type Plane struct {
	genericserver.IEmbedServer
	pb.UnsafePlaneRPCServer
	pb.UnsafeBpmnRPCServer

	cfg Config

	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	etcd  *embed.Etcd
	v3cli *clientv3.Client
	idReq *idutil.Generator

	notifier leader.Notifier

	scheduler *scheduler.Scheduler
}

func NewPlane(cfg Config) (*Plane, error) {

	lg := cfg.Config.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	ctx, cancel := context.WithCancel(context.Background())
	s := &Plane{
		IEmbedServer: embedServer,
		cfg:          cfg,

		ctx:    ctx,
		cancel: cancel,

		lg: lg,
	}

	return s, nil
}

func (p *Plane) Start(stopc <-chan struct{}) error {
	ec := p.cfg.Config
	ec.EnableGRPCGateway = true

	ec.UserHandlers = map[string]http.Handler{
		"/metrics": promhttp.Handler(),
	}

	clusterRPC, err := server.NewCluster(p.v3cli, stopc)
	if err != nil {
		return errors.Wrap(err, "create cluster rpc")
	}

	ec.ServiceRegister = func(gs *grpc.Server) {
		pb.RegisterClusterServer(gs, clusterRPC)
		//pb.RegisterPlaneRPCServer(gs, s)
		//pb.RegisterBpmnRPCServer(gs, s)
	}

	p.etcd, err = embed.StartEtcd(ec)
	if err != nil {
		return errors.Wrap(err, "start embed etcd")
	}

	<-p.etcd.Server.ReadyNotify()
	p.v3cli = v3client.New(p.etcd.Server)
	p.idReq = idutil.NewGenerator(uint16(p.etcd.Server.ID()), time.Now())
	p.notifier = leader.NewNotify(p.etcd.Server)

	sLimit := scheduler.Limit{
		RegionLimit:     p.cfg.RegionLimit,
		DefinitionLimit: p.cfg.RegionDefinitionsLimit,
	}
	p.scheduler = scheduler.New(p.ctx, p.lg, p.v3cli, p.notifier, sLimit, p.StoppingNotify())
	if err = p.scheduler.Start(); err != nil {
		return errors.Wrap(err, "start scheduler")
	}

	p.IEmbedServer.Destroy(p.destroy)

	<-stopc

	return p.stop()
}

func (p *Plane) stop() error {
	p.IEmbedServer.Shutdown()
	return nil
}

func (p *Plane) destroy() {
	p.etcd.Server.HardStop()
	<-p.etcd.Server.StopNotify()
	p.cancel()
}
