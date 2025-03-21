/*
Copyright 2025 The olive Authors

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
	"fmt"

	json "github.com/bytedance/sonic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/olive-io/olive/api/rpc/runnerpb"
	"github.com/olive-io/olive/api/types"
)

type Client struct {
	cfg *Config
}

func NewClient(cfg *Config) (*Client, error) {

	client := &Client{
		cfg: cfg,
	}

	conn, err := client.newConn()
	if err != nil {
		return nil, err
	}
	_ = conn.Close()

	return client, nil
}

func (c *Client) newConn() (*grpc.ClientConn, error) {
	target := c.cfg.Address

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(target, options...)
	return conn, err
}

func (c *Client) getCallOptions() []grpc.CallOption {
	options := []grpc.CallOption{}

	return options
}

func (c *Client) GetRunner(ctx context.Context) (*types.Runner, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetRunnerRequest{}
	options := c.getCallOptions()

	resp, err := cc.GetRunner(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Runner, nil
}

func (c *Client) ListDefinitions(ctx context.Context, id int64) ([]*types.Definition, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.ListDefinitionsRequest{
		Id: id,
	}
	options := c.getCallOptions()

	resp, err := cc.ListDefinitions(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Definitions, nil
}

func (c *Client) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}
	options := c.getCallOptions()

	resp, err := cc.GetDefinition(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Definition, nil
}

func (c *Client) ListProcess(ctx context.Context, definition int64, version uint64) ([]*types.Process, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.ListProcessRequest{
		DefinitionId:      definition,
		DefinitionVersion: version,
	}
	options := c.getCallOptions()

	resp, err := cc.ListProcess(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Processes, nil
}

func (c *Client) GetProcess(ctx context.Context, definition int64, version uint64, id int64) (*types.Process, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	in := &pb.GetProcessRequest{
		DefinitionId:      definition,
		DefinitionVersion: version,
		Id:                id,
	}
	options := c.getCallOptions()

	resp, err := cc.GetProcess(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Process, nil
}

func (c *Client) RunProcess(ctx context.Context, in *pb.RunProcessRequest) (*types.Process, error) {
	conn, err := c.newConn()
	if err != nil {
		return nil, err
	}

	cc := pb.NewRunnerRPCClient(conn)
	options := c.getCallOptions()

	resp, err := cc.RunProcess(ctx, in, options...)
	if err != nil {
		return nil, err
	}

	return resp.Process, nil
}

func (c *Client) BuildProcess() *processBuilder {
	builder := &processBuilder{
		RunProcessRequest: &pb.RunProcessRequest{
			Process: &types.Process{
				Args: &types.BpmnArgs{},
			},
		},
		cc: c,
	}

	return builder
}

type processBuilder struct {
	*pb.RunProcessRequest

	cc *Client
}

func (b *processBuilder) SetDefinition(id int64, version uint64) *processBuilder {
	b.Process.DefinitionsId = id
	b.Process.DefinitionsVersion = version
	return b
}

func (b *processBuilder) SetBpmn(content string) *processBuilder {
	b.Process.DefinitionsContent = content
	return b
}

func (b *processBuilder) SetName(name string) *processBuilder {
	b.Process.Name = name
	return b
}

func (b *processBuilder) SetHeaders(headers map[string]string) *processBuilder {
	b.Process.Args.Headers = headers
	return b
}

func (b *processBuilder) SetProperties(properties map[string]any) *processBuilder {
	b.Process.Args.Properties = map[string][]byte{}
	for k, v := range properties {
		b.Process.Args.Properties[k] = marshalValue(v)
	}

	return b
}

func (b *processBuilder) SetDataObjects(dataObjects map[string]any) *processBuilder {
	b.Process.Args.DataObjects = map[string][]byte{}
	for k, v := range dataObjects {
		b.Process.Args.DataObjects[k] = marshalValue(v)
	}
	return b
}

func (b *processBuilder) Do(ctx context.Context) (*types.Process, error) {
	return b.cc.RunProcess(ctx, b.RunProcessRequest)
}

func marshalValue(v any) []byte {
	switch t := v.(type) {
	case []byte:
		return t
	case string:
		return []byte(t)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return []byte(fmt.Sprintf("%d", t))
	case float32, float64:
		return []byte(fmt.Sprintf("%f", t))
	case bool:
		return []byte(fmt.Sprintf("%t", t))
	default:
		data, _ := json.Marshal(t)
		return data
	}
}
