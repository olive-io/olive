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

package mon

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"

	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/mon/server"
	genericserver "github.com/olive-io/olive/pkg/server"
)

type Monitor struct {
	genericserver.IEmbedServer

	cfg Config

	lg *zap.Logger

	etcd  *embed.Etcd
	v3cli *clientv3.Client

	idGen *idutil.Generator

	notifier leader.Notifier

	//scheduler *scheduler.Scheduler
}

func New(cfg Config) (*Monitor, error) {

	lg := cfg.Config.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	s := &Monitor{
		IEmbedServer: embedServer,

		cfg: cfg,

		lg: lg,
	}

	return s, nil
}

func (mon *Monitor) Start(ctx context.Context) error {
	lg := mon.lg

	lg.Info("starting olive monitor")

	ec := mon.cfg.Config
	ec.EnableGRPCGateway = true

	handlers, register, err := server.ServersRegister(ctx, lg, mon.v3cli)
	if err != nil {
		return err
	}
	ec.UserHandlers = handlers
	ec.ServiceRegister = register

	etcd, err := embed.StartEtcd(ec)
	if err != nil {
		return errors.Wrap(err, "start embed etcd")
	}

	<-etcd.Server.ReadyNotify()

	v3cli := v3client.New(etcd.Server)
	idGen := idutil.NewGenerator(uint16(etcd.Server.ID()), time.Now())
	notifier := leader.NewNotify(etcd.Server)

	//sLimit := scheduler.Limit{
	//	RegionLimit:     p.cfg.RegionLimit,
	//	DefinitionLimit: p.cfg.RegionDefinitionsLimit,
	//}
	//p.scheduler = scheduler.New(p.ctx, p.lg, p.v3cli, p.notifier, sLimit, p.StoppingNotify())
	//if err = p.scheduler.Start(); err != nil {
	//	return errors.Wrap(err, "start scheduler")
	//}

	mon.etcd = etcd
	mon.v3cli = v3cli
	mon.idGen = idGen
	mon.notifier = notifier

	mon.IEmbedServer.Destroy(mon.destroy)

	<-ctx.Done()

	return mon.stop()
}

func (mon *Monitor) stop() error {
	mon.IEmbedServer.Shutdown()
	return nil
}

func (mon *Monitor) destroy() {
	mon.etcd.Server.HardStop()
	<-mon.etcd.Server.StopNotify()
}
