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

package brain

import (
	"context"

	dragonboat "github.com/lni/dragonboat/v4"
	"github.com/oliveio/olive/api"
)

type Server struct {
	Option

	ctx context.Context

	nh *dragonboat.NodeHost

	done chan struct{}
	exit chan struct{}
}

func (s Server) DeployDefinition(ctx context.Context, req *api.DeployDefinitionRequest) (*api.DeployDefinitionResponse, error) {
	//shareID := s.Option.Node.ShardID
	//var query any
	//session := s.nh.GetNoOPSession(shareID)
	//
	//result, err := s.nh.SyncPropose(ctx, session, query)
	//if err != nil {
	//	return nil, err
	//}

	//TODO implement me
	panic("implement me")
}

func (s Server) ListDefinition(ctx context.Context, req *api.ListDefinitionRequest) (*api.ListDefinitionResponse, error) {
	//TODO implement me
	//shareID := s.Option.Node.ShardID
	//var query any
	//result, err := s.nh.SyncRead(ctx, shareID, query)
	//if err != nil {
	//	return nil, err
	//}
	panic("implement me")
}

func (s Server) GetDefinition(ctx context.Context, req *api.GetDefinitionRequest) (*api.GetDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) RemoveDefinition(ctx context.Context, req *api.RemoveDefinitionRequest) (*api.RemoveDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) ExecuteDefinition(ctx context.Context, req *api.ExecuteDefinitionRequest) (*api.ExecuteDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}
