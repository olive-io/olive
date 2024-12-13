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

package client

import (
	"context"

	v3types "go.etcd.io/etcd/client/pkg/v3/types"
	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/types"
)

type (
	Member                types.Member
	MemberListResponse    pb.MemberListResponse
	MemberAddResponse     pb.MemberAddResponse
	MemberRemoveResponse  pb.MemberRemoveResponse
	MemberUpdateResponse  pb.MemberUpdateResponse
	MemberPromoteResponse pb.MemberPromoteResponse
)

type Cluster interface {
	// MemberList lists the current cluster membership.
	MemberList(ctx context.Context) (*MemberListResponse, error)

	// MemberAdd adds a new member into the cluster.
	MemberAdd(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error)

	// MemberAddAsLearner adds a new learner member into the cluster.
	MemberAddAsLearner(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error)

	// MemberRemove removes an existing member from the cluster.
	MemberRemove(ctx context.Context, id uint64) (*MemberRemoveResponse, error)

	// MemberUpdate updates the peer addresses of the member.
	MemberUpdate(ctx context.Context, id uint64, peerAddrs []string) (*MemberUpdateResponse, error)

	// MemberPromote promotes a member from raft learner (non-voting) to raft voting member.
	MemberPromote(ctx context.Context, id uint64) (*MemberPromoteResponse, error)
}

type cluster struct {
	remote   pb.ClusterClient
	callOpts []grpc.CallOption
}

func NewCluster(c *Client) Cluster {
	api := &cluster{remote: RetryClusterClient(c)}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func NewClusterFromClusterClient(remote pb.ClusterClient, c *Client) Cluster {
	api := &cluster{remote: remote}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (c *cluster) MemberAdd(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error) {
	return c.memberAdd(ctx, peerAddrs, false)
}

func (c *cluster) MemberAddAsLearner(ctx context.Context, peerAddrs []string) (*MemberAddResponse, error) {
	return c.memberAdd(ctx, peerAddrs, true)
}

func (c *cluster) memberAdd(ctx context.Context, peerAddrs []string, isLearner bool) (*MemberAddResponse, error) {
	// fail-fast before panic in rafthttp
	if _, err := v3types.NewURLs(peerAddrs); err != nil {
		return nil, err
	}

	r := &pb.MemberAddRequest{
		PeerURLs:  peerAddrs,
		IsLearner: isLearner,
	}
	resp, err := c.remote.MemberAdd(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberAddResponse)(resp), nil
}

func (c *cluster) MemberRemove(ctx context.Context, id uint64) (*MemberRemoveResponse, error) {
	r := &pb.MemberRemoveRequest{ID: id}
	resp, err := c.remote.MemberRemove(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberRemoveResponse)(resp), nil
}

func (c *cluster) MemberUpdate(ctx context.Context, id uint64, peerAddrs []string) (*MemberUpdateResponse, error) {
	// fail-fast before panic in rafthttp
	if _, err := v3types.NewURLs(peerAddrs); err != nil {
		return nil, err
	}

	// it is safe to retry on update.
	r := &pb.MemberUpdateRequest{ID: id, PeerURLs: peerAddrs}
	resp, err := c.remote.MemberUpdate(ctx, r, c.callOpts...)
	if err == nil {
		return (*MemberUpdateResponse)(resp), nil
	}
	return nil, toErr(ctx, err)
}

func (c *cluster) MemberList(ctx context.Context) (*MemberListResponse, error) {
	// it is safe to retry on list.
	resp, err := c.remote.MemberList(ctx, &pb.MemberListRequest{Linearizable: true}, c.callOpts...)
	if err == nil {
		return (*MemberListResponse)(resp), nil
	}
	return nil, toErr(ctx, err)
}

func (c *cluster) MemberPromote(ctx context.Context, id uint64) (*MemberPromoteResponse, error) {
	r := &pb.MemberPromoteRequest{ID: id}
	resp, err := c.remote.MemberPromote(ctx, r, c.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*MemberPromoteResponse)(resp), nil
}
