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

	pb "github.com/olive-io/olive/api/olivepb"
	"google.golang.org/grpc"
)

type RunnerRPC interface {
	RegistryRunner(ctx context.Context, runner *pb.Runner) (uint64, error)
	ReportRunner(ctx context.Context, id uint64, runner *pb.RunnerStat, regions []*pb.RegionStat) error
}

type runnerRPC struct {
	remote   pb.RunnerRPCClient
	callOpts []grpc.CallOption
}

func NewRunnerRPC(c *Client) RunnerRPC {
	api := &runnerRPC{remote: RetryRunnerClient(c)}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (rc *runnerRPC) RegistryRunner(ctx context.Context, runner *pb.Runner) (uint64, error) {
	r := &pb.RegistryRunnerRequest{
		Runner: runner,
	}
	resp, err := rc.remote.RegistryRunner(ctx, r, rc.callOpts...)
	if err != nil {
		return 0, toErr(ctx, err)
	}
	return resp.Id, nil
}

func (rc *runnerRPC) ReportRunner(ctx context.Context, id uint64, runner *pb.RunnerStat, regions []*pb.RegionStat) error {
	r := &pb.ReportRunnerRequest{
		Id:      id,
		Runner:  runner,
		Regions: regions,
	}
	_, err := rc.remote.ReportRunner(ctx, r, rc.callOpts...)
	if err != nil {
		return toErr(ctx, err)
	}

	return nil
}
