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

type MetaRPC interface {
	GetMeta(ctx context.Context) (*pb.Meta, error)
	ListRunner(ctx context.Context) ([]*pb.Runner, error)
	GetRunner(ctx context.Context, id uint64) (*pb.Runner, error)
	ListRegion(ctx context.Context) ([]*pb.Region, error)
	GetRegion(ctx context.Context, id uint64) (*pb.Region, error)
}

type metaRPC struct {
	remote   pb.MetaRPCClient
	callOpts []grpc.CallOption
}

func NewMetaRPC(c *Client) MetaRPC {
	conn := c.ActiveConnection()
	api := &metaRPC{
		remote: RetryMetaClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (mc *metaRPC) GetMeta(ctx context.Context) (*pb.Meta, error) {
	in := &pb.GetMetaRequest{}
	rsp, err := mc.remote.GetMeta(ctx, in, mc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Meta, nil
}

func (mc *metaRPC) ListRunner(ctx context.Context) ([]*pb.Runner, error) {
	in := &pb.ListRunnerRequest{}
	rsp, err := mc.remote.ListRunner(ctx, in, mc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Runners, nil
}

func (mc *metaRPC) GetRunner(ctx context.Context, id uint64) (*pb.Runner, error) {
	in := &pb.GetRunnerRequest{Id: id}
	rsp, err := mc.remote.GetRunner(ctx, in, mc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Runner, nil
}

func (mc *metaRPC) ListRegion(ctx context.Context) ([]*pb.Region, error) {
	in := &pb.ListRegionRequest{}
	rsp, err := mc.remote.ListRegion(ctx, in, mc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Regions, nil
}

func (mc *metaRPC) GetRegion(ctx context.Context, id uint64) (*pb.Region, error) {
	in := &pb.GetRegionRequest{Id: id}
	rsp, err := mc.remote.GetRegion(ctx, in, mc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Region, nil
}
