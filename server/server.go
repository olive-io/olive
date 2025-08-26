/*
Copyright 2025 The olive Authors

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

package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/gorilla/mux"
	gwrt "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
	"github.com/olive-io/olive/server/dao"
	"github.com/olive-io/olive/server/scheduler"
)

const (
	DefaultMaxHeaderBytes = 1024 * 1024 * 30
)

type Server struct {
	cfg *Config
}

func NewServer(cfg *Config) (*Server, error) {
	server := &Server{
		cfg: cfg,
	}

	return server, nil
}

func (s *Server) Start(ctx context.Context) error {
	cfg := s.cfg
	lg := s.cfg.Logger()

	listenAddr := cfg.ListenAddr
	lg.Info("listening on " + listenAddr)
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen tcp on %s: %w", listenAddr, err)
	}

	handler, err := s.buildHandler(ctx)
	if err != nil {
		return fmt.Errorf("build handler: %w", err)
	}

	hs := &http.Server{
		Handler:           handler,
		ReadTimeout:       time.Minute,
		ReadHeaderTimeout: time.Second * 30,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Second * 30,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
	}

	ech := make(chan error, 1)
	go func() {
		err = hs.Serve(ln)
		if err != nil {
			ech <- err
		}
	}()

	select {
	case err = <-ech:
		return fmt.Errorf("start server: %w", err)
	case <-ctx.Done():
	}

	if err = hs.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	return nil
}

func (s *Server) buildHandler(ctx context.Context) (http.Handler, error) {
	lg := s.cfg.Logger()

	dataDir := s.cfg.DataRoot
	db, err := openLocalDB(lg, dataDir)
	if err != nil {
		return nil, fmt.Errorf("open database on %s: %w", dataDir, err)
	}

	definitionsDao, err := dao.NewDefinitionsDao(db)
	if err != nil {
		return nil, fmt.Errorf("create definitions dao: %w", err)
	}
	processDao, err := dao.NewProcessDao(db)
	if err != nil {
		return nil, fmt.Errorf("create process dao: %w", err)
	}

	schedulerOptions := scheduler.NewOptions(lg)
	sch, err := scheduler.NewScheduler(ctx, schedulerOptions)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	bpmnHandler := newBpmnServer(ctx, lg, sch, definitionsDao, processDao)
	systemHandler := newSystemGRPCServer(ctx, lg)

	kaep := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}
	kasp := keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second,
		MaxConnectionAge:      30 * time.Second,
		MaxConnectionAgeGrace: 5 * time.Second,
		Time:                  5 * time.Second,
		Timeout:               3 * time.Second,
	}

	sopts := []grpc.ServerOption{
		grpc.UnaryInterceptor(validateInterceptor),
		grpc.KeepaliveEnforcementPolicy(kaep),
		grpc.KeepaliveParams(kasp),
	}
	gs := grpc.NewServer(sopts...)

	muxOpts := []gwrt.ServeMuxOption{}
	gwmux := gwrt.NewServeMux(muxOpts...)

	pb.RegisterBpmnRPCServer(gs, bpmnHandler)
	if err = pb.RegisterBpmnRPCHandlerServer(ctx, gwmux, bpmnHandler); err != nil {
		return nil, fmt.Errorf("setup olive bpmn handler: %w", err)
	}

	pb.RegisterSystemRPCServer(gs, systemHandler)
	if err = pb.RegisterSystemRPCHandlerServer(ctx, gwmux, systemHandler); err != nil {
		return nil, fmt.Errorf("setup olive system handler: %w", err)
	}

	serveMux := mux.NewRouter()
	serveMux.Handle("/metrics", promhttp.Handler())

	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	serveMux.PathPrefix("/debug/pprof/").Handler(pprofMux)

	serveMux.Handle("/v1/",
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

	return grpcWithHttp(gs, serveMux), nil
}

func grpcWithHttp(gh *grpc.Server, hh http.Handler) http.Handler {
	h2s := &http2.Server{}
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			gh.ServeHTTP(w, r)
		} else {
			hh.ServeHTTP(w, r)
		}
	}), h2s)
}

func validateInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if impl, ok := req.(interface{ ValidateAll() error }); ok {
		if err := impl.ValidateAll(); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	return handler(ctx, req)
}
