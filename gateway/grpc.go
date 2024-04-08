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

	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
)

func (g *Gateway) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Reply: "pong"}, nil
}

func (g *Gateway) Transmit(ctx context.Context, req *pb.TransmitRequest) (*pb.TransmitResponse, error) {
	lg := g.Logger()
	lg.Info("transmit executed", zap.String("activity", req.Activity.String()))
	for key, value := range req.Properties {
		lg.Sugar().Infof("%s=%+v", key, value.Value())
	}
	resp := &pb.TransmitResponse{
		Properties:  map[string]*dsypb.Box{},
		DataObjects: map[string]*dsypb.Box{},
	}
	return resp, nil
}

type TestRPC struct {
	pb.UnsafeTestServiceServer
}

func (t *TestRPC) Hello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{Reply: "hello world"}, nil
}
