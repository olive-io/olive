/*
Copyright 2025 The olive Authors

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

package server

import (
	"context"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/olive-io/olive/api/rpc/monpb"
	"github.com/olive-io/olive/mon/config"
	"github.com/olive-io/olive/mon/service/bpmn"
	"github.com/olive-io/olive/mon/service/cluster"
	"github.com/olive-io/olive/mon/service/system"
)

func ServersRegister(
	ctx context.Context,
	cfg *config.Config,
	lg *zap.Logger,
	v3cli *clientv3.Client,
) (map[string]http.Handler, func(gs *grpc.Server), func() error, error) {

	handlers := map[string]http.Handler{
		"/metrics": promhttp.Handler(),
	}

	clusterService, err := cluster.New(ctx, lg, v3cli)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "create cluster service")
	}

	systemService, err := system.New(ctx, lg, v3cli)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "create system service")
	}

	bpmnService, err := bpmn.New(ctx, cfg, lg, v3cli, cfg.NewIdGenerator())
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "create bpmn service")
	}

	register := func(gs *grpc.Server) {
		pb.RegisterClusterServer(gs, newCluster(clusterService))
		pb.RegisterSystemRPCServer(gs, newSystem(systemService))
		pb.RegisterBpmnRPCServer(gs, newBpmn(bpmnService))
		reflection.Register(gs)
	}

	startFn := func() error {
		if err := clusterService.Start(); err != nil {
			return errors.Wrap(err, "start cluster")
		}
		if err := systemService.Start(); err != nil {
			return errors.Wrap(err, "start system")
		}
		if err := bpmnService.Start(); err != nil {
			return errors.Wrap(err, "start bpmn")
		}

		return nil
	}

	return handlers, register, startFn, nil
}
