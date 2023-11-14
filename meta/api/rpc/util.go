package rpc

import (
	"context"
	"errors"
	"strings"

	"github.com/olive-io/olive/api/rpctypes"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server"
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/mvcc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var toGRPCErrorMap = map[error]error{
	server.ErrNotEnoughStartedMembers: rpctypes.ErrMemberNotEnoughStarted,
	server.ErrLearnerNotReady:         rpctypes.ErrGRPCLearnerNotReady,

	mvcc.ErrCompacted:         rpctypes.ErrGRPCCompacted,
	mvcc.ErrFutureRev:         rpctypes.ErrGRPCFutureRev,
	server.ErrRequestTooLarge: rpctypes.ErrGRPCRequestTooLarge,
	server.ErrNoSpace:         rpctypes.ErrGRPCNoSpace,
	server.ErrTooManyRequests: rpctypes.ErrTooManyRequests,

	server.ErrNoLeader:                   rpctypes.ErrGRPCNoLeader,
	server.ErrNotLeader:                  rpctypes.ErrGRPCNotLeader,
	server.ErrLeaderChanged:              rpctypes.ErrGRPCLeaderChanged,
	server.ErrStopped:                    rpctypes.ErrGRPCStopped,
	server.ErrTimeout:                    rpctypes.ErrGRPCTimeout,
	server.ErrTimeoutDueToLeaderFail:     rpctypes.ErrGRPCTimeoutDueToLeaderFail,
	server.ErrTimeoutDueToConnectionLost: rpctypes.ErrGRPCTimeoutDueToConnectionLost,
	server.ErrTimeoutWaitAppliedIndex:    rpctypes.ErrGRPCTimeoutWaitAppliedIndex,
	server.ErrUnhealthy:                  rpctypes.ErrGRPCUnhealthy,
	server.ErrKeyNotFound:                rpctypes.ErrGRPCKeyNotFound,
	server.ErrCorrupt:                    rpctypes.ErrGRPCCorrupt,
	server.ErrBadLeaderTransferee:        rpctypes.ErrGRPCBadLeaderTransferee,

	server.ErrClusterVersionUnavailable:     rpctypes.ErrGRPCClusterVersionUnavailable,
	server.ErrWrongDowngradeVersionFormat:   rpctypes.ErrGRPCWrongDowngradeVersionFormat,
	server.ErrInvalidDowngradeTargetVersion: rpctypes.ErrGRPCInvalidDowngradeTargetVersion,
	server.ErrDowngradeInProcess:            rpctypes.ErrGRPCDowngradeInProcess,
	server.ErrNoInflightDowngrade:           rpctypes.ErrGRPCNoInflightDowngrade,

	lease.ErrLeaseNotFound:    rpctypes.ErrGRPCLeaseNotFound,
	lease.ErrLeaseExists:      rpctypes.ErrGRPCLeaseExist,
	lease.ErrLeaseTTLTooLarge: rpctypes.ErrGRPCLeaseTTLTooLarge,

	// In sync with status.FromContextError
	context.Canceled:         rpctypes.ErrGRPCCanceled,
	context.DeadlineExceeded: rpctypes.ErrGRPCDeadlineExceeded,
}

func togRPCError(err error) error {
	// let gRPC server convert to codes.Canceled, codes.DeadlineExceeded
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	grpcErr, ok := toGRPCErrorMap[err]
	if !ok {
		return status.Error(codes.Unknown, err.Error())
	}
	return grpcErr
}

func isClientCtxErr(ctxErr error, err error) bool {
	if ctxErr != nil {
		return true
	}

	ev, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch ev.Code() {
	case codes.Canceled, codes.DeadlineExceeded:
		// client-side context cancel or deadline exceeded
		// "rpc error: code = Canceled desc = context canceled"
		// "rpc error: code = DeadlineExceeded desc = context deadline exceeded"
		return true
	case codes.Unavailable:
		msg := ev.Message()
		// client-side context cancel or deadline exceeded with TLS ("http2.errClientDisconnected")
		// "rpc error: code = Unavailable desc = client disconnected"
		if msg == "client disconnected" {
			return true
		}
		// "grpc/transport.ClientTransport.CloseStream" on canceled streams
		// "rpc error: code = Unavailable desc = stream error: stream ID 21; CANCEL")
		if strings.HasPrefix(msg, "stream error: ") && strings.HasSuffix(msg, "; CANCEL") {
			return true
		}
	}
	return false
}

func isRPCSupportedForLearner(req interface{}) bool {
	switch r := req.(type) {
	case *pb.StatusRequest:
		return true
	case *pb.RangeRequest:
		return r.Serializable
	default:
		return false
	}
}
