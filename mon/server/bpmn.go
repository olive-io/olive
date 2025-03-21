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
	"github.com/olive-io/olive/api/types"
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

func (rpc *bpmnRPC) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (*pb.DeployDefinitionResponse, error) {
	definition := req.Definition
	var err error
	definition, err = rpc.s.DeployDefinition(ctx, definition)
	if err != nil {
		return nil, err
	}
	resp := &pb.DeployDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *bpmnRPC) ListDefinitions(ctx context.Context, req *pb.ListDefinitionsRequest) (*pb.ListDefinitionsResponse, error) {
	definitions, err := rpc.s.ListDefinitions(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListDefinitionsResponse{
		Definitions: definitions,
	}
	return resp, nil
}

func (rpc *bpmnRPC) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (*pb.GetDefinitionResponse, error) {
	definition, err := rpc.s.GetDefinition(ctx, req.Id, req.Version)
	if err != nil {
		return nil, err
	}
	resp := &pb.GetDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *bpmnRPC) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (*pb.RemoveDefinitionResponse, error) {
	definition, err := rpc.s.RemoveDefinition(ctx, req.Id, req.Version)
	if err != nil {
		return nil, err
	}
	resp := &pb.RemoveDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *bpmnRPC) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (*pb.ExecuteDefinitionResponse, error) {
	process := &types.Process{
		Name:     req.Name,
		Metadata: make(map[string]string),
		Args: &types.BpmnArgs{
			Headers:     req.Header,
			Properties:  req.Properties,
			DataObjects: req.DataObjects,
		},
		DefinitionsId:      req.DefinitionId,
		DefinitionsVersion: req.DefinitionVersion,
	}
	var err error
	process, err = rpc.s.ExecuteDefinition(ctx, process)
	if err != nil {
		return nil, err
	}
	resp := &pb.ExecuteDefinitionResponse{
		Process: process,
	}
	return resp, nil
}

func (rpc *bpmnRPC) ListProcess(ctx context.Context, req *pb.ListProcessRequest) (*pb.ListProcessResponse, error) {
	processes, err := rpc.s.ListProcess(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.ListProcessResponse{
		Processes: processes,
	}
	return resp, nil
}

func (rpc *bpmnRPC) GetProcess(ctx context.Context, req *pb.GetProcessRequest) (*pb.GetProcessResponse, error) {
	process, err := rpc.s.GetProcess(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &pb.GetProcessResponse{
		Process: process,
	}
	return resp, nil
}

func (rpc *bpmnRPC) UpdateProcess(ctx context.Context, req *pb.UpdateProcessRequest) (*pb.UpdateProcessResponse, error) {
	err := rpc.s.UpdateProcess(ctx, req.Process)
	if err != nil {
		return nil, err
	}
	resp := &pb.UpdateProcessResponse{}
	return resp, nil
}

func (rpc *bpmnRPC) RemoveProcess(ctx context.Context, req *pb.RemoveProcessRequest) (*pb.RemoveProcessResponse, error) {
	process, err := rpc.s.RemoveProcess(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &pb.RemoveProcessResponse{
		Process: process,
	}
	return resp, nil
}
