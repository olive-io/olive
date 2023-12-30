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
	"net"
	"net/http"
	urlpkg "net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gw "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/execute"
	"github.com/olive-io/olive/pkg/addr"
	"github.com/olive-io/olive/pkg/backoff"
	cxmd "github.com/olive-io/olive/pkg/context/metadata"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/mnet"
	"github.com/olive-io/olive/pkg/runtime"
	genericserver "github.com/olive-io/olive/pkg/server"
	"github.com/olive-io/olive/pkg/version"
)

type Executor struct {
	genericserver.Inner

	ctx    context.Context
	cancel context.CancelFunc

	cfg Config

	oct       *client.Client
	discovery dsy.IDiscovery
	gs        *grpc.Server
	serve     *http.Server

	handlers map[string]execute.IHandler

	started chan struct{}

	rmu        sync.RWMutex
	registered bool
	// registry service instance
	rsvc *dsypb.Service
}

func NewExecutor(cfg Config) (*Executor, error) {
	lg := cfg.GetLogger()
	inner := genericserver.NewInnerServer(lg)

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
	executor := &Executor{
		Inner:     inner,
		ctx:       ctx,
		cancel:    cancel,
		cfg:       cfg,
		oct:       oct,
		discovery: discovery,

		handlers: map[string]execute.IHandler{},

		started: make(chan struct{}),
	}

	return executor, nil
}

func (e *Executor) Logger() *zap.Logger {
	return e.cfg.GetLogger()
}

func (e *Executor) Start(stopc <-chan struct{}) error {
	if e.isStarted() {
		return nil
	}
	defer e.beStarted()

	lg := e.Logger()

	ts, err := e.createListener()
	if err != nil {
		return err
	}

	lg.Info("Server [grpc] Listening", zap.String("addr", ts.Addr().String()))

	e.rmu.Lock()
	e.cfg.ListenURL = "http://" + ts.Addr().String()
	e.rmu.Unlock()

	gs := e.buildGRPCServer()

	gwmux, err := e.buildGRPCGateway()
	if err != nil {
		return err
	}
	handler := e.buildUserHandler()

	mux := e.createMux(gwmux, handler)
	e.serve = &http.Server{
		Handler:        grpcHandlerFunc(gs, mux),
		MaxHeaderBytes: 1024 * 1024 * 20,
	}

	// announce self to the world
	if err = e.register(); err != nil {
		lg.Error("Server register", zap.Error(err))
	}

	e.GoAttach(func() {
		_ = e.serve.Serve(ts)
	})
	e.GoAttach(e.process)
	e.Destroy(e.destroy)

	<-stopc

	return e.stop()
}

func (e *Executor) destroy() {}

func (e *Executor) stop() error {
	if err := e.serve.Shutdown(e.ctx); err != nil {
		return err
	}
	e.Inner.Shutdown()
	return nil
}

func (e *Executor) process() {
	lg := e.Logger()
	cfg := e.cfg

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
			if err := e.register(); err != nil {
				lg.Error("Server register", zap.Error(err))
			}
		// wait for exit
		case <-e.StoppingNotify():
			break Loop
		}
	}

	// deregister self
	if err := e.deregister(); err != nil {
		lg.Error("Server deregister", zap.Error(err))
	}
}

func (e *Executor) createListener() (net.Listener, error) {
	cfg := e.cfg
	lg := e.Logger()
	url, err := urlpkg.Parse(cfg.ListenURL)
	if err != nil {
		return nil, err
	}
	host := url.Host

	lg.Debug("listen on " + host)
	listener, err := net.Listen("tcp", host)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (e *Executor) buildGRPCServer() *grpc.Server {
	sopts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(e.handler),
	}
	gs := grpc.NewServer(sopts...)
	dsypb.RegisterExecutorServer(gs, e)

	return gs
}

func (e *Executor) buildGRPCGateway() (*gw.ServeMux, error) {
	gwmux := gw.NewServeMux()
	if err := dsypb.RegisterExecutorHandlerServer(e.ctx, gwmux, e); err != nil {
		return nil, err
	}
	return gwmux, nil
}

func (e *Executor) buildUserHandler() http.Handler {
	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.Handler())

	return handler
}

func (e *Executor) handler(svc interface{}, stream grpc.ServerStream) error {
	resp := &dsypb.Response{
		Properties: map[string]*dsypb.Box{"a": dsypb.BoxFromT("a")},
	}
	return stream.SendMsg(&dsypb.ExecuteResponse{Response: resp})
}

func (e *Executor) createMux(gwmux *gw.ServeMux, handler http.Handler) *http.ServeMux {
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

func (e *Executor) register() error {
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()

	lg := e.Logger()

	e.rmu.RLock()
	cfg := e.cfg
	rsvc := e.rsvc
	e.rmu.RUnlock()

	regFunc := func(service *dsypb.Service) error {
		var regErr error

		for i := 0; i < 3; i++ {
			// set the ttl
			ropts := []dsy.RegisterOption{dsy.RegisterTTL(cfg.RegisterTTL)}
			// attempt to register
			if err := e.discovery.Register(ctx, service, ropts...); err != nil {
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

	e.rmu.RLock()
	// Maps are ordered randomly, sort the keys for consistency
	var handlerList []string
	for n, handler := range e.handlers {
		// Only advertise non-internal handlers
		if !handler.Options().Internal {
			handlerList = append(handlerList, n)
		}
	}
	sort.Strings(handlerList)

	endpoints := make([]*dsypb.Endpoint, 0, len(handlerList))
	for _, h := range handlerList {
		endpoints = append(endpoints, e.handlers[h].Endpoints()...)
	}
	e.rmu.RUnlock()

	svc := &dsypb.Service{
		Name:      execute.DefaultExecuteName,
		Version:   version.Version,
		Nodes:     []*dsypb.Node{node},
		Endpoints: endpoints,
	}

	e.rmu.RLock()
	registered := e.registered
	e.rmu.RUnlock()

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

	e.rmu.Lock()
	defer e.rmu.Unlock()

	e.registered = true
	if cacheService {
		e.rsvc = svc
	}

	return nil
}

func (e *Executor) deregister() error {
	var err error
	var advt, host, port string

	lg := e.Logger()

	e.rmu.RLock()
	cfg := e.cfg
	e.rmu.RUnlock()

	// check the advertisement address first
	// if it exists then use it, otherwise
	// use the address
	if len(cfg.AdvertiseURL) > 0 {
		advt = cfg.AdvertiseURL
	} else {
		advt = cfg.AdvertiseURL
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
		Name:    execute.DefaultExecuteName,
		Version: version.Version,
		Nodes:   []*dsypb.Node{node},
	}

	lg.Info("Deregistering node", zap.String("id", node.Id))
	if err = e.discovery.Deregister(context.TODO(), svc); err != nil {
		return err
	}

	e.rmu.Lock()
	e.rsvc = nil

	if !e.registered {
		e.rmu.Unlock()
		return nil
	}

	e.registered = false
	e.rmu.Unlock()
	return nil
}

func (e *Executor) beStarted() {
	select {
	case <-e.started:
		return
	default:
		close(e.started)
	}
}

func (e *Executor) isStarted() bool {
	select {
	case <-e.started:
		return true
	default:
		return false
	}
}

func (e *Executor) StartNotify() <-chan struct{} {
	return e.started
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Given in gRPC docs.
func grpcHandlerFunc(gh *grpc.Server, hh http.Handler) http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			gh.ServeHTTP(w, r)
		} else {
			hh.ServeHTTP(w, r)
		}
	}), h2s)
}
