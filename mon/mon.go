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

	"github.com/cockroachdb/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"

	"github.com/olive-io/olive/mon/config"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/mon/scheduler"
	"github.com/olive-io/olive/mon/server"
	genericserver "github.com/olive-io/olive/pkg/server"
)

type Monitor struct {
	genericserver.IEmbedServer

	cfg config.Config

	lg *zap.Logger

	etcd     *embed.Etcd
	v3cli    *clientv3.Client
	notifier leader.Notifier
}

func New(cfg config.Config) (*Monitor, error) {

	lg := cfg.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	s := &Monitor{
		IEmbedServer: embedServer,
		cfg:          cfg,
		lg:           lg,
	}

	return s, nil
}

func (mon *Monitor) Start(ctx context.Context) error {
	lg := mon.lg

	lg.Info("starting olive monitor")

	ec := mon.cfg.Config
	ec.EnableGRPCGateway = true

	cfg := mon.cfg

	mon.v3cli = new(clientv3.Client)
	handlers, register, err := server.ServersRegister(ctx, &cfg, lg, mon.v3cli)
	if err != nil {
		return err
	}
	ec.UserHandlers = handlers
	ec.ServiceRegister = register

	etcd, err := embed.StartEtcd(ec)
	if err != nil {
		return errors.Wrap(err, "start embed etcd")
	}

	*mon.v3cli = *v3client.New(etcd.Server)
	mon.notifier = leader.NewNotify(etcd.Server)

	<-etcd.Server.ReadyNotify()
	leader.InitNotifier(etcd.Server)
	mon.etcd = etcd

	if err = scheduler.StartScheduler(ctx, lg, mon.v3cli, mon.notifier); err != nil {
		return errors.Wrap(err, "start global scheduler")
	}

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
