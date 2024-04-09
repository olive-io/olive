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

func (s *Server) MemberAdd(ctx context.Context, req *pb.MemberAddRequest) (*pb.MemberAddResponse, error) {
	var rsp *clientv3.MemberAddResponse
	var err error
	if req.IsLearner {
		rsp, err = s.v3cli.MemberAddAsLearner(ctx, req.PeerURLs)
	} else {
		rsp, err = s.v3cli.MemberAdd(ctx, req.PeerURLs)
	}
	if err != nil {
		return nil, err
	}

	resp := &pb.MemberAddResponse{
		Header:  toHeader(rsp.Header),
		Member:  toMember(rsp.Member),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}

	return resp, nil
}

func (s *Server) MemberRemove(ctx context.Context, req *pb.MemberRemoveRequest) (*pb.MemberRemoveResponse, error) {
	rsp, err := s.v3cli.MemberRemove(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberRemoveResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *Server) MemberUpdate(ctx context.Context, req *pb.MemberUpdateRequest) (*pb.MemberUpdateResponse, error) {
	rsp, err := s.v3cli.MemberUpdate(ctx, req.ID, req.PeerURLs)
	if err != nil {
		return nil, err
	}

	resp := &pb.MemberUpdateResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *Server) MemberList(ctx context.Context, req *pb.MemberListRequest) (*pb.MemberListResponse, error) {
	rsp, err := s.v3cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberListResponse{
		Header:  toHeader(rsp.Header),
		Members: []*pb.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (s *Server) MemberPromote(ctx context.Context, req *pb.MemberPromoteRequest) (*pb.MemberPromoteResponse, error) {
	rsp, err := s.v3cli.MemberPromote(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberPromoteResponse{
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
