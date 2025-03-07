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

package server

import (
	"context"

	pb "github.com/olive-io/olive/api/rpc/monpb"
	"github.com/olive-io/olive/mon/service/bpmn"
)

type bpmnRPC struct {
	pb.UnimplementedBpmnRPCServer

	s *bpmn.Service
}

func newBpmn(s *bpmn.Service) *bpmnRPC {
	rpc := &bpmnRPC{s: s}
	return rpc
}

func (b bpmnRPC) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (*pb.DeployDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (b bpmnRPC) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (*pb.ListDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (b bpmnRPC) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (*pb.GetDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (b bpmnRPC) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (*pb.RemoveDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (b bpmnRPC) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (*pb.ExecuteDefinitionResponse, error) {
	//TODO implement me
	panic("implement me")
}
