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
	urlpkg "net/url"
	"path"
	"runtime/debug"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/gateway/consumer"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/proxy/api"
)

type gatewayImpl struct {
	pb.UnsafeGatewayServer
	*Gateway
}

func (g *gatewayImpl) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{Reply: "pong"}, nil
}

func (g *gatewayImpl) Transmit(ctx context.Context, req *pb.TransmitRequest) (*pb.TransmitResponse, error) {
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
				lg.Sugar().Error(err)
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

func (g *gatewayImpl) createTransmitContext(parent context.Context, r *pb.TransmitRequest) *consumer.Context {
	headers := map[string]string{}
	for k, v := range r.Headers {
		headers[api.OliveHttpKey(k)] = v
	}

	ctx := &consumer.Context{
		Context:     parent,
		Headers:     headers,
		Properties:  r.Properties,
		DataObjects: r.DataObjects,
	}
	return ctx
}

type endpointRouterImpl struct {
	pb.UnsafeEndpointRouterServer
	*Gateway
}

func (g *endpointRouterImpl) Inject(ctx context.Context, req *pb.InjectRequest) (*pb.InjectResponse, error) {
	resp := &pb.InjectResponse{}

	yard := req.Yard
	if yard == nil || yard.Id == "" {
		return nil, status.Newf(codes.InvalidArgument, "invalid yard").Err()
	}

	url, err := urlpkg.Parse(yard.Address)
	if err != nil {
		return nil, status.Newf(codes.InvalidArgument, "invalid yard address").Err()
	}

	lg := g.Logger()
	lg.Info("inject yard",
		zap.String("id", yard.Id),
		zap.String("address", url.String()))

	opts := []dsy.InjectOption{dsy.InjectId(yard.Id)}
	for _, cs := range yard.Consumers {
		ep := extractConsumer(cs, g.cfg.Name)
		ep.Metadata[api.HostKey] = yard.Address
		err = g.discovery.Inject(ctx, ep, opts...)
		if err != nil {
			return nil, err
		}
	}

	g.hc.Inject(yard)

	return resp, nil
}

func (g *endpointRouterImpl) DigOut(ctx context.Context, req *pb.DigOutRequest) (*pb.DigOutResponse, error) {
	resp := &pb.DigOutResponse{}
	yard := req.Yard
	if yard == nil || yard.Id == "" {
		return nil, status.Newf(codes.InvalidArgument, "invalid yard").Err()
	}

	lg := g.Logger()
	lg.Info("dig out yard", zap.String("id", yard.Id))
	g.hc.DigOut(yard.Id)

	return resp, nil
}
