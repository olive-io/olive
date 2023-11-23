// Copyright 2023 Lack (xingyys@gmail.com).
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
	"fmt"
	"path"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/dustin/go-humanize"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/idutil"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type registry struct {
	lg *zap.Logger

	idGen  *idutil.Generator
	v3cli  *clientv3.Client
	prefix string

	mu      sync.RWMutex
	runners map[uint64]*pb.Runner
}

func (s *Server) newRegistry() (*registry, error) {
	idKey := path.Join(runtime.DefaultOliveMeta, "runner", "ids")
	idGen, err := idutil.NewGenerator(idKey, s.v3cli)
	if err != nil {
		return nil, errors.Wrap(err, "new id generator")
	}

	prefix := path.Join(runtime.DefaultOliveMeta, "runners")
	runners := make(map[uint64]*pb.Runner)
	resp, _ := s.v3cli.Get(s.ctx, prefix, clientv3.WithPrefix())
	if resp != nil && len(resp.Kvs) > 0 {
		for _, kv := range resp.Kvs {
			runner := new(pb.Runner)
			if err = runner.Unmarshal(kv.Value); err != nil {
				s.lg.Error("unmarshal runner data",
					zap.Error(err))
				s.v3cli.Delete(s.ctx, string(kv.Key))
				continue
			}
			runners[runner.Id] = runner
		}
	}

	r := &registry{
		lg:      s.lg,
		idGen:   idGen,
		v3cli:   s.v3cli,
		prefix:  prefix,
		runners: runners,
	}

	return r, nil
}

func (r *registry) Register(ctx context.Context, runner *pb.Runner) (uint64, error) {
	id := runner.Id
	if id == 0 {
		id = r.idGen.Next(ctx)
	}

	data, _ := runner.Marshal()
	key := path.Join(r.prefix, fmt.Sprintf("%d", id))
	_, err := r.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return 0, errors.Wrap(err, "put runner")
	}

	r.lg.Info("olive-runner registered",
		zap.Uint64("id", runner.Id),
		zap.String("advertise-listen", runner.AdvertiseListen),
		zap.String("peer-listen", runner.PeerListen),
		zap.Uint32("cpu-core", runner.Cpus),
		zap.String("memory", humanize.IBytes(runner.Memory)),
		zap.String("version", runner.Version))

	r.mu.Lock()
	r.runners[id] = runner
	r.mu.Unlock()

	return id, nil
}

func (r *registry) Get(ctx context.Context, id uint64) (*pb.Runner, bool) {
	r.mu.RLock()
	runner, ok := r.runners[id]
	r.mu.RUnlock()

	if ok {
		return runner, true
	}

	key := path.Join(r.prefix, fmt.Sprintf("%d", id))
	resp, err := r.v3cli.Get(ctx, key)
	if err != nil || len(resp.Kvs) == 0 {
		return nil, false
	}
	runner = &pb.Runner{}
	if err = runner.Unmarshal(resp.Kvs[0].Value); err != nil {
		r.lg.Error("unmarshal runner data",
			zap.Uint64("id", id),
			zap.Error(err))
		r.v3cli.Delete(ctx, key)
		return nil, false
	}

	r.mu.Lock()
	r.runners[id] = runner
	r.mu.Unlock()

	return runner, true
}

func (s *Server) RegistryRunner(ctx context.Context, req *pb.RegistryRunnerRequest) (resp *pb.RegistryRunnerResponse, err error) {
	if req.Runner == nil {
		return resp, status.New(codes.InvalidArgument, "missing runner").Err()
	}

	resp = &pb.RegistryRunnerResponse{}

	resp.Id, err = s.registry.Register(ctx, req.Runner)
	return
}

func (s *Server) ReportRunner(ctx context.Context, req *pb.ReportRunnerRequest) (resp *pb.ReportRunnerResponse, err error) {
	resp = &pb.ReportRunnerResponse{}
	return
}
