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

	pb "github.com/olive-io/olive/api/olivepb"
)

type MetaRunnerRPC interface {
	ListRunner(ctx context.Context) (*pb.ListRunnerResponse, error)
	GetRunner(ctx context.Context, id uint64) (*pb.GetRunnerResponse, error)
}

type runnerRPC struct {
	remote   pb.MetaRunnerRPCClient
	callOpts []grpc.CallOption
}

func NewRunnerRPC(c *Client) MetaRunnerRPC {
	conn := c.ActiveConnection()
	api := &runnerRPC{
		remote: RetryMetaRunnerClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rr *runnerRPC) ListRunner(ctx context.Context) (*pb.ListRunnerResponse, error) {
	in := &pb.ListRunnerRequest{}
	resp, err := rr.remote.ListRunner(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp, nil
}

func (rr *runnerRPC) GetRunner(ctx context.Context, id uint64) (*pb.GetRunnerResponse, error) {
	in := &pb.GetRunnerRequest{Id: id}
	resp, err := rr.remote.GetRunner(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp, nil
}
