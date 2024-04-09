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

package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/client"
	dsy "github.com/olive-io/olive/pkg/discovery"
	grpcproxy "github.com/olive-io/olive/pkg/proxy/server/grpc"
	"github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
)

type Gateway struct {
	pb.UnsafeGatewayServer
	genericserver.IEmbedServer
	*grpcproxy.ProxyServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	oct       *client.Client
	discovery dsy.IDiscovery

	started chan struct{}
}

func NewGateway(cfg *Config) (*Gateway, error) {
	lg := cfg.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	lg.Debug("connect to olive-meta",
		zap.String("endpoints", strings.Join(cfg.Client.Endpoints, ",")))
	oct, err := client.New(cfg.Client)
	if err != nil {
		return nil, err
	}

	prefix := runtime.DefaultRunnerDiscoveryNode
	discovery, err := dsy.NewDiscovery(oct.Client, dsy.SetLogger(lg), dsy.Prefix(prefix))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	gw := &Gateway{
		IEmbedServer: embedServer,
		ctx:          ctx,
		cancel:       cancel,
		cfg:          cfg,
		oct:          oct,
		discovery:    discovery,
		started:      make(chan struct{}),
	}

	cfg.Config.Context = ctx
	if cfg.Config.UserHandlers == nil {
		cfg.Config.UserHandlers = map[string]http.Handler{}
	}
	cfg.Config.UserHandlers["/metrics"] = promhttp.Handler()

	serviceRegister := cfg.Config.ServiceRegister
	cfg.Config.ServiceRegister = func(gs *grpc.Server) {
		if serviceRegister != nil {
			serviceRegister(gs)
		}
		pb.RegisterGatewayServer(gs, gw)
	}
	if cfg.EnableGRPCGateway {
		gatewayRegister := cfg.Config.GRPCGatewayRegister
		cfg.Config.GRPCGatewayRegister = func(ctx context.Context, mux *gwr.ServeMux) error {
			if gatewayRegister != nil {
				if e1 := gatewayRegister(ctx, mux); e1 != nil {
					return e1
				}
			}
			if e1 := pb.RegisterGatewayHandlerServer(ctx, mux, gw); e1 != nil {
				return e1
			}
			return nil
		}
	}

	gw.ProxyServer, err = grpcproxy.NewProxyServer(discovery, &cfg.Config)
	if err != nil {
		return nil, err
	}
	if err = pb.RegisterTestServiceServerHandler(gw, &TestRPC{}); err != nil {
		return nil, err
	}

	return gw, nil
}

func (g *Gateway) Logger() *zap.Logger {
	return g.cfg.GetLogger()
}

func (g *Gateway) Start(stopc <-chan struct{}) error {
	if g.isStarted() {
		return nil
	}

	if err := g.ProxyServer.Start(stopc); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	g.GoAttach(g.process)
	g.Destroy(g.destroy)

	g.beStarted()

	<-stopc

	return g.stop()
}

func (g *Gateway) process() {}

func (g *Gateway) destroy() {}

func (g *Gateway) stop() error {
	g.cancel()
	g.IEmbedServer.Shutdown()
	return nil
}

func (g *Gateway) beStarted() {
	select {
	case <-g.started:
		return
	default:
		close(g.started)
	}
}

func (g *Gateway) isStarted() bool {
	select {
	case <-g.started:
		return true
	default:
		return false
	}
}

func (g *Gateway) StartNotify() <-chan struct{} {
	return g.started
}
