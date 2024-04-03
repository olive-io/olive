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
	"net"
	"net/http"
	urlpkg "net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gwr "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	json "github.com/json-iterator/go"
	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/gatewaypb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/addr"
	"github.com/olive-io/olive/pkg/backoff"
	cxmd "github.com/olive-io/olive/pkg/context/metadata"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/mnet"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/proxy/server"
	"github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"sigs.k8s.io/yaml"
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

type Gateway struct {
	genericserver.IEmbedServer
	pb.UnsafeGatewayServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg Config

	oct       *client.Client
	discovery dsy.IDiscovery
	serve     *http.Server
	gs        *grpc.Server

	rpc *rServer
	wg  *sync.WaitGroup

	started chan struct{}

	rmu        sync.RWMutex
	handlers   map[string]server.IHandler
	registered bool
	// registry service instance
	rsvc *dsypb.Service
}

func NewGateway(cfg Config) (*Gateway, error) {
	lg := cfg.GetLogger()
	embedServer := genericserver.NewEmbedServer(lg)

	lg.Debug("connect to olive-meta",
		zap.String("endpoints", strings.Join(cfg.Client.Endpoints, ",")))
	oct, err := client.New(cfg.Client)
	if err != nil {
		return nil, err
	}

	prefix := runtime.DefaultRunnerDiscoveryNode
	discovery, err := dsy.NewDiscovery(oct.Client, dsy.SetLogger(lg), dsy.Prefix(prefix))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	gw := &Gateway{
		IEmbedServer: embedServer,
		ctx:          ctx,
		cancel:       cancel,
		cfg:          cfg,
		oct:          oct,
		rpc: &rServer{
			lg:         lg,
			serviceMap: map[string]*service{},
		},
		wg:        new(sync.WaitGroup),
		discovery: discovery,
		handlers:  map[string]server.IHandler{},
		started:   make(chan struct{}),
	}

	var gwmux *gwr.ServeMux
	gw.gs, gwmux, err = gw.buildGRPCServer()
	if err != nil {
		return nil, err
	}

	handler := gw.buildUserHandler()
	mux := gw.createMux(gwmux, handler)
	gw.serve = &http.Server{
		Handler:        genericserver.GRPCHandlerFunc(gw.gs, mux),
		MaxHeaderBytes: DefaultMaxHeaderBytes,
	}

	if err = pb.RegisterTestServiceServerHandler(gw, &TestRPC{}); err != nil {
		return nil, err
	}

	return gw, nil
}

func (g *Gateway) Logger() *zap.Logger {
	return g.cfg.GetLogger()
}

func (g *Gateway) Handle(hdlr server.IHandler) error {
	if err := g.rpc.register(hdlr.Handler()); err != nil {
		return err
	}

	g.rmu.Lock()
	defer g.rmu.Unlock()
	g.handlers[hdlr.Name()] = hdlr
	return nil
}

func (g *Gateway) NewHandler(h any, opts ...server.HandlerOption) server.IHandler {
	handler := newRPCHandler(h, opts...)
	if desc := handler.options.ServiceDesc; desc != nil {
		g.gs.RegisterService(desc, h)
	}

	return handler
}

func (g *Gateway) Start(stopc <-chan struct{}) error {
	if g.isStarted() {
		return nil
	}
	defer g.beStarted()

	lg := g.Logger()

	scheme, ts, err := g.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	g.rmu.Lock()
	g.cfg.ListenURL = scheme + ts.Addr().String()
	g.rmu.Unlock()

	// announce self to the world
	if err = g.register(); err != nil {
		lg.Error("Server register", zap.Error(err))
	}

	g.GoAttach(func() {
		if e1 := g.serve.Serve(ts); e1 != nil {
			g.Logger().Sugar().Errorf("starting gateway server: %v", e1)
		}
	})
	g.GoAttach(g.process)
	g.Destroy(g.destroy)

	<-stopc

	return g.stop()
}

func (g *Gateway) destroy() {}

func (g *Gateway) stop() error {
	if err := g.serve.Shutdown(g.ctx); err != nil {
		return err
	}
	g.IEmbedServer.Shutdown()
	return nil
}

func (g *Gateway) process() {
	lg := g.Logger()
	cfg := g.cfg

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
			if err := g.register(); err != nil {
				lg.Error("Server register", zap.Error(err))
			}
		// wait for exit
		case <-g.StoppingNotify():
			break Loop
		}
	}

	// deregister self
	if err := g.deregister(); err != nil {
		lg.Error("Server deregister", zap.Error(err))
	}
}

func (g *Gateway) createListener() (string, net.Listener, error) {
	cfg := g.cfg
	lg := g.Logger()
	url, err := urlpkg.Parse(cfg.ListenURL)
	if err != nil {
		return "", nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return "", nil, err
	}

	return "http://", listener, nil
}

