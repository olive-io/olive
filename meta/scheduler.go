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
	"strings"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/meta/leader"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

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
		stopping: s.StoppingNotify(),
	}

	return sc
}

func (sc *scheduler) Start() error {
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

	go sc.run()
	return nil
}

func (sc *scheduler) schedulingCycle() error {
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
	interval := time.Millisecond * 500
	for {
		if !sc.waitUtilLeader() {
			time.Sleep(interval)
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
			var wr clientv3.WatchResponse
			select {
			case <-sc.stopping:
				return
			case <-sc.notifier.ChangeNotify():
				break LOOP
			case wr = <-wch:
			}

			if wr.Canceled {
				break LOOP
			}

			for _, event := range wr.Events {
				sc.processEvent(event)
			}
		}
	}
}

func (sc *scheduler) processEvent(event *clientv3.Event) {
	kv := event.Kv
	key := string(kv.Key)
	switch {
	case strings.HasPrefix(key, runtime.DefaultMetaRunnerStat):
	case strings.HasPrefix(key, runtime.DefaultMetaRegionStat):
	case strings.HasPrefix(key, runtime.DefaultRunnerPrefix):
	}
}
