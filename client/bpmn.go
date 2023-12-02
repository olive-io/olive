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
	DeployDefinition(ctx context.Context, id, name string, body []byte) (*pb.Definition, error)
	ListDefinitions(ctx context.Context, options ...ListDefinitionOption) ([]*pb.Definition, string, error)
	GetDefinition(ctx context.Context, id string, version uint64) (*pb.Definition, error)
	RemoveDefinition(ctx context.Context, id string) error
	ExecuteDefinition(ctx context.Context, id string, options ...ExecDefinitionOption) (*pb.ProcessInstance, error)
}

type bpmnRPC struct {
	client   *Client
	remote   pb.BpmnRPCClient
	callOpts []grpc.CallOption
}

func NewBpmnRPC(c *Client) BpmnRPC {
	conn := c.ActiveConnection()
	api := &bpmnRPC{
		client: c,
		remote: RetryBpmnClient(conn),
	}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (bc *bpmnRPC) DeployDefinition(ctx context.Context, id, name string, body []byte) (*pb.Definition, error) {
	endpoints := bc.client.Endpoints()
	defer bc.client.SetEndpoints(endpoints...)

	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		bc.client.SetEndpoints(leaderEndpoints...)
	}

	var definitions schema.Definitions
	if err := xml.Unmarshal(body, &definitions); err != nil {
		return nil, errors.Wrap(ErrBadFormatDefinitions, err.Error())
	}

	r := &pb.DeployDefinitionRequest{
		Id:      id,
		Name:    name,
		Content: body,
	}
	resp, err := bc.remote.DeployDefinition(ctx, r, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	definition := &pb.Definition{
		Header: &pb.OliveHeader{
			Runner: 0,
			Region: resp.Region,
		},
		Id:      id,
		Name:    name,
		Content: body,
		Version: resp.Version,
	}

	return definition, nil
}

type ListDefinitionOption func(request *pb.ListDefinitionRequest)

func WithLimit(limit int64) ListDefinitionOption {
	return func(req *pb.ListDefinitionRequest) {
		req.Limit = limit
	}
}

func WithContinue(token string) ListDefinitionOption {
	return func(req *pb.ListDefinitionRequest) {
		req.Continue = token
	}
}

func (bc *bpmnRPC) ListDefinitions(ctx context.Context, options ...ListDefinitionOption) ([]*pb.Definition, string, error) {
	in := pb.ListDefinitionRequest{}
	for _, option := range options {
		option(&in)
	}
	rsp, err := bc.remote.ListDefinition(ctx, &in, bc.callOpts...)
	if err != nil {
		return nil, "", toErr(ctx, err)
	}

	return rsp.Definitions, rsp.ContinueToken, nil
}

func (bc *bpmnRPC) GetDefinition(ctx context.Context, id string, version uint64) (*pb.Definition, error) {
	in := &pb.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}

	rsp, err := bc.remote.GetDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Definition, nil
}

func (bc *bpmnRPC) RemoveDefinition(ctx context.Context, id string) error {
	endpoints := bc.client.Endpoints()
	defer bc.client.SetEndpoints(endpoints...)

	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return err
	}
	if len(leaderEndpoints) > 0 {
		bc.client.SetEndpoints(leaderEndpoints...)
	}

	in := &pb.RemoveDefinitionRequest{Id: id}
	_, err = bc.remote.RemoveDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return toErr(ctx, err)
	}
	return nil
}

type ExecDefinitionOption func(request *pb.ExecuteDefinitionRequest)

func WithName(name string) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.Name = name
	}
}

func WithHeader(headers map[string]string) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.Header = headers
	}
}

func (bc *bpmnRPC) ExecuteDefinition(ctx context.Context, id string, options ...ExecDefinitionOption) (*pb.ProcessInstance, error) {
	endpoints := bc.client.Endpoints()
	defer bc.client.SetEndpoints(endpoints...)

	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		bc.client.SetEndpoints(leaderEndpoints...)
	}

	in := &pb.ExecuteDefinitionRequest{Id: id}
	for _, option := range options {
		option(in)
	}

	rsp, err := bc.remote.ExecuteDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp.Instance, nil
}
