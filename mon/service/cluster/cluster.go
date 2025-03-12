/*
Copyright 2025 The olive Authors

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

package cluster

import (
	"context"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

type Service struct {
	ctx context.Context
	lg  *zap.Logger

	v3cli *clientv3.Client
}

func New(ctx context.Context, lg *zap.Logger, v3cli *clientv3.Client) (*Service, error) {
	s := &Service{
		ctx:   ctx,
		lg:    lg,
		v3cli: v3cli,
	}

	return s, nil
}

func (rpc *Service) MemberAdd(ctx context.Context, peers []string, isLearner bool) (*types.ResponseHeader, *types.Member, []*types.Member, error) {
	var resp *clientv3.MemberAddResponse
	var err error
	if isLearner {
		resp, err = rpc.v3cli.MemberAddAsLearner(ctx, peers)
	} else {
		resp, err = rpc.v3cli.MemberAdd(ctx, peers)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
	}

	return types.PbToHeader(resp.Header), types.PbToMember(resp.Member), members, nil
}

func (rpc *Service) MemberRemove(ctx context.Context, id uint64) (*types.ResponseHeader, []*types.Member, error) {
	resp, err := rpc.v3cli.MemberRemove(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
	}
	return types.PbToHeader(resp.Header), members, nil
}

func (rpc *Service) MemberUpdate(ctx context.Context, id uint64, peers []string) (*types.ResponseHeader, []*types.Member, error) {
	resp, err := rpc.v3cli.MemberUpdate(ctx, id, peers)
	if err != nil {
		return nil, nil, err
	}

	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
	}
	return types.PbToHeader(resp.Header), members, nil
}

func (rpc *Service) MemberList(ctx context.Context) (*types.ResponseHeader, []*types.Member, error) {
	resp, err := rpc.v3cli.MemberList(ctx)
	if err != nil {
		return nil, nil, err
	}
	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
	}
	return types.PbToHeader(resp.Header), members, nil
}

func (rpc *Service) MemberPromote(ctx context.Context, id uint64) (*types.ResponseHeader, []*types.Member, error) {
	resp, err := rpc.v3cli.MemberPromote(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
	}
	return types.PbToHeader(resp.Header), members, nil
}
