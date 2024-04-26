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
	"fmt"
	"net"
	"net/http"
	urlpkg "net/url"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/api/version"

	"github.com/olive-io/olive/pkg/addr"
	"github.com/olive-io/olive/pkg/backoff"
	cxmd "github.com/olive-io/olive/pkg/context/metadata"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/mnet"
	"github.com/olive-io/olive/pkg/proxy/server"
	genericserver "github.com/olive-io/olive/pkg/server"
)

var (
	// DefaultMaxMsgSize define maximum message size that server can send
	// or receive. Default value is 100MB.
	DefaultMaxMsgSize     = 1024 * 1024 * 100
	DefaultContentType    = "application/grpc"
	DefaultMaxHeaderBytes = 1024 * 1024 * 20
)

func init() {
	encoding.RegisterCodec(wrapCodec{jsonCodec{}})
	encoding.RegisterCodec(wrapCodec{protoCodec{}})
	encoding.RegisterCodec(wrapCodec{bytesCodec{}})
}

type ProxyServer struct {
	cfg *Config

	discovery dsy.IDiscovery

	serve *http.Server
	gs    *grpc.Server

	rpc *rServer
	wg  *sync.WaitGroup

	pmu      sync.RWMutex
	handlers map[string]server.IHandler
	// marks the serve as started
	started bool
	// used for first registration
	registered bool
	// registry service instance
	rsvc *dsypb.Service
}

func NewProxyServer(discovery dsy.IDiscovery, cfg *Config) (*ProxyServer, error) {

	lg := cfg.logger
	ps := &ProxyServer{
		cfg:       cfg,
		discovery: discovery,
		rpc: &rServer{
			lg:         lg,
			serviceMap: map[string]*service{},
		},
		wg:       new(sync.WaitGroup),
		handlers: map[string]server.IHandler{},
	}

	var err error
	var gwmux *gwr.ServeMux
	ps.gs, gwmux, err = ps.buildGRPCServer()
	if err != nil {
		return nil, err
	}

	handler := ps.buildUserHandler()
	mux := ps.createMux(gwmux, handler)
	ps.serve = &http.Server{
		Handler:        genericserver.GRPCHandlerFunc(ps.gs, mux),
		MaxHeaderBytes: DefaultMaxHeaderBytes,
	}

	return ps, nil
}

func (ps *ProxyServer) Handle(hdlr server.IHandler) error {
	if err := ps.rpc.register(hdlr.Handler()); err != nil {
		return err
	}

	ps.pmu.Lock()
	defer ps.pmu.Unlock()
	ps.handlers[hdlr.Name()] = hdlr
	return nil
}

func (ps *ProxyServer) NewHandler(h any, opts ...server.HandlerOption) server.IHandler {
	handler := newRPCHandler(h, opts...)
	if desc := handler.options.ServiceDesc; desc != nil {
		ps.gs.RegisterService(desc, h)
	}

	return handler
}

func (ps *ProxyServer) Logger() *zap.Logger {
	return ps.cfg.logger
}

func (ps *ProxyServer) Start(stopc <-chan struct{}) error {
	ps.pmu.RLock()
	if ps.started {
		ps.pmu.RUnlock()
		return nil
	}
	ps.pmu.RUnlock()

	lg := ps.Logger()

	scheme, ts, err := ps.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	ps.pmu.Lock()
	ps.cfg.ListenURL = scheme + "//" + ts.Addr().String()
	ps.pmu.Unlock()

	// announce self to the world
	if err = ps.register(); err != nil {
		lg.Error("Server register", zap.Error(err))
	}

	go func() {
		if e1 := ps.serve.Serve(ts); e1 != nil {
			ps.Logger().Sugar().Errorf("starting server: %v", e1)
		}
	}()

	go func() {
		cfg := ps.cfg

		t := new(time.Ticker)

		// only process if it exists
		if cfg.RegisterInterval > time.Duration(0) {
			// new ticker
			t = time.NewTicker(cfg.RegisterInterval)
		}

	Loop:
		for {
			select {
			case <-t.C:
				if err := ps.register(); err != nil {
					lg.Error("Server register", zap.Error(err))
				}
			// wait for exit
			case <-stopc:
				break Loop
			}
		}

		// deregister self
		if err := ps.deregister(); err != nil {
			lg.Error("Server deregister", zap.Error(err))
		}

		// wait for waitgroup
		if ps.wg != nil {
			ps.wg.Wait()
		}

		// stop the grpc server
		exit := make(chan bool)

		go func() {
			ps.gs.GracefulStop()
			close(exit)
		}()

		select {
		case <-exit:
		case <-time.After(time.Second):
			ps.gs.Stop()
		}
	}()

	// mark the server as started
	ps.pmu.Lock()
	ps.started = true
	ps.pmu.Unlock()

	return nil
}

