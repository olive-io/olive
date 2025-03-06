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
	GetCluster(ctx context.Context) (*types.Monitor, error)
	Registry(ctx context.Context, runner *types.Runner, stat *types.RunnerStat) (*types.Runner, error)
	ListRunners(ctx context.Context) ([]*types.Runner, error)
	GetRunner(ctx context.Context, name string) (*types.Runner, *types.RunnerStat, error)
}

type systemRPC struct {
	remote   pb.SystemRPCClient
	callOpts []grpc.CallOption
}

func NewSystemRPC(c *Client) SystemRPC {
	conn := c.ActiveConnection()
	api := &systemRPC{
		remote: RetryPlaneClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rpc *systemRPC) GetCluster(ctx context.Context) (*types.Monitor, error) {
	in := &pb.GetClusterRequest{}
	resp, err := rpc.remote.GetCluster(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Monitor, nil
}

func (rpc *systemRPC) Registry(ctx context.Context, runner *types.Runner, stat *types.RunnerStat) (*types.Runner, error) {
	in := &pb.RegistryRequest{
		Runner: runner,
		Stat:   stat,
	}
	resp, err := rpc.remote.Registry(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Runner, nil
}

func (rpc *systemRPC) ListRunners(ctx context.Context) ([]*types.Runner, error) {
	in := &pb.ListRunnersRequest{}
	resp, err := rpc.remote.ListRunners(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Runners, nil
}

func (rpc *systemRPC) GetRunner(ctx context.Context, name string) (runner *types.Runner, stat *types.RunnerStat, err error) {
	in := &pb.GetRunnerRequest{Name: name}
	resp, err := rpc.remote.GetRunner(ctx, in, rpc.callOpts...)
	if err != nil {
		return nil, nil, toErr(ctx, err)
	}
	return resp.Runner, resp.Stat, nil
}
