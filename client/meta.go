// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
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
