// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package client

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
	"google.golang.org/grpc"
)

type MetaRPC interface {
	GetMeta(ctx context.Context) (*pb.Meta, error)
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
