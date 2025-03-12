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
	DeployDefinition(ctx context.Context, definition *types.Definition) (*types.Definition, error)
	ListDefinitions(ctx context.Context, page, size int32) ([]*types.Definition, int64, error)
	GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error)
	RemoveDefinition(ctx context.Context, id int64) error
	ExecuteDefinition(ctx context.Context, id int64, version uint64, options *ExecuteOptions) (*types.Process, error)
	ListProcess(ctx context.Context, definition int64, version uint64, page, size int32) ([]*types.Process, int64, error)
	GetProcess(ctx context.Context, id int64) (*types.Process, error)
	UpdateProcess(ctx context.Context, process *types.Process) error
	RemoveProcess(ctx context.Context, id int64) (*types.Process, error)
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

func (bc *bpmnRPC) DeployDefinition(ctx context.Context, definition *types.Definition) (*types.Definition, error) {
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
		Definition: definition,
	}
	resp, err := bc.remoteClient(conn).DeployDefinition(ctx, req, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return resp.Definition, nil
}

func (bc *bpmnRPC) ListDefinitions(ctx context.Context, page, size int32) ([]*types.Definition, int64, error) {
	conn := bc.client.conn
	in := pb.ListDefinitionsRequest{
		Page: page,
		Size: size,
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

type ExecuteOptions struct {
	Name        string
	Header      map[string]string
	Properties  map[string][]byte
	DataObjects map[string][]byte
}

func (bc *bpmnRPC) ExecuteDefinition(ctx context.Context, id int64, version uint64, options *ExecuteOptions) (*types.Process, error) {
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
	in := &pb.ExecuteDefinitionRequest{
		Name:              options.Name,
		DefinitionId:      id,
		DefinitionVersion: version,
		Header:            options.Header,
		Properties:        options.Properties,
		DataObjects:       options.DataObjects,
	}

	resp, err := bc.remoteClient(conn).ExecuteDefinition(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Process, nil
}

func (bc *bpmnRPC) ListProcess(ctx context.Context, definition int64, version uint64, page, size int32) ([]*types.Process, int64, error) {
	conn := bc.client.conn
	in := &pb.ListProcessRequest{
		Definition: definition,
		Version:    version,
		Page:       page,
		Size:       size,
	}
	resp, err := bc.remoteClient(conn).ListProcess(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, 0, toErr(ctx, err)
	}
	return resp.Processes, resp.Total, nil
}

func (bc *bpmnRPC) GetProcess(ctx context.Context, id int64) (*types.Process, error) {
	conn := bc.client.conn
	in := &pb.GetProcessRequest{Id: id}
	resp, err := bc.remoteClient(conn).GetProcess(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Process, nil
}

func (bc *bpmnRPC) UpdateProcess(ctx context.Context, process *types.Process) error {
	conn := bc.client.conn
	in := &pb.UpdateProcessRequest{Process: process}
	_, err := bc.remoteClient(conn).UpdateProcess(ctx, in, bc.callOpts...)
	if err != nil {
		return toErr(ctx, err)
	}
	return nil
}

func (bc *bpmnRPC) RemoveProcess(ctx context.Context, id int64) (*types.Process, error) {
	conn := bc.client.conn
	in := &pb.RemoveProcessRequest{Id: id}
	resp, err := bc.remoteClient(conn).RemoveProcess(ctx, in, bc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Process, nil
}

func (bc *bpmnRPC) remoteClient(conn *grpc.ClientConn) pb.BpmnRPCClient {
	return RetryBpmnClient(conn)
}
