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

	"go.uber.org/zap"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
)

var _ pb.SystemRPCServer = (*systemGRPCServer)(nil)

type systemGRPCServer struct {
	pb.UnimplementedSystemRPCServer

	ctx context.Context
	lg  *zap.Logger
}

func newSystemGRPCServer(ctx context.Context, lg *zap.Logger) *systemGRPCServer {
	server := &systemGRPCServer{
		ctx: ctx,
		lg:  lg,
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
	//TODO implement me
	panic("implement me")
}

func (sgs *systemGRPCServer) Disregister(ctx context.Context, req *pb.DisregisterRequest) (*pb.DisregisterResponse, error) {
	//TODO implement me
	panic("implement me")
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
