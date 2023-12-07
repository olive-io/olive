// Copyright 2023 The olive Authors
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

package raft

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
)

var (
	definitionPrefix = []byte("definitions")
	processPrefix    = []byte("processes")
)

func (r *Region) DeployDefinition(ctx context.Context, req *pb.RegionDeployDefinitionRequest) (*pb.RegionDeployDefinitionResponse, error) {
	resp := &pb.RegionDeployDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{DeployDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionDeployDefinitionResponse)
	return resp, nil
}

func (r *Region) ExecuteDefinition(ctx context.Context, req *pb.RegionExecuteDefinitionRequest) (*pb.RegionExecuteDefinitionResponse, error) {
	resp := &pb.RegionExecuteDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{ExecuteDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionExecuteDefinitionResponse)
	return resp, nil
}
