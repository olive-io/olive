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
	"fmt"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	meta "github.com/olive-io/olive/pkg/context/metadata"
	"github.com/olive-io/olive/pkg/proxy/server"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
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

func (g *Gateway) handler(svc interface{}, stream grpc.ServerStream) error {
	if g.wg != nil {
		g.wg.Add(1)
		defer g.wg.Done()
	}

	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Errorf(codes.Internal, "method does not exist in context")
	}

	serviceName, methodName, err := serverMethod(fullMethod)
	if err != nil {
		return status.New(codes.InvalidArgument, err.Error()).Err()
	}

	// get grpc metadata
	gmd, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		gmd = metadata.MD{}
	}

	// copy the metadata to vine.metadata
	md := meta.Metadata{}
	for k, v := range gmd {
		md.Set(k, strings.Join(v, ", "))
	}

	// timeout for server deadline
	to, _ := md.Get("timeout")

	// get content type
	ct := DefaultContentType
	if ctype, ok := md.Get("x-content-type"); ok {
		ct = ctype
	}
	if ctype, ok := md.Get("content-type"); ok {
		ct = ctype
	}

	md.Delete("x-content-type")
	md.Delete("timeout")

	// create new context
	ctx := meta.NewContext(stream.Context(), md)

	// get peer from context
	if p, ok := peer.FromContext(stream.Context()); ok {
		md.Set("Remote", p.Addr.String())
		ctx = peer.NewContext(ctx, p)
	}

	// set the timeout if we have it
	if len(to) > 0 {
		if n, err := strconv.ParseUint(to, 10, 64); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
			defer cancel()
		}
	}

	// process the standard request flow
	g.rpc.mu.Lock()
	s := g.rpc.serviceMap[serviceName]
	g.rpc.mu.Unlock()

	if s == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s", serviceName)).Err()
	}

	mtype := s.method[methodName]
	if mtype == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s.%s", serviceName, methodName)).Err()
	}

	// process unary
	if !mtype.stream {
		return g.processRequest(stream, s, mtype, ct, ctx)
	}

	// process stream
	return g.processStream(stream, s, mtype, ct, ctx)
}

func (g *Gateway) processRequest(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	log := g.Logger().Sugar()
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Errorf(codes.Internal, "method does not exist in context")
	}

	serviceName, methodName, err := serverMethod(fullMethod)
	if err != nil {
		return status.New(codes.InvalidArgument, err.Error()).Err()
	}

	for {
		var argv reflect.Value

		// Decode the argument value.
		argIsValue := false // if true, need to indirect before calling.
		if mtype.ArgType.Kind() == reflect.Ptr {
			argv = reflect.New(mtype.ArgType.Elem())
		} else {
			argv = reflect.New(mtype.ArgType)
			argIsValue = true
		}

		if argIsValue {
			argv = argv.Elem()
		}

		var argvi interface{}
		switch ct {
		case "application/proto",
			"application/protobuf",
			"application/octet-stream",
			"application/grpc",
			"application/grpc+proto":
			argvi = argv.Interface()
		case "application/json", "application/grpc+json":
			vv := argv.Interface()
			argvi = &vv
		}

		// Unmarshal request
		if err := stream.RecvMsg(argvi); err != nil {
			return err
		}

		// reply value

		function := mtype.method.Func
		var returnValues []reflect.Value

		cc, err := g.newGRPCCodec(ct)
		if err != nil {
			return err
		}
		b, err := cc.Marshal(argvi)
		if err != nil {
			return err
		}

		codec := &grpcCodec{
			method:   fmt.Sprintf("%s.%s", serviceName, methodName),
			endpoint: fmt.Sprintf("%s.%s", serviceName, methodName),
			target:   g.cfg.Id,
			s:        stream,
			c:        cc,
		}

		// create a client.Request
		r := &rpcRequest{
			service:     g.cfg.Id,
			method:      fmt.Sprintf("%s.%s", service.name, mtype.method.Name),
			contentType: ct,
			codec:       codec,
			body:        b,
			payload:     argvi,
		}

		// define the handler func
		fn := func(ctx context.Context, req server.Request) (rsp any, err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Error("panic recovered: ", r)
					log.Error(string(debug.Stack()))
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			returnValues = function.Call([]reflect.Value{service.rcvr, mtype.prepareContext(ctx), reflect.ValueOf(argv.Interface())})

			if len(returnValues) != mtype.NumOut {
				return nil, fmt.Errorf("method %v of %v has wrong number of ins: (%d!=%d)", methodName, serviceName, len(returnValues), mtype.NumOut)
			}
			// The return value for the method is an error.
			if rerr := returnValues[1].Interface(); rerr != nil {
				err = rerr.(error)
			}

			return returnValues[0].Interface(), err
		}

		statusCode := codes.OK
		statusDesc := ""
		// execute the handler
		rsp, appErr := fn(ctx, r)
		if appErr != nil {
			var errStatus *status.Status
			switch verr := appErr.(type) {
			case proto.Message:
				// user defined error that proto based we can attach it to grpc status
				statusCode = convertCode(appErr)
				statusDesc = appErr.Error()
				errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
				if err != nil {
					return err
				}
			default:
				// default case user pass own error type that not proto based
				statusCode = convertCode(verr)
				statusDesc = verr.Error()
				errStatus = status.New(statusCode, statusDesc)
			}

			return errStatus.Err()
		}

		if err := stream.SendMsg(rsp); err != nil {
			return err
		}

		return status.New(statusCode, statusDesc).Err()
	}
}

