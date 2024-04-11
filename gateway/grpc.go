/*
   Copyright 2023 The olive Authors

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

package gateway

import (
	"context"
	"fmt"
	"path"
	"runtime/debug"

	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/gateway/consumer"
	"github.com/olive-io/olive/pkg/proxy/api"
)

func (g *Gateway) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Reply: "pong"}, nil
}

func (g *Gateway) Transmit(ctx context.Context, req *pb.TransmitRequest) (*pb.TransmitResponse, error) {
	lg := g.Logger()
	lg.Debug("transmit executed",
		zap.Stringer("activity", req.Activity))

	act := req.Activity
	prefix := path.Join("/", act.Type.String(), act.TaskType)

	if cid, ok := req.Headers[api.RequestPrefix+"Id"]; ok {
		prefix = path.Join(prefix, cid)
	}

	var handler consumer.IConsumer
	g.walkConsumer(prefix, func(key string, c consumer.IConsumer) bool {
		handler = c

		if key == prefix {
			return true
		}

		//TODO: more selector?

		return false
	})
	if handler == nil {
		return nil, fmt.Errorf("not found handler")
	}

	// define the handler func
	fn := func(ctx *consumer.Context) (rsp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic recovered: %v", r)
				lg.Sugar().Error(err.Error())
				lg.Error(string(debug.Stack()))
			}
		}()

		rsp, err = handler.Handle(ctx)
		return rsp, err
	}

	// wrap the handler func
	for i := len(g.handlerWrappers); i > 0; i-- {
		fn = g.handlerWrappers[i-1](fn)
	}

	cc := g.createTransmitContext(ctx, req)
	out, err := fn(cc)
	if err != nil {
		// TODO: handle error
		return nil, err
	}

	responseBox := dsypb.BoxFromAny(out)
	resp := &pb.TransmitResponse{
		Properties:  map[string]*dsypb.Box{},
		DataObjects: map[string]*dsypb.Box{},
	}
	outs := responseBox.Split()
	for name := range outs {
		resp.Properties[name] = outs[name]
	}

	return resp, nil
}

func (g *Gateway) createTransmitContext(parent context.Context, r *pb.TransmitRequest) *consumer.Context {
	ctx := &consumer.Context{
		Context:     parent,
		Headers:     r.Headers,
		Properties:  r.Properties,
		DataObjects: r.DataObjects,
	}
	return ctx
}
