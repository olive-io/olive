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

package server

import (
	"context"

	"github.com/olive-io/olive/api/gatewaypb"
	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

func (gw *Gateway) Ping(ctx context.Context, req *gatewaypb.PingRequest) (*gatewaypb.PingResponse, error) {
	return &gatewaypb.PingResponse{Reply: "pong"}, nil
}

func (gw *Gateway) Transmit(ctx context.Context, req *gatewaypb.TransmitRequest) (*gatewaypb.TransmitResponse, error) {
	lg := gw.Logger()
	lg.Info("transmit executed", zap.String("activity", req.Activity.String()))
	for key, value := range req.Properties {
		lg.Sugar().Infof("%s=%+v", key, value.Value())
	}
	resp := &gatewaypb.TransmitResponse{
		Properties:  map[string]*dsypb.Box{},
		DataObjects: map[string]*dsypb.Box{},
	}
	return resp, nil
}

type testRPC struct {
	gatewaypb.UnsafeTestServiceServer
}

func (t testRPC) Hello(ctx context.Context, req *gatewaypb.HelloRequest) (*gatewaypb.HelloResponse, error) {
	return &gatewaypb.HelloResponse{Reply: "hello world"}, nil
}
