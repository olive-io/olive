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

package system

import (
	"context"
	"path/filepath"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api/types"
)

const prefix = "/olive/v1/runners"

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

func (s *Service) GetCluster(ctx context.Context) (*types.ResponseHeader, *types.Monitor, error) {
	resp, err := s.v3cli.MemberList(ctx)
	if err != nil {
		return nil, nil, err
	}

	header := types.PbToHeader(resp.Header)

	leaderId := uint64(0)
	members := make([]*types.Member, len(resp.Members))
	for i, mem := range resp.Members {
		members[i] = types.PbToMember(mem)
		statusRsp, err := s.v3cli.Status(ctx, mem.ClientURLs[0])
		if err != nil {
			return nil, nil, err
		}
		if statusRsp.Leader == mem.ID {
			leaderId = statusRsp.Leader
		}
	}

	monitor := &types.Monitor{
		ClusterId: resp.Header.GetClusterId(),
		Leader:    leaderId,
		Members:   members,
	}

	return header, monitor, nil
}

func (s *Service) Registry(ctx context.Context, runner *types.Runner, stat *types.RunnerStat) (*types.Runner, error) {

	runnerKey := filepath.Join(prefix, runner.Name)
	runnerBytes, err := proto.Marshal(runner)
	if err != nil {
		return nil, err
	}

	opts := []clientv3.OpOption{}
	_, err = s.v3cli.Put(ctx, runnerKey, string(runnerBytes), opts...)
	if err != nil {
		return nil, err
	}

	statKey := filepath.Join(prefix, "stat")
	statBytes, err := proto.Marshal(stat)
	if err != nil {
		return nil, err
	}

	opts = []clientv3.OpOption{}
	_, err = s.v3cli.Put(ctx, statKey, string(statBytes), opts...)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func (s *Service) ListRunners(ctx context.Context) ([]*types.Runner, error) {

	key := prefix
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := s.v3cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	runners := make([]*types.Runner, 0)
	for _, kv := range resp.Kvs {
		var runner types.Runner
		if err = proto.Unmarshal(kv.Value, &runner); err != nil {
			continue
		}
		runners = append(runners, &runner)
	}

	return runners, nil
}

func (s *Service) GetRunner(ctx context.Context, name string) (*types.Runner, *types.RunnerStat, error) {
	var runner *types.Runner
	var stat *types.RunnerStat

	runnerKey := filepath.Join(prefix, name)

	opts := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}
	resp, err := s.v3cli.Get(ctx, runnerKey, opts...)
	if err != nil {
		return nil, nil, err
	}

	if resp.Kvs == nil || len(resp.Kvs) == 0 {
		return nil, nil, status.New(codes.NotFound, "runner not found").Err()
	}

	if err = proto.Unmarshal(resp.Kvs[0].Value, runner); err != nil {
		return nil, nil, err
	}

	statKey := filepath.Join(runnerKey, "stat")
	resp, err = s.v3cli.Get(ctx, statKey)
	if err != nil {
		return nil, nil, err
	}
	if resp.Kvs == nil || len(resp.Kvs) == 0 {
		return nil, nil, status.New(codes.NotFound, "stat not found").Err()
	}
	if err = proto.Unmarshal(resp.Kvs[0].Value, stat); err != nil {
		return nil, nil, err
	}

	return runner, stat, nil
}
