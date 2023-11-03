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

	"github.com/olive-io/olive/api"
	"google.golang.org/grpc"
)

type IDefinitionKV interface {
	Deploy(ctx context.Context, id, name string, content []byte) (int64, error)
	List(ctx context.Context) ([]*api.Definition, error)
	Get(ctx context.Context, id string, version int64) (*api.Definition, error)
	Remove(ctx context.Context, id string) error
	Execute(ctx context.Context, id string) (*api.ProcessInstance, error)
}

type definitionKVC struct {
	remote   api.DefinitionRPCClient
	callOpts []grpc.CallOption
}

func NewDefinitionKV(c *Client) IDefinitionKV {
	//kvc := &definitionKVC{remote: RetryDefinitionClient(c)}
	kvc := &definitionKVC{remote: api.NewDefinitionRPCClient(c.conn)}
	if c != nil {
		kvc.callOpts = c.callOpts
	}
	return kvc
}

func NewDefinitionKVFromClient(remote api.DefinitionRPCClient, c *Client) IDefinitionKV {
	kvc := &definitionKVC{remote: remote}
	if c != nil {
		kvc.callOpts = c.callOpts
	}
	return kvc
}

func (kvc *definitionKVC) Deploy(ctx context.Context, id, name string, content []byte) (int64, error) {
	in := &api.DeployDefinitionRequest{
		Id:      id,
		Name:    name,
		Content: content,
	}
	resp, err := kvc.remote.DeployDefinition(ctx, in, kvc.callOpts...)
	if err != nil {
		return 0, toErr(ctx, err)
	}
	return resp.Version, nil
}

func (kvc *definitionKVC) List(ctx context.Context) ([]*api.Definition, error) {
	in := &api.ListDefinitionRequest{}
	resp, err := kvc.remote.ListDefinition(ctx, in, kvc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Definitions, nil
}

func (kvc *definitionKVC) Get(ctx context.Context, id string, version int64) (*api.Definition, error) {
	in := &api.GetDefinitionRequest{
		Id:      id,
		Version: version,
	}
	resp, err := kvc.remote.GetDefinition(ctx, in, kvc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Definition, nil
}

func (kvc *definitionKVC) Remove(ctx context.Context, id string) error {
	in := &api.RemoveDefinitionRequest{
		Id:   id,
		Name: "",
	}
	_, err := kvc.remote.RemoveDefinition(ctx, in, kvc.callOpts...)
	return toErr(ctx, err)
}

func (kvc *definitionKVC) Execute(ctx context.Context, id string) (*api.ProcessInstance, error) {
	in := &api.ExecuteDefinitionRequest{
		Id:   id,
		Name: "",
	}
	resp, err := kvc.remote.ExecuteDefinition(ctx, in, kvc.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return resp.Instance, nil
}
