// Copyright 2023 The olive Authors
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

	cfg Config

	oct       *client.Client
	discovery dsy.IDiscovery

	started chan struct{}
}

func NewGateway(cfg Config) (*Gateway, error) {
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

	cfg.Config.UserHandlers = map[string]http.Handler{
		"/metrics": promhttp.Handler(),
	}
	cfg.Config.ServiceRegister = func(gs *grpc.Server) {
		pb.RegisterGatewayServer(gs, gw)
	}
	if cfg.EnableGRPCGateway {
		cfg.Config.GRPCGatewayRegister = func(ctx context.Context, mux *gwr.ServeMux) error {
			if rErr := pb.RegisterGatewayHandlerServer(ctx, mux, gw); rErr != nil {
				return rErr
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
	defer g.beStarted()

	if err := g.ProxyServer.Start(stopc); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	g.GoAttach(g.process)
	g.Destroy(g.destroy)

	<-stopc

	return g.stop()
}

func (g *Gateway) process() {}

func (g *Gateway) destroy() {}

func (g *Gateway) stop() error {
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
