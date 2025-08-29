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
	"io"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
	"github.com/olive-io/olive/server/dao"
)

var _ pb.SystemRPCServer = (*systemGRPCServer)(nil)

type systemGRPCServer struct {
	pb.UnimplementedSystemRPCServer

	ctx context.Context
	lg  *zap.Logger

	runnerDao *dao.RunnerDao
}

func newSystemGRPCServer(ctx context.Context, lg *zap.Logger, runnerDao *dao.RunnerDao) *systemGRPCServer {
	server := &systemGRPCServer{
		ctx: ctx,
		lg:  lg,

		runnerDao: runnerDao,
	}

	return server
}

func (sgs *systemGRPCServer) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	return &pb.PingResponse{}, nil
}

func (sgs *systemGRPCServer) GetCluster(ctx context.Context, req *pb.GetClusterRequest) (*pb.GetClusterResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sgs *systemGRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	runner := req.GetRunner()
	runner.Online = 1

	value, _ := sgs.runnerDao.GetRunner(ctx, runner.Id, runner.Uid)
	if value != nil {
		runner.Id = value.Id
		runner.OnlineTimestamp = time.Now().UnixMilli()

		sgs.lg.Info("runner online",
			zap.String("uid", runner.Uid),
			zap.String("name", runner.Name),
			zap.String("transport", runner.Transport.String()),
			zap.String("listen", runner.ListenURL))

		err := sgs.runnerDao.UpdateRunner(ctx, runner)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

	} else {
		err := sgs.runnerDao.CreateRunner(ctx, runner)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		sgs.lg.Info("register new runner",
			zap.String("uid", runner.Uid),
			zap.String("name", runner.Name),
			zap.String("transport", runner.Transport.String()),
			zap.String("listen", runner.ListenURL))
	}

	rsp := &pb.RegisterResponse{
		Runner: runner,
	}
	return rsp, nil
}

func (sgs *systemGRPCServer) Disregister(ctx context.Context, req *pb.DisregisterRequest) (*pb.DisregisterResponse, error) {
	runner, err := sgs.runnerDao.GetRunner(ctx, 0, req.Id)
	if err != nil {
		if dao.IsNotFound(err) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	runner.OfflineTimestamp = time.Now().UnixMilli()
	runner.Online = 0

	err = sgs.runnerDao.UpdateRunner(ctx, runner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rsp := &pb.DisregisterResponse{
		Runner: runner,
	}
	return rsp, nil
}

func (sgs *systemGRPCServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sgs *systemGRPCServer) ListRunners(ctx context.Context, req *pb.ListRunnersRequest) (*pb.ListRunnersResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sgs *systemGRPCServer) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (*pb.GetRunnerResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (sgs *systemGRPCServer) RunnerConnection(stream pb.SystemRPC_RunnerConnectionServer) error {

	var listenURL string

	lg := sgs.lg
	ctx := stream.Context()
	peerInfo, ok := peer.FromContext(ctx)
	if ok {
		listenURL = "grpcstream://" + peerInfo.Addr.String()
	}

	rsp, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}

	handshake := rsp.Handshake
	if handshake == nil {
		return status.Errorf(codes.InvalidArgument, "missing handshake message")
	}

	runner, err := sgs.runnerDao.GetRunner(ctx, 0, handshake.Uid)
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}
	runner.ListenURL = listenURL
	_ = sgs.runnerDao.UpdateRunner(ctx, runner)

	reply := &pb.RunnerConnectionResponse{
		Handshake: &pb.HandshakeResponse{},
	}
	if err = stream.Send(reply); err != nil {
		lg.Error("returns handshake response", zap.Error(err))
	}

	lg.Info("add new pipe",
		zap.String("uid", runner.Uid),
		zap.String("listen-url", runner.ListenURL),
		zap.String("hostname", runner.Hostname),
	)

	//err = p.start()

	lg.Info("remove pipe",
		zap.String("uid", runner.Uid),
		zap.String("listen-url", runner.ListenURL),
		zap.String("hostname", runner.Hostname),
	)

	if err != nil {
		return status.Errorf(codes.Internal, "start stream error: %v", err)
	}

	return nil
}
