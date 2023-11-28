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
	"math"
	"strings"
	"sync"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/meta/leader"
	"github.com/olive-io/olive/meta/schedule"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type scheduleLimit struct {
	RegionLimit     int
	DefinitionLimit int
}

type scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	v3cli    *clientv3.Client
	notifier leader.Notifier

	rmu     sync.RWMutex
	runners map[uint64]*pb.Runner

	rgmu    sync.RWMutex
	regions map[uint64]*pb.Region

	rumu        sync.Mutex
	runnerQueue *schedule.PriorityQueue[*pb.RunnerStat]

	remu        sync.Mutex
	regionQueue *schedule.PriorityQueue[*pb.RegionStat]

	limit scheduleLimit

	stopping <-chan struct{}
}

func (s *Server) newScheduler() *scheduler {
	ctx, cancel := context.WithCancel(s.ctx)

	sc := &scheduler{
		ctx:      ctx,
		cancel:   cancel,
		lg:       s.lg,
		v3cli:    s.v3cli,
		notifier: s.notifier,
		runners:  map[uint64]*pb.Runner{},
		regions:  map[uint64]*pb.Region{},
		limit: scheduleLimit{
			RegionLimit:     s.cfg.RegionLimit,
			DefinitionLimit: s.cfg.RegionDefinitionLimit,
		},
		stopping: s.StoppingNotify(),
	}

	runnerStatQueue := schedule.New[*pb.RunnerStat](sc.runnerGetter())
	sc.runnerQueue = runnerStatQueue

	regionStateQueue := schedule.New[*pb.RegionStat](sc.regionGetter())
	sc.regionQueue = regionStateQueue

	return sc
}

func (sc *scheduler) runnerGetter() schedule.ChaosFn[*pb.RunnerStat] {
	return func(stat *pb.RunnerStat) int {
		sc.rmu.RLock()
		runner, ok := sc.runners[stat.Id]
		sc.rmu.RUnlock()
		if !ok {
			return math.MaxInt64
		}
		cpus := float64(runner.Cpu)
		memory := float64(runner.Memory / 1024 / 1024)

		chaos := int((100-stat.CpuPer)/100*cpus)%30 +
			int((100-stat.MemoryPer)/100*memory)%30 +
			(sc.limit.RegionLimit - len(stat.Regions)%30) +
			(sc.limit.RegionLimit - len(stat.Leaders)%10)

		return chaos
	}
}

func (sc *scheduler) regionGetter() schedule.ChaosFn[*pb.RegionStat] {
	return func(stat *pb.RegionStat) int {
		sc.rmu.RLock()
		_, ok := sc.regions[stat.Id]
		sc.rmu.RUnlock()
		if !ok {
			return math.MaxInt64
		}

		chaos := int(float64(stat.Definitions) / float64(sc.limit.DefinitionLimit) * 100)

		return chaos
	}
}

func (sc *scheduler) sync() error {
	ctx := sc.ctx
	client := sc.v3cli
	runners := make(map[uint64]*pb.Runner)
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	key := runtime.DefaultMetaRunnerRegistry
	resp, err := client.Get(ctx, key, options...)
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		runner := new(pb.Runner)
		err = runner.Unmarshal(kv.Value)
		if err != nil {
			continue
		}
		runners[runner.Id] = runner
	}
	sc.rmu.Lock()
	sc.runners = runners
	sc.rmu.Unlock()

	regions := make(map[uint64]*pb.Region)
	key = runtime.DefaultMetaRegionStat
	resp, err = client.Get(ctx, key, options...)
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		region := new(pb.Region)
		err = region.Unmarshal(kv.Value)
		if err != nil {
			continue
		}
		regions[region.Id] = region
	}
	sc.rgmu.Lock()
	sc.regions = regions
	sc.rgmu.Unlock()
	return nil
}

func (sc *scheduler) Start() error {
	if err := sc.sync(); err != nil {
		return err
	}

	go sc.run()
	return nil
}

func (sc *scheduler) schedulingRunnerCycle() error {
	return nil
}

func (sc *scheduler) waitUtilLeader() bool {
	for {
		if sc.notifier.IsLeader() {
			<-sc.notifier.ReadyNotify()
			return true
		}

		select {
		case <-sc.stopping:
			return false
		case <-sc.notifier.ChangeNotify():
		}
	}
}

func (sc *scheduler) run() {
	for {
		if !sc.waitUtilLeader() {
			select {
			case <-sc.stopping:
				return
			default:
			}
			continue
		}

		//TODO: handle regions (expend raft shard replica)

		ctx := sc.ctx
		prefix := runtime.DefaultMetaRunnerPrefix
		options := []clientv3.OpOption{
			clientv3.WithPrefix(),
		}
		wch := sc.v3cli.Watch(ctx, prefix, options...)

	LOOP:
		for {
			select {
			case <-sc.stopping:
				return
			case <-sc.notifier.ChangeNotify():
				break LOOP
			case wr := <-wch:
				if wr.Canceled {
					break LOOP
				}

				for _, event := range wr.Events {
					sc.processEvent(event)
				}
			}
		}
	}
}

func (sc *scheduler) processEvent(event *clientv3.Event) {
	lg := sc.lg
	kv := event.Kv
	key := string(kv.Key)
	switch {
	case strings.HasPrefix(key, runtime.DefaultMetaRunnerStat):
		rs := new(pb.RunnerStat)
		if err := rs.Unmarshal(kv.Value); err != nil {
			lg.Error("unmarshal RunnerState", zap.Error(err))
			return
		}

		sc.handleRunnerStat(rs)
	case strings.HasPrefix(key, runtime.DefaultMetaRegionStat):
		rs := new(pb.RegionStat)
		if err := rs.Unmarshal(kv.Value); err != nil {
			lg.Error("unmarshal RegionState", zap.Error(err))
			return
		}

		sc.handleRegionStat(rs)
	case strings.HasPrefix(key, runtime.DefaultMetaRunnerRegistry):
		runner := new(pb.Runner)
		if err := runner.Unmarshal(kv.Value); err != nil {
			lg.Error("unmarshal Runner", zap.Error(err))
			return
		}

		sc.handleRunner(runner)
	}
}

func (sc *scheduler) handleRunnerStat(stat *pb.RunnerStat) {
	sc.rumu.Lock()
	defer sc.rumu.Unlock()
	sc.runnerQueue.Set(stat)
}

func (sc *scheduler) handleRegionStat(stat *pb.RegionStat) {
	sc.remu.Lock()
	defer sc.remu.Unlock()
	sc.regionQueue.Set(stat)
}

func (sc *scheduler) handleRunner(runner *pb.Runner) {
	if runner.Id == 0 {
		return
	}

	sc.rmu.Lock()
	defer sc.rmu.Unlock()
	sc.runners[runner.Id] = runner
}
