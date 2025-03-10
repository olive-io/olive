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

type BpmnRPC interface {
	DeployDefinition(ctx context.Context, id int64, name string, description string, metadata map[string]string, content string) (*types.Definition, error)
	ListDefinitions(ctx context.Context, options ...ListDefinitionsOption) ([]*types.Definition, int64, error)
	GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error)
	RemoveDefinition(ctx context.Context, id int64) error
	ExecuteDefinition(ctx context.Context, id int64, options ...ExecDefinitionOption) (*types.ProcessInstance, error)
}

type bpmnRPC struct {
	client   *Client
	callOpts []grpc.CallOption
}

func NewBpmnRPC(c *Client) BpmnRPC {
	api := &bpmnRPC{client: c}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (bc *bpmnRPC) DeployDefinition(ctx context.Context, id int64, name string, description string, metadata map[string]string, content string) (*types.Definition, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
		defer conn.Close()
	}

	req := &pb.DeployDefinitionRequest{
		Id:          id,
		Name:        name,
		Description: description,
		Metadata:    metadata,
		Content:     content,
	}
	resp, err := bc.remoteClient(conn).DeployDefinition(ctx, req, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return resp.Definition, nil
}

type ListDefinitionsOption func(request *pb.ListDefinitionsRequest)

func ListWithPagination(page, size int32) ListDefinitionsOption {
	return func(req *pb.ListDefinitionsRequest) {
		req.Page = page
		if req.Page <= 0 {
			req.Page = 1
		}
		req.Size = size
	}
}

func (bc *bpmnRPC) ListDefinitions(ctx context.Context, options ...ListDefinitionsOption) ([]*types.Definition, int64, error) {
	conn := bc.client.conn
	in := pb.ListDefinitionsRequest{}
	for _, option := range options {
		option(&in)
	}
	resp, err := bc.remoteClient(conn).ListDefinitions(ctx, &in, bc.callOpts...)
	if err != nil {
		return nil, 0, toErr(ctx, err)
	}

	return resp.Definitions, resp.Total, nil
}

func (bc *bpmnRPC) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	conn := bc.client.conn
	in := &pb.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}

	resp, err := bc.remoteClient(conn).GetDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Definition, nil
}

func (bc *bpmnRPC) RemoveDefinition(ctx context.Context, id int64) error {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.Dial(leaderEndpoints[0])
		if err != nil {
			return err
		}
		defer conn.Close()
	}

	in := &pb.RemoveDefinitionRequest{Id: id}
	_, err = bc.remoteClient(conn).RemoveDefinition(ctx, in, bc.callOpts...)
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

func WithVersion(version uint64) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.DefinitionVersion = version
	}
}

func WithHeaders(headers map[string]string) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.Header = headers
	}
}

func WithProperties(properties map[string][]byte) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		if req.Properties == nil {
			req.Properties = map[string][]byte{}
		}
		for name, value := range properties {
			req.Properties[name] = value
		}
	}
}

func WithDataObjects(objects map[string][]byte) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		if req.Properties == nil {
			req.Properties = map[string][]byte{}
		}
		for name, value := range objects {
			req.Properties[name] = value
		}
	}
}

func (bc *bpmnRPC) ExecuteDefinition(ctx context.Context, id int64, options ...ExecDefinitionOption) (*types.ProcessInstance, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
		defer conn.Close()
	}
	in := &pb.ExecuteDefinitionRequest{DefinitionId: id}
	for _, option := range options {
		option(in)
	}

	resp, err := bc.remoteClient(conn).ExecuteDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Process, nil
}

func (bc *bpmnRPC) remoteClient(conn *grpc.ClientConn) pb.BpmnRPCClient {
	return RetryBpmnClient(conn)
}
