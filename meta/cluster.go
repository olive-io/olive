// Copyright 2024 The olive Authors
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
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
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
