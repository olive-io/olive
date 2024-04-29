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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
)

type MetaRegionRPC interface {
	ListRegion(ctx context.Context) (*pb.ListRegionResponse, error)
	GetRegion(ctx context.Context, id uint64) (*pb.GetRegionResponse, error)
}

type regionRPC struct {
	remote   pb.MetaRegionRPCClient
	callOpts []grpc.CallOption
}

func NewRegionRPC(c *Client) MetaRegionRPC {
	conn := c.ActiveConnection()
	api := &regionRPC{
		remote: RetryMetaRegionClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rr *regionRPC) ListRegion(ctx context.Context) (*pb.ListRegionResponse, error) {
	in := &pb.ListRegionRequest{}
	resp, err := rr.remote.ListRegion(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp, nil
}

func (rr *regionRPC) GetRegion(ctx context.Context, id uint64) (*pb.GetRegionResponse, error) {
	in := &pb.GetRegionRequest{Id: id}
	resp, err := rr.remote.GetRegion(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp, nil
}
