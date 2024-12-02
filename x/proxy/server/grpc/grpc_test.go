/*
   Copyright 2024 The olive Authors

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

package grpc_test

import (
	"context"
	"net/http"
	"testing"

	gw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/client"
	dsy "github.com/olive-io/olive/x/discovery"
	"github.com/olive-io/olive/x/logutil"
	grpcproxy "github.com/olive-io/olive/x/proxy/server/grpc"
	"github.com/olive-io/olive/x/runtime"
	genericserver "github.com/olive-io/olive/x/server"
)

var DefaultEndpoints = []string{"http://127.0.0.1:4379"}

type TestRPC struct {
	pb.UnsafeTestServiceServer
}

func (t *TestRPC) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Reply: "hello world"}, nil
}

func TestStartProxy(t *testing.T) {
	logging := logutil.NewLogConfig()

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = logging.GetLogger()

	oct, err := client.New(clientCfg)
	if err != nil {
		t.Fatal(err)
	}

	prefix := runtime.DefaultRunnerDiscoveryNode
	discovery, err := dsy.NewDiscovery(oct.Client, dsy.SetLogger(logging.GetLogger()), dsy.Prefix(prefix))
	if err != nil {
		t.Fatal(err)
	}

	stopc := genericserver.SetupSignalHandler()
	cfg := grpcproxy.NewConfig()
	cfg.UserHandlers = map[string]http.Handler{
		"/metrics": promhttp.Handler(),
	}
	cfg.ServiceRegister = func(server *grpc.Server) {
		pb.RegisterTestServiceServer(server, &TestRPC{})
	}
	cfg.GRPCGatewayRegister = func(ctx context.Context, mux *gw.ServeMux) error {

		return nil
	}
	ps, err := grpcproxy.NewProxyServer(discovery, &cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err = ps.Start(stopc); err != nil {
		t.Fatal(err)
	}
}
