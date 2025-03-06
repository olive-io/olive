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

package server

import (
	"context"

	pb "github.com/olive-io/olive/api/rpc/monpb"
	"github.com/olive-io/olive/mon/service/cluster"
)

type clusterRPC struct {
	pb.UnsafeClusterServer

	s *cluster.Service
}

func newCluster(s *cluster.Service) *clusterRPC {
	rpc := &clusterRPC{s: s}
	return rpc
}

func (rpc *clusterRPC) MemberAdd(ctx context.Context, req *pb.MemberAddRequest) (*pb.MemberAddResponse, error) {
	header, member, members, err := rpc.s.MemberAdd(ctx, req.PeerURLs, req.IsLearner)
	if err != nil {
		return nil, err
	}

	return &pb.MemberAddResponse{Header: header, Member: member, Members: members}, nil
}

func (rpc *clusterRPC) MemberRemove(ctx context.Context, req *pb.MemberRemoveRequest) (*pb.MemberRemoveResponse, error) {
	header, members, err := rpc.s.MemberRemove(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &pb.MemberRemoveResponse{Header: header, Members: members}, nil
}

func (rpc *clusterRPC) MemberUpdate(ctx context.Context, req *pb.MemberUpdateRequest) (*pb.MemberUpdateResponse, error) {
	header, members, err := rpc.s.MemberUpdate(ctx, req.ID, req.PeerURLs)
	if err != nil {
		return nil, err
	}
	return &pb.MemberUpdateResponse{Header: header, Members: members}, nil
}

func (rpc *clusterRPC) MemberList(ctx context.Context, req *pb.MemberListRequest) (*pb.MemberListResponse, error) {
	header, members, err := rpc.s.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.MemberListResponse{Header: header, Members: members}, nil
}

func (rpc *clusterRPC) MemberPromote(ctx context.Context, req *pb.MemberPromoteRequest) (*pb.MemberPromoteResponse, error) {
	header, members, err := rpc.s.MemberPromote(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &pb.MemberPromoteResponse{Header: header, Members: members}, nil
}
