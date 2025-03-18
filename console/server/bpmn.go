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

	"github.com/olive-io/olive/api/rpc/consolepb"
	"github.com/olive-io/olive/console/service/bpmn"
)

type BpmnRPC struct {
	consolepb.UnimplementedBpmnRPCServer

	s *bpmn.Service
}

func NewBpmnRPC(s *bpmn.Service) *BpmnRPC {
	return &BpmnRPC{s: s}
}

func (rpc *BpmnRPC) ListDefinitions(ctx context.Context, req *consolepb.ListDefinitionsRequest) (*consolepb.ListDefinitionsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	result, err := rpc.s.ListDefinitions(ctx, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.ListDefinitionsResponse{
		Definitions: result.List,
		Total:       result.Total,
	}
	return resp, nil
}

func (rpc *BpmnRPC) GetDefinition(ctx context.Context, req *consolepb.GetDefinitionRequest) (*consolepb.GetDefinitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	definition, err := rpc.s.GetDefinition(ctx, req.Id, req.Version)
	if err != nil {
		return nil, err
	}

	resp := &consolepb.GetDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *BpmnRPC) DeployDefinition(ctx context.Context, req *consolepb.DeployDefinitionRequest) (*consolepb.DeployDefinitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	definition, err := rpc.s.DeployDefinition(ctx, req.Definition)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.DeployDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *BpmnRPC) DeleteDefinition(ctx context.Context, req *consolepb.DeleteDefinitionRequest) (*consolepb.DeleteDefinitionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	definition, err := rpc.s.DeleteDefinition(ctx, req.Id, req.Version)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.DeleteDefinitionResponse{
		Definition: definition,
	}
	return resp, nil
}

func (rpc *BpmnRPC) ListProcesses(ctx context.Context, req *consolepb.ListProcessesRequest) (*consolepb.ListProcessesResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	result, err := rpc.s.ListProcesses(ctx, req.Page, req.Size, req.Definition, req.Version, req.Status)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.ListProcessesResponse{
		Processes: result.List,
		Total:     result.Total,
	}
	return resp, nil
}

func (rpc *BpmnRPC) GetProcess(ctx context.Context, req *consolepb.GetProcessRequest) (*consolepb.GetProcessResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	process, err := rpc.s.GetProcess(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.GetProcessResponse{
		Process: process,
	}
	return resp, nil
}

func (rpc *BpmnRPC) DeleteProcess(ctx context.Context, req *consolepb.DeleteProcessRequest) (*consolepb.DeleteProcessResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	process, err := rpc.s.DeleteProcess(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	resp := &consolepb.DeleteProcessResponse{
		Process: process,
	}
	return resp, nil
}
