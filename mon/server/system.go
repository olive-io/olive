/*
Copyright 2024 The olive Authors

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

	pb "github.com/olive-io/olive/api/rpc/monpb"
	"github.com/olive-io/olive/mon/service/system"
)

type systemRPC struct {
	pb.UnsafeSystemRPCServer

	s *system.Service
}

func newSystem(s *system.Service) *systemRPC {
	return &systemRPC{s: s}
}

func (rpc *systemRPC) GetCluster(ctx context.Context, req *pb.GetClusterRequest) (*pb.GetClusterResponse, error) {
	header, monitor, err := rpc.s.GetCluster(ctx)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetClusterResponse{
		Header:  header,
		Monitor: monitor,
	}
	return resp, nil
}

func (rpc *systemRPC) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	runner, err := rpc.s.Register(ctx, req.Runner)
	if err != nil {
		return nil, err
	}
	resp := &pb.RegisterResponse{
		Runner: runner,
	}
	return resp, nil
}

func (rpc *systemRPC) Disregister(ctx context.Context, req *pb.DisregisterRequest) (*pb.DisregisterResponse, error) {
	runner, err := rpc.s.Disregister(ctx, req.Runner)
	if err != nil {
		return nil, err
	}
	resp := &pb.DisregisterResponse{
		Runner: runner,
	}
	return resp, nil
}

func (rpc *systemRPC) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	err := rpc.s.Heartbeat(ctx, req.Stat)
	if err != nil {
		return nil, err
	}
	resp := &pb.HeartbeatResponse{}
	return resp, nil
}

func (rpc *systemRPC) ListRunners(ctx context.Context, req *pb.ListRunnersRequest) (*pb.ListRunnersResponse, error) {
	runners, err := rpc.s.ListRunners(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListRunnersResponse{
		Runners: runners,
	}
	return resp, nil
}

func (rpc *systemRPC) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (*pb.GetRunnerResponse, error) {
	runner, stat, err := rpc.s.GetRunner(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &pb.GetRunnerResponse{
		Runner: runner,
		Stat:   stat,
	}
	return resp, nil
}