func (ps *ProxyServer) createListener() (string, net.Listener, error) {
	lg := ps.Logger()

	ps.pmu.RLock()
	listenURL := ps.cfg.ListenURL
	ps.pmu.RUnlock()
	url, err := urlpkg.Parse(listenURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return url.Scheme, listener, nil
}

func (ps *ProxyServer) buildGRPCServer() (*grpc.Server, *gwr.ServeMux, error) {
	sopts := []grpc.ServerOption{
		grpc.MaxSendMsgSize(ps.getMaxMsgSize()),
		grpc.MaxRecvMsgSize(ps.getMaxMsgSize()),
		grpc.UnknownServiceHandler(ps.handler),
	}
	gsvc := grpc.NewServer(sopts...)
	reflection.Register(gsvc)
	if register := ps.cfg.ServiceRegister; register != nil {
		register(gsvc)
	}

	ctx := ps.cfg.Context
	gwmux := gwr.NewServeMux()
	if register := ps.cfg.GRPCGatewayRegister; register != nil {
		if err := register(ctx, gwmux); err != nil {
			return gsvc, gwmux, err
		}
	}

	return gsvc, gwmux, nil
}

func (ps *ProxyServer) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	for pattern, h := range ps.cfg.UserHandlers {
		handler.Handle(pattern, h)
	}

	return handler
}

func (ps *ProxyServer) createMux(gwmux *gwr.ServeMux, handler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	if gwmux != nil {
		mux.Handle(
			"/v1/",
			wsproxy.WebsocketProxy(
				gwmux,
				wsproxy.WithRequestMutator(
					// Default to the POST method for streams
					func(_ *http.Request, outgoing *http.Request) *http.Request {
						outgoing.Method = "POST"
						return outgoing
					},
				),
				wsproxy.WithMaxRespBodyBufferSize(0x7fffffff),
			),
		)
	}
	if handler != nil {
		mux.Handle("/", handler)
	}
	return mux
}

func (ps *ProxyServer) handler(svc interface{}, stream grpc.ServerStream) error {
	if ps.wg != nil {
		ps.wg.Add(1)
		defer ps.wg.Done()
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
	md := cxmd.Metadata{}
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
	ctx := cxmd.NewContext(stream.Context(), md)

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
	ps.rpc.mu.Lock()
	s := ps.rpc.serviceMap[serviceName]
	ps.rpc.mu.Unlock()

	if s == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s", serviceName)).Err()
	}

	mtype := s.method[methodName]
	if mtype == nil {
		return status.New(codes.Unimplemented, fmt.Sprintf("unknown service %s.%s", serviceName, methodName)).Err()
	}

	// process unary
	if !mtype.stream {
		return ps.processRequest(stream, s, mtype, ct, ctx)
	}

	// process stream
	return ps.processStream(stream, s, mtype, ct, ctx)
}

