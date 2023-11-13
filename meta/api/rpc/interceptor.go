package rpc

import (
	"context"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

const (
	maxNoLeaderCnt          = 3
	warnUnaryRequestLatency = 300 * time.Millisecond
	snapshotMethod          = "/serverpb.Maintenance/Snapshot"
)

type streamsMap struct {
	mu      sync.Mutex
	streams map[grpc.ServerStream]struct{}
}

func newUnaryInterceptor(ra *server.Replica) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		//if s.IsMemberExist(s.ID()) && s.IsLearner() && !isRPCSupportedForLearner(req) {
		//	return nil, rpctypes.ErrGPRCNotSupportedForLearner
		//}
		//
		//md, ok := metadata.FromIncomingContext(ctx)
		//if ok {
		//	ver, vs := "unknown", md.Get(rpctypes.MetadataClientAPIVersionKey)
		//	if len(vs) > 0 {
		//		ver = vs[0]
		//	}
		//	clientRequests.WithLabelValues("unary", ver).Inc()
		//
		//	if ks := md[rpctypes.MetadataRequireLeaderKey]; len(ks) > 0 && ks[0] == rpctypes.MetadataHasLeader {
		//		if s.Leader() == types.ID(raft.None) {
		//			return nil, rpctypes.ErrGRPCNoLeader
		//		}
		//	}
		//}

		return handler(ctx, req)
	}
}

func newLogUnaryInterceptor(ra *server.Replica) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		resp, err := handler(ctx, req)
		lg := ra.Logger()
		if lg != nil { // acquire stats if debug level is enabled or request is expensive
			defer logUnaryRequestStats(ctx, lg, info, startTime, req, resp)
		}
		return resp, err
	}
}

func logUnaryRequestStats(ctx context.Context, lg *zap.Logger, info *grpc.UnaryServerInfo, startTime time.Time, req interface{}, resp interface{}) {
	duration := time.Since(startTime)
	var enabledDebugLevel, expensiveRequest bool
	if lg.Core().Enabled(zap.DebugLevel) {
		enabledDebugLevel = true
	}
	if duration > warnUnaryRequestLatency {
		expensiveRequest = true
	}
	if !enabledDebugLevel && !expensiveRequest {
		return
	}
	remote := "No remote client info."
	peerInfo, ok := peer.FromContext(ctx)
	if ok {
		remote = peerInfo.Addr.String()
	}
	responseType := info.FullMethod
	var reqCount, respCount int64
	var reqSize, respSize int
	var reqContent string
	switch _resp := resp.(type) {
	case *pb.RangeResponse:
		_req, ok := req.(*pb.RangeRequest)
		if ok {
			reqCount = 0
			reqSize = _req.XSize()
			reqContent = _req.String()
		}
		if _resp != nil {
			respCount = _resp.Count
			respSize = _resp.XSize()
		}
	case *pb.PutResponse:
		_req, ok := req.(*pb.PutRequest)
		if ok {
			reqCount = 1
			reqSize = _req.XSize()
			reqContent = pb.NewLoggablePutRequest(_req).String()
			// redact value field from request content, see PR #9821
		}
		if _resp != nil {
			respCount = 0
			respSize = _resp.XSize()
		}
	case *pb.DeleteRangeResponse:
		_req, ok := req.(*pb.DeleteRangeRequest)
		if ok {
			reqCount = 0
			reqSize = _req.XSize()
			reqContent = _req.String()
		}
		if _resp != nil {
			respCount = _resp.Deleted
			respSize = _resp.XSize()
		}
	case *pb.TxnResponse:
		_req, ok := req.(*pb.TxnRequest)
		if ok && _resp != nil {
			if _resp.Succeeded { // determine the 'actual' count and size of request based on success or failure
				reqCount = int64(len(_req.Success))
				reqSize = 0
				for _, r := range _req.Success {
					reqSize += r.XSize()
				}
			} else {
				reqCount = int64(len(_req.Failure))
				reqSize = 0
				for _, r := range _req.Failure {
					reqSize += r.XSize()
				}
			}
			reqContent = pb.NewLoggableTxnRequest(_req).String()
			// redact value field from request content, see PR #9821
		}
		if _resp != nil {
			respCount = 0
			respSize = _resp.XSize()
		}
	default:
		reqCount = -1
		reqSize = -1
		respCount = -1
		respSize = -1
	}

	if enabledDebugLevel {
		logGenericRequestStats(lg, startTime, duration, remote, responseType, reqCount, reqSize, respCount, respSize, reqContent)
	} else if expensiveRequest {
		logExpensiveRequestStats(lg, startTime, duration, remote, responseType, reqCount, reqSize, respCount, respSize, reqContent)
	}
}

func logGenericRequestStats(lg *zap.Logger, startTime time.Time, duration time.Duration, remote string, responseType string,
	reqCount int64, reqSize int, respCount int64, respSize int, reqContent string) {
	lg.Debug("request stats",
		zap.Time("start time", startTime),
		zap.Duration("time spent", duration),
		zap.String("remote", remote),
		zap.String("response type", responseType),
		zap.Int64("request count", reqCount),
		zap.Int("request size", reqSize),
		zap.Int64("response count", respCount),
		zap.Int("response size", respSize),
		zap.String("request content", reqContent),
	)
}