func (g *Gateway) processStream(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	log := g.Logger().Sugar()
	r := &rpcRequest{
		service:     g.cfg.Id,
		contentType: ct,
		method:      fmt.Sprintf("%s.%s", service.name, mtype.method.Name),
		stream:      true,
	}

	var argv reflect.Value
	if mtype.ArgType != nil {
		// Decode the argument value.
		argIsValue := false // if true, need to indirect before calling.
		if mtype.ArgType.Kind() == reflect.Ptr {
			argv = reflect.New(mtype.ArgType.Elem())
		} else {
			argv = reflect.New(mtype.ArgType)
			argIsValue = true
		}

		if argIsValue {
			argv = argv.Elem()
		}

		var argvi interface{}
		switch ct {
		case "application/proto",
			"application/protobuf",
			"application/octet-stream",
			"application/grpc",
			"application/grpc+proto":
			argvi = argv.Interface()
		case "application/json", "application/grpc+json":
			vv := argv.Interface()
			argvi = &vv
		}

		// Unmarshal request
		if err := stream.RecvMsg(argvi); err != nil {
			return err
		}
	}

	ss := &rpcStream{
		request: r,
		s:       stream,
	}

	function := mtype.method.Func
	var returnValues []reflect.Value

	// Invoke the method, providing a new value for the reply.
	fn := func(ctx context.Context, req server.Request, stream interface{}) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered: ", r)
				log.Error(string(debug.Stack()))
				err = fmt.Errorf("panic recovered: %v", r)
			}
		}()

		switch {
		case mtype.NumIn == 2:
			returnValues = function.Call([]reflect.Value{service.rcvr, reflect.ValueOf(argv.Interface()), reflect.ValueOf(stream)})
		default:
			returnValues = function.Call([]reflect.Value{service.rcvr, reflect.ValueOf(stream)})
		}

		if verr := returnValues[0].Interface(); verr != nil {
			return verr.(error)
		}

		return nil
	}

	statusCode := codes.OK
	statusDesc := ""

	if appErr := fn(ctx, r, ss); appErr != nil {
		var err error
		var errStatus *status.Status
		switch verr := appErr.(type) {
		case proto.Message:
			// user defined error that proto based we can attach it to grpc status
			statusCode = convertCode(appErr)
			statusDesc = appErr.Error()
			errStatus, err = status.New(statusCode, statusDesc).WithDetails(verr)
			if err != nil {
				return err
			}
		default:
			// default case user pass own error type that not proto based
			statusCode = convertCode(verr)
			statusDesc = verr.Error()
			errStatus = status.New(statusCode, statusDesc)
		}
		return errStatus.Err()
	}

	return status.New(statusCode, statusDesc).Err()
}

func (g *Gateway) newGRPCCodec(ct string) (encoding.Codec, error) {
	codecs := make(map[string]encoding.Codec)
	if c, ok := codecs[ct]; ok {
		return c, nil
	}
	if c, ok := defaultGRPCCodecs[ct]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unsupported Content-Type: %s", ct)
}
