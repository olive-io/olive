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

	pb "github.com/olive-io/olive/api/pb/olive"
)

type (
	ListRunnerResponse pb.ListRunnerResponse
	GetRunnerResponse  pb.GetRunnerResponse
)

type MetaRunnerRPC interface {
	ListRunner(ctx context.Context, opts ...PageOption) (*ListRunnerResponse, error)
	GetRunner(ctx context.Context, id uint64) (*GetRunnerResponse, error)
}

type runnerRPC struct {
	client   *Client
	callOpts []grpc.CallOption
}

func NewRunnerRPC(c *Client) MetaRunnerRPC {
	api := &runnerRPC{client: c}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rr *runnerRPC) ListRunner(ctx context.Context, opts ...PageOption) (*ListRunnerResponse, error) {
	var page PageOptions
	for _, opt := range opts {
		opt(&page)
	}

	conn := rr.client.conn
	in := &pb.ListRunnerRequest{
		Limit:    page.Limit,
		Continue: page.Token,
	}
	resp, err := rr.remoteClient(conn).ListRunner(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*ListRunnerResponse)(resp), nil
}

func (rr *runnerRPC) GetRunner(ctx context.Context, id uint64) (*GetRunnerResponse, error) {
	conn := rr.client.conn
	in := &pb.GetRunnerRequest{Id: id}
	resp, err := rr.remoteClient(conn).GetRunner(ctx, in, rr.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*GetRunnerResponse)(resp), nil
}

func (rr *runnerRPC) remoteClient(conn *grpc.ClientConn) pb.MetaRunnerRPCClient {
	return RetryMetaRunnerClient(conn)
}