func logExpensiveRequestStats(lg *zap.Logger, startTime time.Time, duration time.Duration, remote string, responseType string,
	reqCount int64, reqSize int, respCount int64, respSize int, reqContent string) {
	lg.Warn("request stats",
		zap.Time("start time", startTime),
		zap.Duration("time spent", duration),
		zap.String("remote", remote),
		zap.String("response type", responseType),
		zap.Int64("request count", reqCount),
		zap.Int("request size", reqSize),
		zap.Int64("response count", respCount),
		zap.Int("response size", respSize),
		zap.String("request content", reqContent),
	)
}

func newStreamInterceptor(ra *server.Replica) grpc.StreamServerInterceptor {
	//smap := monitorLeader(s)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		//if s.IsMemberExist(s.ID()) && s.IsLearner() && info.FullMethod != snapshotMethod { // learner does not support stream RPC except Snapshot
		//	return rpctypes.ErrGPRCNotSupportedForLearner
		//}
		//
		//md, ok := metadata.FromIncomingContext(ss.Context())
		//if ok {
		//	ver, vs := "unknown", md.Get(rpctypes.MetadataClientAPIVersionKey)
		//	if len(vs) > 0 {
		//		ver = vs[0]
		//	}
		//	clientRequests.WithLabelValues("stream", ver).Inc()
		//
		//	if ks := md[rpctypes.MetadataRequireLeaderKey]; len(ks) > 0 && ks[0] == rpctypes.MetadataHasLeader {
		//		if s.Leader() == types.ID(raft.None) {
		//			return rpctypes.ErrGRPCNoLeader
		//		}
		//
		//		ctx := newCancellableContext(ss.Context())
		//		ss = serverStreamWithCtx{ctx: ctx, ServerStream: ss}
		//
		//		smap.mu.Lock()
		//		smap.streams[ss] = struct{}{}
		//		smap.mu.Unlock()
		//
		//		defer func() {
		//			smap.mu.Lock()
		//			delete(smap.streams, ss)
		//			smap.mu.Unlock()
		//			// TODO: investigate whether the reason for cancellation here is useful to know
		//			ctx.Cancel(nil)
		//		}()
		//	}
		//}

		return handler(srv, ss)
	}
}

// cancellableContext wraps a context with new cancellable context that allows a
// specific cancellation error to be preserved and later retrieved using the
// Context.Err() function. This is so downstream context users can disambiguate
// the reason for the cancellation which could be from the client (for example)
// or from this interceptor code.
type cancellableContext struct {
	context.Context

	lock         sync.RWMutex
	cancel       context.CancelFunc
	cancelReason error
}

func newCancellableContext(parent context.Context) *cancellableContext {
	ctx, cancel := context.WithCancel(parent)
	return &cancellableContext{
		Context: ctx,
		cancel:  cancel,
	}
}

// Cancel stores the cancellation reason and then delegates to context.WithCancel
// against the parent context.
func (c *cancellableContext) Cancel(reason error) {
	c.lock.Lock()
	c.cancelReason = reason
	c.lock.Unlock()
	c.cancel()
}

// Err will return the preserved cancel reason error if present, and will
// otherwise return the underlying error from the parent context.
func (c *cancellableContext) Err() error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.cancelReason != nil {
		return c.cancelReason
	}
	return c.Context.Err()
}

type serverStreamWithCtx struct {
	grpc.ServerStream

	// ctx is used so that we can preserve a reason for cancellation.
	ctx *cancellableContext
}

func (ssc serverStreamWithCtx) Context() context.Context { return ssc.ctx }

func monitorLeader(ra *server.Replica) *streamsMap {
	smap := &streamsMap{
		streams: make(map[grpc.ServerStream]struct{}),
	}
	//
	//ra.GoAttach(func() {
	//	election := time.Duration(s.RTTMillisecond) * time.Duration(s.ElectionTTL) * time.Millisecond
	//	//noLeaderCnt := 0
	//
	//	for {
	//		select {
	//		case <-s.StoppingNotify():
	//			return
	//		case <-time.After(election):
	//			//if s.Leader() == types.ID(raft.None) {
	//			//	noLeaderCnt++
	//			//} else {
	//			//	noLeaderCnt = 0
	//			//}
	//			//
	//			//// We are more conservative on canceling existing streams. Reconnecting streams
	//			//// cost much more than just rejecting new requests. So we wait until the member
	//			//// cannot find a leader for maxNoLeaderCnt election timeouts to cancel existing streams.
	//			//if noLeaderCnt >= maxNoLeaderCnt {
	//			//	smap.mu.Lock()
	//			//	for ss := range smap.streams {
	//			//		if ssWithCtx, ok := ss.(serverStreamWithCtx); ok {
	//			//			ssWithCtx.ctx.Cancel(rpctypes.ErrGRPCNoLeader)
	//			//			<-ss.Context().Done()
	//			//		}
	//			//	}
	//			//	smap.streams = make(map[grpc.ServerStream]struct{})
	//			//	smap.mu.Unlock()
	//			//}
	//		}
	//	}
	//})

	return smap
}
