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

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/clientgo"
)

type streamTransport struct {
	ctx    context.Context
	runner *Runner

	lg *zap.Logger

	client *clientgo.Client

	stream pb.SystemRPC_RunnerConnectionClient

	connected *atomic.Bool

	wmu       sync.RWMutex
	workUnits map[uint64]WorkUnit
}

func (r *Runner) ListenOnStream(ctx context.Context, client *clientgo.Client) error {
	lg := r.cfg.Logger
	stream, err := client.NewConnection(ctx)
	if err != nil {
		return err
	}

	transport := &streamTransport{
		ctx:       ctx,
		runner:    r,
		lg:        lg,
		client:    client,
		stream:    stream,
		connected: atomic.NewBool(false),

		workUnits: make(map[uint64]WorkUnit),
	}

	return transport.serve()
}

func (st *streamTransport) serve() error {
	ctx := st.ctx
	attempts := 0
	var interval time.Duration

	timer := time.NewTimer(interval)
	defer timer.Stop()

	for {
		interval = retryInterval(attempts)
		timer.Reset(interval)

		attempts += 1

		select {
		case <-st.ctx.Done():
			st.lg.Info("olive connections disconnected")
			goto EXIT
		case <-timer.C:
		}

		if !st.connected.Load() {
			st.lg.Info("reconnecting to olive server")
			runner, err := st.connect(ctx)
			if err != nil {
				if isUnavailable(err) {
					st.lg.Error("olive connection is unavailable")
				} else {
					st.lg.Error("connects to olive server error", zap.Error(err))
				}
				st.connected.Store(false)
				continue
			} else {
				// 重连成功，重置连接间隔
				attempts = 0
				st.runner.tr.Store(runner)
			}
		}

		st.lg.Info("receives connection message")
	LOOP:
		for {
			select {
			case <-ctx.Done():
				goto EXIT
			default:
			}

			msg, err := st.stream.Recv()
			if err != nil {
				if isUnavailable(err) {
					st.lg.Error("connection is unavailable")
				} else {
					st.lg.Error("connect to olive system error", zap.Error(err))
				}
				st.connected.Store(false)
				break LOOP
			}

			if req := msg.CallTask; req != nil {
				rsp := st.doWorkUnit(ctx, req)
				_ = st.stream.Send(&pb.RunnerConnectionRequest{CallTask: rsp})
			}
		}
	}
EXIT:

	if derr := st.disconnect(context.Background()); derr != nil {
		st.lg.Error("olive disconnect error", zap.Error(derr))
	}

	return nil
}

func (st *streamTransport) connect(ctx context.Context) (*types.Runner, error) {
	st.lg.Info("connecting to olive server")
	stream, err := st.client.NewConnection(ctx)
	if err != nil {
		return nil, err
	}

	runner := st.runner.tr.Load()
	runner.Transport = types.Runner_GRPCStream

	endpoints := make([]*types.Endpoint, 0)
	for _, endpoint := range st.runner.endpoints {
		endpoints = append(endpoints, endpoint)
	}
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].URL() < endpoints[j].URL() })

	runner, err = st.client.Register(ctx, runner, endpoints)
	if err != nil {
		return nil, err
	}

	if err = stream.Send(&pb.RunnerConnectionRequest{
		Handshake: &pb.HandshakeRequest{Uid: runner.Uid},
	}); err != nil {
		return nil, fmt.Errorf("send handshake request: %w", err)
	}
	st.stream = stream

	st.lg.Info("connect to olive server succeeded")
	st.connected.Store(true)
	return runner, nil
}

func (st *streamTransport) disconnect(ctx context.Context) error {
	st.lg.Info("disconnecting to olive server")

	runner := st.runner.tr.Load()
	_, err := st.client.Disregister(ctx, runner.Uid)
	if err != nil {
		return err
	}
	return nil
}

func (st *streamTransport) doWorkUnit(ctx context.Context, req *pb.CallTaskRequest) *pb.CallTaskResponse {

	sessionId := req.Id
	rsp := &pb.CallTaskResponse{
		Id:          sessionId,
		Results:     map[string]string{},
		DataObjects: map[string]string{},
		Uid:         req.Uid,
	}

	st.lg.Info(fmt.Sprintf("[%d] [%s] work unit [%s]", req.Id, req.Stage.String(), req.Name))

	var err error
	defer func() {
		if err != nil {
			rsp.Error = err.Error()
		}
	}()

	switch req.Stage {
	case pb.CallTaskRequest_Commit:
		opts := []WuOption{
			WithType(req.FlowType),
			WithKind(req.Kind),
			WithID(req.Name),
		}
		workUnit, ok := st.runner.findWorkUnit(opts...)
		if !ok {
			err = status.Error(codes.NotFound, "work unit not found")
			return rsp
		}
		options := workUnit.Options()
		params := reflect.New(options.Request)
		if err = InjectTypeFields(params, req.Properties); err != nil {
			err = status.Error(codes.InvalidArgument, fmt.Sprintf("inject work unit parameters: %v", err))
			return rsp
		}

		if len(req.Headers) > 0 {
			ctx = context.WithValue(ctx, "headers", req.Headers)
		}

		st.addWorkUnit(sessionId, workUnit)

		out, cErr := workUnit.Commit(ctx, params.Interface())
		if cErr != nil {
			err = status.Error(codes.Internal, cErr.Error())
			return rsp
		}

		if out != nil {
			rsp.Results = ExtractTypeFields(out)
		}

	case pb.CallTaskRequest_Rollback:

		workUnit, ok := st.getWorkUnit(sessionId)
		if ok {
			if rerr := workUnit.Rollback(ctx); rerr != nil {

			}
		}

	case pb.CallTaskRequest_Destroy:
		workUnit, ok := st.getWorkUnit(sessionId)
		if ok {
			st.removeWorkUnit(sessionId)
			if derr := workUnit.Destroy(ctx); derr != nil {

			}
		}
	}

	return rsp
}

func (st *streamTransport) addWorkUnit(id uint64, workUnit WorkUnit) {
	st.wmu.Lock()
	defer st.wmu.Unlock()
	st.workUnits[id] = workUnit
}

func (st *streamTransport) removeWorkUnit(id uint64) {
	st.wmu.Lock()
	defer st.wmu.Unlock()
	delete(st.workUnits, id)
}

func (st *streamTransport) getWorkUnit(id uint64) (WorkUnit, bool) {
	st.wmu.RLock()
	defer st.wmu.RUnlock()
	workUnit, ok := st.workUnits[id]
	return workUnit, ok
}

func isUnavailable(err error) bool {
	return err == io.EOF ||
		errors.Is(err, context.Canceled) ||
		status.Code(err) == codes.Unavailable
}

func retryInterval(attempts int) time.Duration {
	if attempts <= 0 {
		return time.Millisecond * 100
	}
	if attempts > 5 {
		return time.Minute
	}
	return time.Duration(attempts*10) * time.Second
}
