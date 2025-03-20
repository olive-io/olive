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

	"github.com/gorilla/mux"
	gwrt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"

	"github.com/olive-io/olive/api/rpc/consolepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
	"github.com/olive-io/olive/console/docs"
	"github.com/olive-io/olive/console/service/auth"
	"github.com/olive-io/olive/console/service/bpmn"
	"github.com/olive-io/olive/console/service/system"
	genericserver "github.com/olive-io/olive/pkg/server"
)

func RegisterServer(ctx context.Context, cfg *config.Config, oct *client.Client) (http.Handler, error) {

	if err := dao.Init(cfg); err != nil {
		return nil, err
	}

	bpmnService, err := bpmn.NewBpmn(ctx, cfg, oct)
	if err != nil {
		return nil, err
	}
	systemService, err := system.NewSystem(ctx, cfg, oct)
	if err != nil {
		return nil, err
	}
	authService, err := auth.NewAuth(ctx, cfg, oct)
	if err != nil {
		return nil, err
	}

	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)

	muxOpts := []gwrt.ServeMuxOption{}
	gwmux := gwrt.NewServeMux(muxOpts...)

	bpmnRPC := NewBpmnRPC(bpmnService)
	consolepb.RegisterBpmnRPCServer(gs, bpmnRPC)
	if err := consolepb.RegisterBpmnRPCHandlerServer(ctx, gwmux, bpmnRPC); err != nil {
		return nil, err
	}

	systemRPC := NewSystemRPC(systemService)
	consolepb.RegisterSystemRPCServer(gs, systemRPC)
	if err := consolepb.RegisterSystemRPCHandlerServer(ctx, gwmux, systemRPC); err != nil {
		return nil, err
	}

	authRPC := NewAuthRPC(authService)
	consolepb.RegisterAuthRPCServer(gs, authRPC)
	if err := consolepb.RegisterAuthRPCHandlerServer(ctx, gwmux, authRPC); err != nil {
		return nil, err
	}

	serveMux := mux.NewRouter()
	serveMux.Handle("/metrics", promhttp.Handler())

	serveMux.Handle("/v1/",
		wsproxy.WebsocketProxy(
			gwmux,
			wsproxy.WithRequestMutator(
				// Default to the POST method for streams
				func(_ *http.Request, outgoing *http.Request) *http.Request {
					outgoing.Method = "POST"
					return outgoing
				},
			),
			wsproxy.WithMaxRespBodyBufferSize(0x7fffffff),
		),
	)

	serveMux.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		openapiYAML, _ := docs.GetOpenYAML()
		w.WriteHeader(http.StatusOK)
		w.Write(openapiYAML)
	})

	pattern := "/swagger-ui/"
	swaggerFs, err := docs.GetSwagger()
	if err != nil {
		return nil, err
	}
	serveMux.PathPrefix(pattern).Handler(http.StripPrefix(pattern, http.FileServer(http.FS(swaggerFs))))
	serveMux.PathPrefix("/").Handler(gwmux)

	handler := genericserver.HybridHandler(gs, serveMux)

	return handler, nil
}
