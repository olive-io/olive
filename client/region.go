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

type (
	ListRegionResponse pb.ListRegionResponse
	GetRegionResponse  pb.GetRegionResponse
)

type MetaRegionRPC interface {
	ListRegion(ctx context.Context, opts ...PageOption) (*ListRegionResponse, error)
	GetRegion(ctx context.Context, id uint64) (*GetRegionResponse, error)
}

type regionRPC struct {
	client   *Client
	callOpts []grpc.CallOption
}

func NewRegionRPC(c *Client) MetaRegionRPC {
	api := &regionRPC{client: c}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rr *regionRPC) ListRegion(ctx context.Context, opts ...PageOption) (*ListRegionResponse, error) {
	var page PageOptions
	for _, opt := range opts {
		opt(&page)
	}

	conn := rr.client.conn
	in := &pb.ListRegionRequest{
		Limit:    page.Limit,
		Continue: page.Token,
	}

	resp, err := rr.remoteClient(conn).ListRegion(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*ListRegionResponse)(resp), nil
}

func (rr *regionRPC) GetRegion(ctx context.Context, id uint64) (*GetRegionResponse, error) {
	conn := rr.client.conn
	in := &pb.GetRegionRequest{Id: id}
	resp, err := rr.remoteClient(conn).GetRegion(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*GetRegionResponse)(resp), nil
}

func (rr *regionRPC) remoteClient(conn *grpc.ClientConn) pb.MetaRegionRPCClient {
	return RetryMetaRegionClient(conn)
}
