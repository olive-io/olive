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

package meta

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
)

func (s *Server) RegisterRunner(ctx context.Context, req *pb.RegisterRunnerRequest) (resp *pb.RegisterRunnerResponse, err error) {
	if req.Runner == nil {
		return resp, rpctypes.ErrGRPCInvalidRunner
	}

	resp = &pb.RegisterRunnerResponse{}
	resp.Id, err = s.registry.Register(ctx, req.Runner)
	return
}

func (s *Server) ReportRunner(ctx context.Context, req *pb.ReportRunnerRequest) (resp *pb.ReportRunnerResponse, err error) {
	resp = &pb.ReportRunnerResponse{}

	//runner := req.Runner
	//regions := req.Regions

	return
}

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	//TODO implement me
	return
}

func (s *Server) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (resp *pb.ListDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (resp *pb.RemoveDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (resp *pb.ExecuteDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}
