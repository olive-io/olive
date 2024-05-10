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
	"os"
	"path"
	"strings"
	"sync"

	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	json "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"sigs.k8s.io/yaml"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/client"
	dsy "github.com/olive-io/olive/pkg/discovery"
	grpcproxy "github.com/olive-io/olive/pkg/proxy/server/grpc"
	ort "github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/tonic/openapi"

	"github.com/olive-io/olive/gateway/consumer"
)

type Gateway struct {
	genericserver.IEmbedServer
	*grpcproxy.ProxyServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	oct       *client.Client
	discovery dsy.IDiscovery

	started chan struct{}

	hmu             sync.RWMutex
	hall            *hall
	handlerWrappers []consumer.HandlerWrapper

	// internal http Consumer
	hc *consumer.HttpConsumer

	openapiDocs *openapi.OpenAPI
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

	prefix := ort.DefaultRunnerDiscoveryNode
	discovery, err := dsy.NewDiscovery(oct.ActiveEtcdClient(), dsy.SetLogger(lg), dsy.Prefix(prefix))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	gw := &Gateway{
		IEmbedServer:    embedServer,
		ctx:             ctx,
		cancel:          cancel,
		cfg:             cfg,
		oct:             oct,
		discovery:       discovery,
		started:         make(chan struct{}),
		hall:            createHall(),
		handlerWrappers: make([]consumer.HandlerWrapper, 0),
	}

	if cfg.ExtensionEndpoints == nil {
		cfg.ExtensionEndpoints = make([]*dsypb.Endpoint, 0)
	}

	// read openapi docs
	var oapi *openapi.OpenAPI
	if cfg.OpenAPI != "" {
		data, err := os.ReadFile(cfg.OpenAPI)
		if err != nil {
			return nil, fmt.Errorf("read openapi v3 docs: %v", err)
		}

		ext := path.Ext(cfg.OpenAPI)
		switch ext {
		case ".yml", ".yaml":
			// convert yaml to json
			data, err = yaml.YAMLToJSON(data)
			if err != nil {
				return nil, fmt.Errorf("convert openapi v3 docs: %v", err)
			}
		}
		if err = json.Unmarshal(data, &oapi); err != nil {
			return nil, fmt.Errorf("read openapi docs: %v", err)
		}

		// extracts endpoints of OpenAPI docs
		for _, ep := range extractOpenAPIDocs(oapi) {
			cfg.ExtensionEndpoints = append(cfg.ExtensionEndpoints, ep)
		}
	}
	gw.openapiDocs = oapi

	cfg.Config.Context = ctx
	if cfg.Config.UserHandlers == nil {
		cfg.Config.UserHandlers = map[string]http.Handler{}
	}
	cfg.Config.UserHandlers["/metrics"] = promhttp.Handler()

	gwImpl := &gatewayImpl{Gateway: gw}
	routerImpl := &endpointRouterImpl{Gateway: gw}
	serviceRegister := cfg.Config.ServiceRegister
	cfg.Config.ServiceRegister = func(gs *grpc.Server) {
		if serviceRegister != nil {
			serviceRegister(gs)
		}
		pb.RegisterGatewayServer(gs, gwImpl)
		pb.RegisterEndpointRouterServer(gs, routerImpl)
	}
	if cfg.EnableGRPCGateway {
		gatewayRegister := cfg.Config.GRPCGatewayRegister
		cfg.Config.GRPCGatewayRegister = func(ctx context.Context, mux *gwr.ServeMux) error {
			if gatewayRegister != nil {
				if err = gatewayRegister(ctx, mux); err != nil {
					return err
				}
			}
			if err = pb.RegisterGatewayHandlerServer(ctx, mux, gwImpl); err != nil {
				return err
			}
			if err = pb.RegisterEndpointRouterHandlerServer(ctx, mux, routerImpl); err != nil {
				return err
			}
			return nil
		}
	}

	gw.ProxyServer, err = grpcproxy.NewProxyServer(discovery, &cfg.Config)
	if err != nil {
		return nil, err
	}

	if err = gw.installHandler(); err != nil {
		return nil, err
	}

	return gw, nil
}

func (g *Gateway) installInternalHandler() (err error) {
	// http consumer pattern
	hp := path.Join("/", dsypb.ActivityType_ServiceTask.String(), "http")
	hc := consumer.NewHttpConsumer()
	if err = g.addConsumer(hp, hc); err != nil {
		return
	}
	g.hc = hc

	return nil
}

func (g *Gateway) Logger() *zap.Logger {
	return g.cfg.GetLogger()
}

func (g *Gateway) StartNotify() <-chan struct{} {
	return g.started
}

func (g *Gateway) Start(stopc <-chan struct{}) error {
	if g.isStarted() {
		return nil
	}

	if err := g.ProxyServer.Start(stopc); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	g.GoAttach(g.process)
	g.OnDestroy(g.destroy)

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
