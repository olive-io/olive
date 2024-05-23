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

package grpc

import (
	"context"
	"net/http"
	"time"

	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	dsypb "github.com/olive-io/olive/apis/pb/discovery"

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

	ListenURL    string `json:"listen-url"`
	AdvertiseURL string `json:"advertise-url"`
	// The interval on which to register
	RegisterInterval time.Duration `json:"register-interval"`
	// The register expiry time
	RegisterTTL time.Duration `json:"register-ttl"`

	// UserHandlers is for registering users handlers and only used for
	// embedding proxy into other applications.
	// The map key is the route path for the handler.
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

	ExtensionEndpoints []*dsypb.Endpoint `json:"-"`

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