func (g *Gateway) buildGRPCServer() (*grpc.Server, *gwr.ServeMux, error) {
	sopts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(g.handler),
	}
	gs := grpc.NewServer(sopts...)
	pb.RegisterGatewayServer(gs, g)

	ctx := g.ctx
	gwmux := gwr.NewServeMux()

	if err := pb.RegisterGatewayHandlerServer(ctx, gwmux, g); err != nil {
		return nil, nil, err
	}

	return gs, gwmux, nil
}

func (g *Gateway) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())

	return handler
}

func (g *Gateway) createMux(gwmux *gwr.ServeMux, handler http.Handler) *http.ServeMux {
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

func (g *Gateway) register() error {
	ctx, cancel := context.WithCancel(g.ctx)
	defer cancel()

	lg := g.Logger()

	g.rmu.RLock()
	cfg := g.cfg
	rsvc := g.rsvc
	g.rmu.RUnlock()

	regFunc := func(service *dsypb.Service) error {
		var regErr error

		for i := 0; i < 3; i++ {
			// set the ttl
			ropts := []dsy.RegisterOption{dsy.RegisterTTL(cfg.RegisterTTL)}
			// attempt to register
			if err := g.discovery.Register(ctx, service, ropts...); err != nil {
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

	// make copy of metadata
	md := cxmd.Copy(cfg.Metadata)

	// register service
	node := &dsypb.Node{
		Id:       cfg.Id,
		Address:  mnet.HostPort(saddr, port),
		Metadata: md,
	}
	node.Port, _ = strconv.ParseInt(port, 10, 64)

	node.Metadata["protocol"] = "grpc"

	g.rmu.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, handler := range g.handlers {
		// Only advertise non-internal handlers
		if !handler.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	endpoints := make([]*dsypb.Endpoint, 0, len(handlerList))
	for _, h := range handlerList {
		endpoints = append(endpoints, g.handlers[h].Endpoints()...)
	}

	g.rmu.RUnlock()

	// read openapi docs
	var openapi *dsypb.OpenAPI
	if cfg.OpenAPI != "" {
		data, err := os.ReadFile(cfg.OpenAPI)
		if err != nil {
			return fmt.Errorf("read openapi v3 docs: %v", err)
		}

		ext := path.Ext(cfg.OpenAPI)
		switch ext {
		case ".yml", ".yaml":
			// convert yaml to json
			data, err = yaml.YAMLToJSON(data)
			if err != nil {
				return fmt.Errorf("convert openapi v3 docs: %v", err)
			}
		}
		if err = json.Unmarshal(data, &openapi); err != nil {
			return fmt.Errorf("read openapi docs: %v", err)
		}

		// extracts endpoints of OpenAPI docs
		for _, ep := range extractOpenAPIDocs(openapi) {
			endpoints = append(endpoints, ep)
		}
	}

	svc := &dsypb.Service{
		Name:      api.DefaultService,
		Version:   version.Version,
		Nodes:     []*dsypb.Node{node},
		Endpoints: endpoints,
		Openapi:   openapi,
	}

	g.rmu.RLock()
	registered := g.registered
	g.rmu.RUnlock()

	if !registered {
		lg.Info("registering node", zap.String("id", cfg.Id))
	}

	// register the service
	if err = regFunc(svc); err != nil {
		return err
	}

	// already registered? don't need to register subscribers
	if registered {
		return nil
	}

	g.rmu.Lock()
	defer g.rmu.Unlock()

	g.registered = true
	if cacheService {
		g.rsvc = svc
	}

	return nil
}

func (g *Gateway) deregister() error {
	var err error
	var advt, host, port string

	lg := g.Logger()

	g.rmu.RLock()
	cfg := g.cfg
	g.rmu.RUnlock()

	// check the advertisement address first
	// if it exists then use it, otherwise
	// use the address
	if len(cfg.AdvertiseURL) > 0 {
		advt = cfg.AdvertiseURL
	} else {
		advt = cfg.ListenURL
	}

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
		Id:      cfg.Id,
		Address: mnet.HostPort(nAddr, port),
	}

	svc := &dsypb.Service{
		Name:    api.DefaultService,
		Version: version.Version,
		Nodes:   []*dsypb.Node{node},
	}

	lg.Info("Deregistering node", zap.String("id", node.Id))
	ctx := g.ctx
	if err = g.discovery.Deregister(ctx, svc); err != nil {
		return err
	}

	g.rmu.Lock()
	g.rsvc = nil

	if !g.registered {
		g.rmu.Unlock()
		return nil
	}

	g.registered = false
	g.rmu.Unlock()
	return nil
}

func (g *Gateway) beStarted() {
	select {
	case <-g.started:
		return
	default:
		close(g.started)
	}
}

func (g *Gateway) isStarted() bool {
	select {
	case <-g.started:
		return true
	default:
		return false
	}
}

func (g *Gateway) StartNotify() <-chan struct{} {
	return g.started
}
