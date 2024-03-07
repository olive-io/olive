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

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"go.uber.org/zap"
)

func (gw *Gateway) Ping(ctx context.Context, req *dsypb.PingRequest) (*dsypb.PingResponse, error) {
	return &dsypb.PingResponse{Reply: "pong"}, nil
}

func (gw *Gateway) Transmit(ctx context.Context, req *dsypb.TransmitRequest) (*dsypb.TransmitResponse, error) {
	lg := gw.Logger()
	lg.Info("transmit executed", zap.String("activity", req.Activity.String()))
	resp := &dsypb.TransmitResponse{}
	resp.Response = &dsypb.Response{
		Properties:  map[string]*dsypb.Box{},
		DataObjects: map[string]*dsypb.Box{},
	}
	return resp, nil
}
