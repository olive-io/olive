// Copyright 2024 The olive Authors
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
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/logutil"
	grpcproxy "github.com/olive-io/olive/pkg/proxy/server/grpc"
	"github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
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
		if err := pb.RegisterTestServiceHandlerServer(ctx, mux, &TestRPC{}); err != nil {
			return err
		}
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
