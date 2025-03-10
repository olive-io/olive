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

	"github.com/olive-io/olive/api/rpc/runnerpb"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/runner/config"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/storage"
)

func RegisterServer(ctx context.Context, cfg *config.Config, sch *scheduler.Scheduler, gather *gather.Gather, bs storage.Storage) (http.Handler, error) {
	sopts := []grpc.ServerOption{}
	gs := grpc.NewServer(sopts...)

	runnerRPC := NewGRPCRunnerServer(sch, gather, bs)
	runnerpb.RegisterRunnerRPCServer(gs, runnerRPC)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	muxOpts := []gwrt.ServeMuxOption{}
	gwmux := gwrt.NewServeMux(muxOpts...)

	if err := runnerpb.RegisterRunnerRPCHandlerServer(ctx, gwmux, runnerRPC); err != nil {
		return nil, err
	}

	mux.Handle(
		"/v1/",
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
