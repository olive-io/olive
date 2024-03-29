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
	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/addr"
	"github.com/olive-io/olive/pkg/backoff"
	cxmd "github.com/olive-io/olive/pkg/context/metadata"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/mnet"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"sigs.k8s.io/yaml"
)

const (
	DefaultMaxHeaderBytes = 1024 * 1024 * 20
)

type Gateway struct {
	genericserver.IEmbedServer

	ctx    context.Context
	cancel context.CancelFunc

	cfg Config

	oct       *client.Client
	discovery dsy.IDiscovery
	serve     *http.Server

	started chan struct{}

	rmu        sync.RWMutex
	handlers   map[string]api.IHandler
	registered bool
	// registry service instance
	rsvc *pb.Service
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
		discovery:    discovery,

		handlers: map[string]api.IHandler{},

		started: make(chan struct{}),
	}

	return gw, nil
}

func (gw *Gateway) Logger() *zap.Logger {
	return gw.cfg.GetLogger()
}

func (gw *Gateway) Start(stopc <-chan struct{}) error {
	if gw.isStarted() {
		return nil
	}
	defer gw.beStarted()

	lg := gw.Logger()

	scheme, ts, err := gw.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	gw.rmu.Lock()
	gw.cfg.ListenURL = scheme + ts.Addr().String()
	gw.rmu.Unlock()

	gs, gwmux, err := gw.buildGRPCServer()
	if err != nil {
		return err
	}

	handler := gw.buildUserHandler()
	mux := gw.createMux(gwmux, handler)
	gw.serve = &http.Server{
		Handler:        genericserver.GRPCHandlerFunc(gs, mux),
		MaxHeaderBytes: DefaultMaxHeaderBytes,
	}

	// announce self to the world
	if err = gw.register(); err != nil {
		lg.Error("Server register", zap.Error(err))
	}

	gw.GoAttach(func() {
		_ = gw.serve.Serve(ts)
	})
	gw.GoAttach(gw.process)
	gw.Destroy(gw.destroy)

	<-stopc

	return gw.stop()
}

func (gw *Gateway) destroy() {}

func (gw *Gateway) stop() error {
	if err := gw.serve.Shutdown(gw.ctx); err != nil {
		return err
	}
	gw.IEmbedServer.Shutdown()
	return nil
}

func (gw *Gateway) process() {
	lg := gw.Logger()
	cfg := gw.cfg

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
			if err := gw.register(); err != nil {
				lg.Error("Server register", zap.Error(err))
			}
		// wait for exit
		case <-gw.StoppingNotify():
			break Loop
		}
	}

	// deregister self
	if err := gw.deregister(); err != nil {
		lg.Error("Server deregister", zap.Error(err))
	}
}

func (gw *Gateway) createListener() (string, net.Listener, error) {
	cfg := gw.cfg
	lg := gw.Logger()
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

func (gw *Gateway) buildGRPCServer() (*grpc.Server, *gwr.ServeMux, error) {
	sopts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(gw.internalHandler),
	}
	gs := grpc.NewServer(sopts...)
	rpc := &gatewayRpc{gw: gw}
	pb.RegisterGatewayServer(gs, rpc)

	gwmux := gwr.NewServeMux()
	if err := pb.RegisterGatewayHandlerServer(gw.ctx, gwmux, rpc); err != nil {
		return nil, nil, err
	}

	return gs, gwmux, nil
}

func (gw *Gateway) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())

	return handler
}

func (gw *Gateway) internalHandler(svc interface{}, stream grpc.ServerStream) error {
	resp := &pb.Response{
		Properties: map[string]*pb.Box{"a": pb.BoxFromAny("a")},
	}
	return stream.SendMsg(&pb.TransmitResponse{Response: resp})
}

func (gw *Gateway) createMux(gwmux *gwr.ServeMux, handler http.Handler) *http.ServeMux {
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

func (gw *Gateway) register() error {
	ctx, cancel := context.WithCancel(gw.ctx)
	defer cancel()

	lg := gw.Logger()

	gw.rmu.RLock()
	cfg := gw.cfg
	rsvc := gw.rsvc
	gw.rmu.RUnlock()

	regFunc := func(service *pb.Service) error {
		var regErr error

		for i := 0; i < 3; i++ {
			// set the ttl
			ropts := []dsy.RegisterOption{dsy.RegisterTTL(cfg.RegisterTTL)}
			// attempt to register
			if err := gw.discovery.Register(ctx, service, ropts...); err != nil {
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
	node := &pb.Node{
		Id:       cfg.Id,
		Address:  mnet.HostPort(saddr, port),
		Metadata: md,
	}
	node.Port, _ = strconv.ParseInt(port, 10, 64)

	node.Metadata["protocol"] = "grpc"

	gw.rmu.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, handler := range gw.handlers {
		// Only advertise non-internal handlers
		if !handler.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	endpoints := make([]*pb.Endpoint, 0, len(handlerList))
	for _, h := range handlerList {
		endpoints = append(endpoints, gw.handlers[h].Endpoints()...)
	}

	gw.rmu.RUnlock()

	// read openapi docs
	var openapi *pb.OpenAPI
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

	svc := &pb.Service{
		Name:      api.DefaultService,
		Version:   version.Version,
		Nodes:     []*pb.Node{node},
		Endpoints: endpoints,
		Openapi:   openapi,
	}

	gw.rmu.RLock()
	registered := gw.registered
	gw.rmu.RUnlock()

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

	gw.rmu.Lock()
	defer gw.rmu.Unlock()

	gw.registered = true
	if cacheService {
		gw.rsvc = svc
	}

	return nil
}

func (gw *Gateway) deregister() error {
	var err error
	var advt, host, port string

	lg := gw.Logger()

	gw.rmu.RLock()
	cfg := gw.cfg
	gw.rmu.RUnlock()

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

	node := &pb.Node{
		Id:      cfg.Id,
		Address: mnet.HostPort(nAddr, port),
	}

	svc := &pb.Service{
		Name:    api.DefaultService,
		Version: version.Version,
		Nodes:   []*pb.Node{node},
	}

	lg.Info("Deregistering node", zap.String("id", node.Id))
	if err = gw.discovery.Deregister(context.TODO(), svc); err != nil {
		return err
	}

	gw.rmu.Lock()
	gw.rsvc = nil

	if !gw.registered {
		gw.rmu.Unlock()
		return nil
	}

	gw.registered = false
	gw.rmu.Unlock()
	return nil
}

func (gw *Gateway) beStarted() {
	select {
	case <-gw.started:
		return
	default:
		close(gw.started)
	}
}

func (gw *Gateway) isStarted() bool {
	select {
	case <-gw.started:
		return true
	default:
		return false
	}
}

func (gw *Gateway) StartNotify() <-chan struct{} {
	return gw.started
}
