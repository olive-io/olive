// Copyright 2023 Lack (xingyys@gmail.com).
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
	"encoding/xml"

	"github.com/cockroachdb/errors"
	"github.com/olive-io/bpmn/schema"
	pb "github.com/olive-io/olive/api/olivepb"
	"google.golang.org/grpc"
)

type BpmnRPC interface {
	DeployDefinition(ctx context.Context, id, name string, body []byte) (int64, error)
}

type bpmnRPC struct {
	remote   pb.BpmnRPCClient
	callOpts []grpc.CallOption
}

func NewBpmnRPC(c *Client) BpmnRPC {
	api := &bpmnRPC{remote: RetryBpmnClient(c)}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (bc *bpmnRPC) DeployDefinition(ctx context.Context, id, name string, body []byte) (int64, error) {
	var definitions schema.Definitions

	if err := xml.Unmarshal(body, &definitions); err != nil {
		return 0, errors.Wrap(ErrBadFormatDefinitions, err.Error())
	}

	r := &pb.DeployDefinitionRequest{
		Id:      id,
		Name:    name,
		Content: body,
	}
	resp, err := bc.remote.DeployDefinition(ctx, r, bc.callOpts...)
	if err != nil {
		return 0, toErr(ctx, err)
	}
	return resp.Version, nil
}