func (ps *ProxyServer) processRequest(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	log := ps.Logger().Sugar()
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

		cc, err := ps.newGRPCCodec(ct)
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
			target:   ps.cfg.Name,
			s:        stream,
			c:        cc,
		}

		// create a client.Request
		r := &rpcRequest{
			service:     ps.cfg.Name,
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

func (ps *ProxyServer) processStream(stream grpc.ServerStream, service *service, mtype *methodType, ct string, ctx context.Context) error {
	log := ps.Logger().Sugar()
	r := &rpcRequest{
		service:     ps.cfg.Id,
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

func (ps *ProxyServer) register() error {
	ctx, cancel := context.WithCancel(ps.cfg.Context)
	defer cancel()

	lg := ps.Logger()

	ps.pmu.RLock()
	cfg := ps.cfg
	rsvc := ps.rsvc
	ps.pmu.RUnlock()

	regFunc := func(service *dsypb.Service) error {
		var regErr error

		for i := 0; i < 3; i++ {
			// set the ttl
			ropts := []dsy.RegisterOption{dsy.RegisterTTL(cfg.RegisterTTL)}
			// attempt to register
			if err := ps.discovery.Register(ctx, service, ropts...); err != nil {
				// set the error
				regErr = err
				// backoff the retry
				time.Sleep(backoff.Do(i + 1))
				continue
			}
			// success so nil error
			regErr = nil
			break
		}

		return regErr
	}

	// if service already filled, reuse it and return early
	if rsvc != nil {
		if err := regFunc(rsvc); err != nil {
			return err
		}
		return nil
	}

	var err error
	var advt, host, port string
	var cacheService bool

	// check the advertisement address first
	// if it exists then use it, otherwise
	// use the address
	if len(cfg.AdvertiseURL) > 0 {
		advt = cfg.AdvertiseURL
	} else {
		advt = cfg.ListenURL
	}
	url, err := urlpkg.Parse(advt)
	if err != nil {
		return err
	}
	advt = url.Host

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	if ip := net.ParseIP(host); ip != nil {
		cacheService = true
	}

	saddr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	id := cfg.Id
	sname := cfg.Name

	// make copy of metadata
	md := cxmd.Copy(cfg.Metadata)

	// register service
	node := &dsypb.Node{
		Id:       id,
		Address:  mnet.HostPort(saddr, port),
		Metadata: md,
	}
	node.Port, _ = strconv.ParseInt(port, 10, 64)

	node.Metadata["protocol"] = "grpc"

	ps.pmu.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, handler := range ps.handlers {
		// Only advertise non-internal handlers
		if !handler.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	endpoints := make([]*dsypb.Endpoint, 0, len(handlerList))
	for _, h := range handlerList {
		endpoints = append(endpoints, ps.handlers[h].Endpoints()...)
	}
	ps.pmu.RUnlock()

	for _, ep := range cfg.ExtensionEndpoints {
		endpoints = append(endpoints, ep)
	}

	svc := &dsypb.Service{
		Name:      sname,
		Version:   version.Version,
		Nodes:     []*dsypb.Node{node},
		Endpoints: endpoints,
	}

	ps.pmu.RLock()
	registered := ps.registered
	ps.pmu.RUnlock()

	if !registered {
		lg.Info("registering node", zap.String("id", id))
	}

	// register the service
	if err = regFunc(svc); err != nil {
		return err
	}

	if registered {
		return nil
	}

	ps.pmu.Lock()
	defer ps.pmu.Unlock()

	ps.registered = true
	if cacheService {
		ps.rsvc = svc
	}

	return nil
}

func (ps *ProxyServer) deregister() error {
	var err error
	var advt, host, port string

	lg := ps.Logger()

	ps.pmu.RLock()
	cfg := ps.cfg
	ps.pmu.RUnlock()

	// check the advertisement address first
	// if it exists then use it, otherwise
	// use the address
	if len(cfg.AdvertiseURL) > 0 {
		advt = cfg.AdvertiseURL
	} else {
		advt = cfg.ListenURL
	}

	id := cfg.Id
	sname := cfg.Name

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		// ipv6 address in format [host]:port or ipv4 host:port
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return err
		}
	} else {
		host = advt
	}

	nAddr, err := addr.Extract(host)
	if err != nil {
		return err
	}

	node := &dsypb.Node{
		Id:      id,
		Address: mnet.HostPort(nAddr, port),
	}

	svc := &dsypb.Service{
		Name:    sname,
		Version: version.Version,
		Nodes:   []*dsypb.Node{node},
	}

	lg.Info("Deregistering node", zap.String("id", node.Id))
	ctx := ps.cfg.Context
	if err = ps.discovery.Deregister(ctx, svc); err != nil {
		return err
	}

	ps.pmu.Lock()
	ps.rsvc = nil

	if !ps.registered {
		ps.pmu.Unlock()
		return nil
	}

	ps.registered = false
	ps.pmu.Unlock()
	return nil
}

func (ps *ProxyServer) newGRPCCodec(ct string) (encoding.Codec, error) {
	codecs := make(map[string]encoding.Codec)
	if c, ok := codecs[ct]; ok {
		return c, nil
	}
	if c, ok := defaultGRPCCodecs[ct]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unsupported Content-Type: %s", ct)
}

func (ps *ProxyServer) getMaxMsgSize() int {
	return DefaultMaxMsgSize
}
