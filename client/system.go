/*
Copyright 2023 The olive Authors

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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/rpc/monpb"
	"github.com/olive-io/olive/api/types"
)

type SystemRPC interface {
	Ping(ctx context.Context) error
	GetCluster(ctx context.Context) (*types.Monitor, error)
	Register(ctx context.Context, runner *types.Runner) (*types.Runner, error)
	Disregister(ctx context.Context, runner *types.Runner) (*types.Runner, error)
	Heartbeat(ctx context.Context, stat *types.RunnerStat) error
	ListRunners(ctx context.Context) ([]*types.Runner, error)
	GetRunner(ctx context.Context, id uint64) (*types.Runner, *types.RunnerStat, error)
}

type systemRPC struct {
	remote   pb.SystemRPCClient
	callOpts []grpc.CallOption
}

func NewSystemRPC(c *Client) SystemRPC {
	conn := c.ActiveConnection()
	api := &systemRPC{
		remote: RetrySystemClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rpc *systemRPC) Ping(ctx context.Context) error {
	in := &pb.PingRequest{}
	_, err := rpc.remote.Ping(ctx, in, rpc.callOpts...)
	return toErr(ctx, err)
}

func (rpc *systemRPC) GetCluster(ctx context.Context) (*types.Monitor, error) {
	in := &pb.GetClusterRequest{}
	resp, err := rpc.remote.GetCluster(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Monitor, nil
}

func (rpc *systemRPC) Register(ctx context.Context, runner *types.Runner) (*types.Runner, error) {
	in := &pb.RegisterRequest{
		Runner: runner,
	}
	resp, err := rpc.remote.Register(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Runner, nil
}

func (rpc *systemRPC) Disregister(ctx context.Context, runner *types.Runner) (*types.Runner, error) {
	in := &pb.DisregisterRequest{
		Runner: runner,
	}
	resp, err := rpc.remote.Disregister(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Runner, nil
}

func (rpc *systemRPC) Heartbeat(ctx context.Context, stat *types.RunnerStat) error {
	in := &pb.HeartbeatRequest{
		Stat: stat,
	}
	_, err := rpc.remote.Heartbeat(ctx, in, rpc.callOpts...)
	return toErr(ctx, err)
}

func (rpc *systemRPC) ListRunners(ctx context.Context) ([]*types.Runner, error) {
	in := &pb.ListRunnersRequest{}
	resp, err := rpc.remote.ListRunners(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Runners, nil
}

func (rpc *systemRPC) GetRunner(ctx context.Context, id uint64) (runner *types.Runner, stat *types.RunnerStat, err error) {
	in := &pb.GetRunnerRequest{Id: id}
	resp, err := rpc.remote.GetRunner(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, nil, toErr(ctx, err)
	}
	return resp.Runner, resp.Stat, nil
}
