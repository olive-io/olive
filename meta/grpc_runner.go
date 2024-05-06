/*
   Copyright 2024 The olive Authors

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

package meta

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/runtime"
)

type runnerServer struct {
	pb.UnsafeMetaRunnerRPCServer
	pb.UnsafeMetaRegionRPCServer

	*Server
}

func (s *runnerServer) ListRunner(ctx context.Context, req *pb.ListRunnerRequest) (resp *pb.ListRunnerResponse, err error) {
	resp = &pb.ListRunnerResponse{}
	key := runtime.DefaultMetaRunnerRegistrar
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	runners := make([]*pb.Runner, 0, rsp.Count)
	for _, kv := range rsp.Kvs {
		runner := new(pb.Runner)
		if e1 := proto.Unmarshal(kv.Value, runner); e1 == nil {
			runners = append(runners, runner)
		}
	}
	resp.Header = s.responseHeader()
	resp.Runners = runners
	return resp, nil
}

func (s *runnerServer) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (resp *pb.GetRunnerResponse, err error) {
	resp = &pb.GetRunnerResponse{}
	resp.Runner, err = s.getRunner(ctx, req.Id)
	resp.Header = s.responseHeader()
	return
}

func (s *runnerServer) ListRegion(ctx context.Context, req *pb.ListRegionRequest) (resp *pb.ListRegionResponse, err error) {
	resp = &pb.ListRegionResponse{}
	key := runtime.DefaultRunnerRegion
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	regions := make([]*pb.Region, 0, rsp.Count)
	for _, kv := range rsp.Kvs {
		region := new(pb.Region)
		if e1 := proto.Unmarshal(kv.Value, region); e1 == nil {
			regions = append(regions, region)
		}
	}

	resp.Header = s.responseHeader()
	resp.Regions = regions
	return resp, nil
}

func (s *runnerServer) GetRegion(ctx context.Context, req *pb.GetRegionRequest) (resp *pb.GetRegionResponse, err error) {
	resp = &pb.GetRegionResponse{}
	resp.Region, err = s.getRegion(ctx, req.Id)
	resp.Header = s.responseHeader()
	return
}
