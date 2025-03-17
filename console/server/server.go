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

	gwrt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
	genericserver "github.com/olive-io/olive/pkg/server"
)

func RegisterServer(ctx context.Context, cfg *config.Config, v3cli *client.Client) (http.Handler, error) {

	if err := dao.Init(cfg); err != nil {
		return nil, err
	}

	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)

	muxOpts := []gwrt.ServeMuxOption{}
	gwmux := gwrt.NewServeMux(muxOpts...)

	//runnerpb.RegisterRunnerRPCServer(gs, runnerRPC)
	//if err := runnerpb.RegisterRunnerRPCHandlerServer(ctx, gwmux, runnerRPC); err != nil {
	//	return nil, err
	//}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.Handle("/v1/",
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

	root := http.NewServeMux()
	root.Handle("/", mux)

	handler := genericserver.HybridHandler(gs, root)

	return handler, nil
}
