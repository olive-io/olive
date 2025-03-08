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
	"fmt"
	"path"
	"strconv"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
)

const prefix = api.RunnerPrefix

type Service struct {
	ctx context.Context
	lg  *zap.Logger

	v3cli *clientv3.Client
	idGen *idutil.Generator
}

func New(ctx context.Context, lg *zap.Logger, v3cli *clientv3.Client, idGen *idutil.Generator) (*Service, error) {
	s := &Service{
		ctx:   ctx,
		lg:    lg,
		v3cli: v3cli,
		idGen: idGen,
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

func (s *Service) Register(ctx context.Context, runner *types.Runner) (*types.Runner, error) {

	if runner.Id == 0 {
		runner.Id = s.idGen.Next()

		s.lg.Info("register new runner",
			zap.Uint64("id", runner.Id),
			zap.String("version", runner.Version),
			zap.String("listen", runner.ListenURL),
		)
	}

	runnerKey := path.Join(prefix, strconv.FormatUint(runner.Id, 10))
	runnerBytes, err := proto.Marshal(runner)
	if err != nil {
		return nil, err
	}

	opts := []clientv3.OpOption{}
	_, err = s.v3cli.Put(ctx, runnerKey, string(runnerBytes), opts...)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func (s *Service) Disregister(ctx context.Context, runner *types.Runner) (*types.Runner, error) {

	s.lg.Info("disregister new runner",
		zap.Uint64("id", runner.Id),
		zap.String("version", runner.Version),
		zap.String("listen", runner.ListenURL),
	)

	runnerKey := path.Join(prefix, strconv.FormatUint(runner.Id, 10))

	opts := []clientv3.OpOption{}
	_, err := s.v3cli.Delete(ctx, runnerKey, opts...)
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func (s *Service) Heartbeat(ctx context.Context, stat *types.RunnerStat) error {
	statKey := path.Join(api.RunnerStatPrefix, fmt.Sprintf("%d", stat.Id))
	statBytes, err := proto.Marshal(stat)
	if err != nil {
		return err
	}

	opts := []clientv3.OpOption{}
	_, err = s.v3cli.Put(ctx, statKey, string(statBytes), opts...)
	if err != nil {
		return err
	}

	return nil
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

func (s *Service) GetRunner(ctx context.Context, id uint64) (*types.Runner, *types.RunnerStat, error) {

	runnerKey := path.Join(prefix, strconv.FormatUint(id, 10))

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

	var runner *types.Runner
	if err = proto.Unmarshal(resp.Kvs[0].Value, runner); err != nil {
		return nil, nil, err
	}

	statKey := path.Join(api.RunnerStatPrefix, fmt.Sprintf("%d", id))
	resp, err = s.v3cli.Get(ctx, statKey, opts...)
	if err != nil {
		return nil, nil, err
	}
	if resp.Kvs == nil || len(resp.Kvs) == 0 {
		return nil, nil, status.New(codes.NotFound, "stat not found").Err()
	}

	var stat *types.RunnerStat
	if err = proto.Unmarshal(resp.Kvs[0].Value, stat); err != nil {
		return nil, nil, err
	}

	return runner, stat, nil
}
