package rpc

import (
	"context"
	"crypto/tls"
	"math"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/client/credentials"
	"github.com/olive-io/olive/server"
	"github.com/olive-io/olive/server/auth"
	"github.com/olive-io/olive/server/mvcc"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	grpcOverheadBytes = 512 * 1024
	maxSendBytes      = math.MaxInt32
)

type KVGetter interface {
	KV() mvcc.IWatchableKV
}

type BackendGetter interface {
	Backend() backend.IBackend
}

type AuthGetter interface {
	AuthInfoFromCtx(ctx context.Context) (*auth.AuthInfo, error)
	AuthStore() auth.AuthStore
}

func Server(s *server.KVServer, tls *tls.Config, interceptor grpc.UnaryServerInterceptor, gopts ...grpc.ServerOption) *grpc.Server {
	var opts []grpc.ServerOption
	encoding.RegisterCodec(&codec{})
	if tls != nil {
		bundle := credentials.NewBundle(credentials.Config{TLSConfig: tls})
		opts = append(opts, grpc.Creds(bundle.TransportCredentials()))
	}
	chainUnaryInterceptors := []grpc.UnaryServerInterceptor{
		newLogUnaryInterceptor(s),
		newUnaryInterceptor(s),
		grpc_prometheus.UnaryServerInterceptor,
	}
	if interceptor != nil {
		chainUnaryInterceptors = append(chainUnaryInterceptors, interceptor)
	}

	chainStreamInterceptors := []grpc.StreamServerInterceptor{
		newStreamInterceptor(s),
		grpc_prometheus.StreamServerInterceptor,
	}

	if s.ExperimentalEnableDistributedTracing {
		chainUnaryInterceptors = append(chainUnaryInterceptors, otelgrpc.UnaryServerInterceptor(s.ExperimentalTracerOptions...))
		chainStreamInterceptors = append(chainStreamInterceptors, otelgrpc.StreamServerInterceptor(s.ExperimentalTracerOptions...))

	}

	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(chainUnaryInterceptors...)))
	opts = append(opts, grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(chainStreamInterceptors...)))

	opts = append(opts, grpc.MaxRecvMsgSize(int(s.MaxRequestBytes+grpcOverheadBytes)))
	opts = append(opts, grpc.MaxSendMsgSize(maxSendBytes))
	opts = append(opts, grpc.MaxConcurrentStreams(s.MaxConcurrentStreams))

	grpcServer := grpc.NewServer(append(opts, gopts...)...)

	pb.RegisterKVServer(grpcServer, NewKVServer(s))
	pb.RegisterWatchServer(grpcServer, NewWatchServer(s))
	//pb.RegisterLeaseServer(grpcServer, NewQuotaLeaseServer(s))
	//pb.RegisterClusterServer(grpcServer, NewClusterServer(s))
	//pb.RegisterAuthServer(grpcServer, NewAuthServer(s))
	//pb.RegisterMaintenanceServer(grpcServer, NewMaintenanceServer(s))

	// server should register all the services manually
	// use empty service name for all etcd services' health status,
	// see https://github.com/grpc/grpc/blob/master/doc/health-checking.md for more
	hsrv := health.NewServer()
	hsrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, hsrv)

	// set zero values for metrics registered for this grpc server
	grpc_prometheus.Register(grpcServer)

	return grpcServer
}
