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

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	pb "github.com/olive-io/olive/api/olivepb"
)

type clusterServer struct {
	pb.UnsafeMetaClusterRPCServer

	*Server
}

func newClusterServer(s *Server) (*clusterServer, error) {
	cs := &clusterServer{Server: s}
	return cs, nil
}

func (s *clusterServer) MemberAdd(ctx context.Context, req *pb.MemberAddRequest) (resp *pb.MemberAddResponse, err error) {

	var rsp *clientv3.MemberAddResponse
	if req.IsLearner {
		rsp, err = s.v3cli.MemberAddAsLearner(ctx, req.PeerURLs)
	} else {
		rsp, err = s.v3cli.MemberAdd(ctx, req.PeerURLs)
	}
	if err != nil {
		return nil, err
	}

	resp = &pb.MemberAddResponse{
		Header:  toHeader(rsp.Header),
		Member:  toMember(rsp.Member),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}

	return resp, nil
}

func (s *clusterServer) MemberRemove(ctx context.Context, req *pb.MemberRemoveRequest) (resp *pb.MemberRemoveResponse, err error) {

	rsp, err := s.v3cli.MemberRemove(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp = &pb.MemberRemoveResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *clusterServer) MemberUpdate(ctx context.Context, req *pb.MemberUpdateRequest) (resp *pb.MemberUpdateResponse, err error) {

	rsp, err := s.v3cli.MemberUpdate(ctx, req.ID, req.PeerURLs)
	if err != nil {
		return nil, err
	}

	resp = &pb.MemberUpdateResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *clusterServer) MemberList(ctx context.Context, req *pb.MemberListRequest) (resp *pb.MemberListResponse, err error) {

	rsp, err := s.v3cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	resp = &pb.MemberListResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *clusterServer) MemberPromote(ctx context.Context, req *pb.MemberPromoteRequest) (resp *pb.MemberPromoteResponse, err error) {

	rsp, err := s.v3cli.MemberPromote(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp = &pb.MemberPromoteResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func toHeader(header *etcdserverpb.ResponseHeader) *pb.ResponseHeader {
	return &pb.ResponseHeader{
		ClusterId: header.ClusterId,
		MemberId:  header.MemberId,
		RaftTerm:  header.RaftTerm,
	}
}

func toMember(member *etcdserverpb.Member) *pb.Member {
	return &pb.Member{
		ID:         member.ID,
		Name:       member.Name,
		PeerURLs:   member.PeerURLs,
		ClientURLs: member.ClientURLs,
		IsLearner:  member.IsLearner,
	}
}
