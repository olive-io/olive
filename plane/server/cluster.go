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

	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	pb "github.com/olive-io/olive/api/rpc/planepb"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
)

type ClusterRPC struct {
	pb.UnsafeClusterServer
	v3cli *clientv3.Client

	stopc <-chan struct{}
}

func NewCluster(v3cli *clientv3.Client, stopc <-chan struct{}) (*ClusterRPC, error) {
	rpc := &ClusterRPC{
		v3cli: v3cli,
		stopc: stopc,
	}

	return rpc, nil
}

func (rpc *ClusterRPC) MemberAdd(ctx context.Context, req *pb.MemberAddRequest) (*pb.MemberAddResponse, error) {
	var rsp *clientv3.MemberAddResponse
	var err error
	if req.IsLearner {
		rsp, err = rpc.v3cli.MemberAddAsLearner(ctx, req.PeerURLs)
	} else {
		rsp, err = rpc.v3cli.MemberAdd(ctx, req.PeerURLs)
	}
	if err != nil {
		return nil, err
	}

	resp := &pb.MemberAddResponse{
		Header:  toHeader(rsp.Header),
		Member:  toMember(rsp.Member),
		Members: []*corev1.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}

	return resp, nil
}

func (rpc *ClusterRPC) MemberRemove(ctx context.Context, req *pb.MemberRemoveRequest) (*pb.MemberRemoveResponse, error) {
	rsp, err := rpc.v3cli.MemberRemove(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberRemoveResponse{
		Header:  toHeader(rsp.Header),
		Members: []*corev1.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (rpc *ClusterRPC) MemberUpdate(ctx context.Context, req *pb.MemberUpdateRequest) (*pb.MemberUpdateResponse, error) {
	rsp, err := rpc.v3cli.MemberUpdate(ctx, req.ID, req.PeerURLs)
	if err != nil {
		return nil, err
	}

	resp := &pb.MemberUpdateResponse{
		Header:  toHeader(rsp.Header),
		Members: []*corev1.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (rpc *ClusterRPC) MemberList(ctx context.Context, req *pb.MemberListRequest) (*pb.MemberListResponse, error) {
	rsp, err := rpc.v3cli.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberListResponse{
		Header:  toHeader(rsp.Header),
		Members: []*corev1.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func (rpc *ClusterRPC) MemberPromote(ctx context.Context, req *pb.MemberPromoteRequest) (*pb.MemberPromoteResponse, error) {
	rsp, err := rpc.v3cli.MemberPromote(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	resp := &pb.MemberPromoteResponse{
		Header:  toHeader(rsp.Header),
		Members: []*corev1.Member{},
	}
	for _, mem := range rsp.Members {
		resp.Members = append(resp.Members, toMember(mem))
	}
	return resp, nil
}

func toHeader(header *etcdserverpb.ResponseHeader) *corev1.ResponseHeader {
	return &corev1.ResponseHeader{
		ClusterID: header.ClusterId,
		MemberID:  header.MemberId,
		RaftTerm:  header.RaftTerm,
	}
}

func toMember(member *etcdserverpb.Member) *corev1.Member {
	return &corev1.Member{
		ID:         member.ID,
		Name:       member.Name,
		PeerURLs:   member.PeerURLs,
		ClientURLs: member.ClientURLs,
		IsLeader:   member.IsLearner,
	}
}
