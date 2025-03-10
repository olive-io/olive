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
	"encoding/xml"

	"github.com/cockroachdb/errors"
	"github.com/olive-io/bpmn/schema"
	"google.golang.org/grpc"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/olivepb"
)

type BpmnRPC interface {
	ListDefinitions(ctx context.Context, limit int64, continueToken string) (*pb.ListDefinitionResponse, error)
	DeployDefinition(ctx context.Context, id, name string, body []byte) (*pb.DeployDefinitionResponse, error)
	GetDefinition(ctx context.Context, id string, version uint64) (*pb.GetDefinitionResponse, error)
	RemoveDefinition(ctx context.Context, id string) (*pb.RemoveDefinitionResponse, error)
	ExecuteDefinition(ctx context.Context, id string, options ...ExecDefinitionOption) (*pb.ExecuteDefinitionResponse, error)
	ListProcessInstances(ctx context.Context, definitionId string, definitionVersion uint64, limit int64, continueToken string) (*pb.ListProcessInstancesResponse, error)
	GetProcessInstance(ctx context.Context, definitionId string, definitionVersion uint64, id string) (*pb.GetProcessInstanceResponse, error)
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

func (bc *bpmnRPC) ListDefinitions(ctx context.Context, limit int64, continueToken string) (*pb.ListDefinitionResponse, error) {
	conn := bc.client.conn
	in := pb.ListDefinitionRequest{
		Limit:    limit,
		Continue: continueToken,
	}
	rsp, err := bc.remoteClient(conn).ListDefinition(ctx, &in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return rsp, nil
}

func (bc *bpmnRPC) DeployDefinition(ctx context.Context, id, name string, body []byte) (*pb.DeployDefinitionResponse, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
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
	resp, err := bc.remoteClient(conn).DeployDefinition(ctx, r, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return resp, nil
}

func (bc *bpmnRPC) GetDefinition(ctx context.Context, id string, version uint64) (*pb.GetDefinitionResponse, error) {
	conn := bc.client.conn
	in := &pb.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}

	rsp, err := bc.remoteClient(conn).GetDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp, nil
}

func (bc *bpmnRPC) RemoveDefinition(ctx context.Context, id string) (*pb.RemoveDefinitionResponse, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.RemoveDefinitionRequest{Id: id}
	rsp, err := bc.remoteClient(conn).RemoveDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp, nil
}

type ExecDefinitionOption func(request *pb.ExecuteDefinitionRequest)

func WithVersion(version uint64) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.DefinitionVersion = version
	}
}

func WithName(name string) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.Name = name
	}
}

func WithHeaders(headers map[string]string) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		req.Header = headers
	}
}

func WithProperties(properties map[string]any) ExecDefinitionOption {
	return func(req *pb.ExecuteDefinitionRequest) {
		if req.Properties == nil {
			req.Properties = map[string]*dsypb.Box{}
		}
		for name, value := range properties {
			req.Properties[name] = dsypb.BoxFromAny(value)
		}
	}
}

func (bc *bpmnRPC) ExecuteDefinition(ctx context.Context, id string, options ...ExecDefinitionOption) (*pb.ExecuteDefinitionResponse, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}
	in := &pb.ExecuteDefinitionRequest{DefinitionId: id}
	for _, option := range options {
		option(in)
	}

	rsp, err := bc.remoteClient(conn).ExecuteDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp, nil
}

func (bc *bpmnRPC) ListProcessInstances(ctx context.Context, definitionId string, definitionVersion uint64, limit int64, continueToken string) (*pb.ListProcessInstancesResponse, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}
	in := &pb.ListProcessInstancesRequest{
		DefinitionId:      definitionId,
		DefinitionVersion: definitionVersion,
		Limit:             limit,
		Continue:          continueToken,
	}

	rsp, err := bc.remoteClient(conn).ListProcessInstances(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp, nil
}

func (bc *bpmnRPC) GetProcessInstance(ctx context.Context, definitionId string, definitionVersion uint64, id string) (*pb.GetProcessInstanceResponse, error) {
	conn := bc.client.conn
	leaderEndpoints, err := bc.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = bc.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}
	in := &pb.GetProcessInstanceRequest{
		DefinitionId:      definitionId,
		DefinitionVersion: definitionVersion,
		Id:                id,
	}

	rsp, err := bc.remoteClient(conn).GetProcessInstance(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return rsp, nil
}

func (bc *bpmnRPC) remoteClient(conn *grpc.ClientConn) pb.BpmnRPCClient {
	return RetryBpmnClient(conn)
}
