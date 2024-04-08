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

package grpc

import (
	"context"
	"net/http"
	"time"

	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/olive-io/olive/pkg/proxy/api"
)

var (
	DefaultId = "grpc-proxy"
)

const (
	DefaultListenURL        = "http://127.0.0.1:5390"
	DefaultRegisterInterval = time.Second * 20
	DefaultRegisterTTL      = time.Second * 30
)

type Config struct {
	Context context.Context

	Id       string            `json:"id"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"-"`
	OpenAPI  string            `json:"openapi"`

	ListenURL    string `json:"listen-url"`
	AdvertiseURL string `json:"advertise-url"`
	// The interval on which to register
	RegisterInterval time.Duration `json:"register-interval"`
	// The register expiry time
	RegisterTTL time.Duration `json:"register-ttl"`

	// UserHandlers is for registering users handlers and only used for
	// embedding etcd into other applications.
	// The map key is the route path for the handler, and
	UserHandlers map[string]http.Handler `json:"-"`
	// ServiceRegister is for registering users' gRPC services. A simple usage example:
	//	cfg := grpc.NewConfig()
	//	cfg.ServerRegister = func(s *grpc.Server) {
	//		pb.RegisterFooServer(s, &fooServer{})
	//		pb.RegisterBarServer(s, &barServer{})
	//	}
	//	grpc.NewProxyServer(discovery, &cfg)
	ServiceRegister func(*grpc.Server) `json:"-"`
	// GRPCGatewayRegister is for registering users' gRPC gateway services.
	GRPCGatewayRegister func(ctx context.Context, mux *gwr.ServeMux) error `json:"-"`

	logger *zap.Logger
}

func NewConfig() Config {
	cfg := Config{
		Context:          context.Background(),
		Id:               DefaultId,
		Name:             api.DefaultService,
		ListenURL:        DefaultListenURL,
		RegisterInterval: DefaultRegisterInterval,
		RegisterTTL:      DefaultRegisterTTL,
		logger:           zap.NewExample(),
	}

	return cfg
}

func (cfg *Config) SetLogger(logger *zap.Logger) {
	cfg.logger = logger
}
