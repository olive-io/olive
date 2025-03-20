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

	pb "github.com/olive-io/olive/api/rpc/consolepb"
	"github.com/olive-io/olive/console/service/system"
)

type SystemRPC struct {
	pb.UnimplementedSystemRPCServer

	s *system.Service
}

func NewSystemRPC(s *system.Service) *SystemRPC {
	rpc := &SystemRPC{s: s}
	return rpc
}

func (rpc *SystemRPC) ListRunners(ctx context.Context, req *pb.ListRunnersRequest) (*pb.ListRunnersResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	runners, err := rpc.s.ListRunners(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListRunnersResponse{
		Runners: runners,
	}
	return resp, nil
}

func (rpc *SystemRPC) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (*pb.GetRunnerResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
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

func (rpc *SystemRPC) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	result, err := rpc.s.ListUsers(ctx, req.Page, req.Size, req.Name, req.Email, req.Mobile)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListUsersResponse{
		Users: result.List,
		Total: result.Total,
	}
	return resp, nil
}
