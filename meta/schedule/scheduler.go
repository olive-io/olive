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

package schedule

import (
	"context"
	"fmt"
	"math"
	"path"
	"strings"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/meta/leader"
	"github.com/olive-io/olive/meta/schedule/queue"
	"github.com/olive-io/olive/pkg/idutil"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	messageParallel = 20
	replicaNum      = 3

	defaultRegionElectionTTL  = 10
	defaultRegionHeartbeatTTL = 2
)

type Limit struct {
	RegionLimit     int
	DefinitionLimit int
}

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	v3cli    *clientv3.Client
	notifier leader.Notifier

	rmu     sync.RWMutex
	runners map[uint64]*pb.Runner

	rgmu    sync.RWMutex
	regions map[uint64]*pb.Region

	runnerQ     *queue.SyncPriorityQueue[*pb.RunnerStat]
	regionQ     *queue.SyncPriorityQueue[*pb.RegionStat]
	definitionQ *queue.SyncPriorityQueue[*pb.DefinitionMeta]

	messageCh chan imessage

	limit Limit

	stopping <-chan struct{}
}

func New(
	ctx context.Context, logger *zap.Logger,
	client *clientv3.Client, notifier leader.Notifier,
	limit Limit, stopping <-chan struct{}) *Scheduler {

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	sc := &Scheduler{
		ctx:       ctx,
		cancel:    cancel,
		lg:        logger,
		v3cli:     client,
		notifier:  notifier,
		runners:   map[uint64]*pb.Runner{},
		regions:   map[uint64]*pb.Region{},
		messageCh: make(chan imessage, messageParallel),
		limit:     limit,
		stopping:  stopping,
	}

	sc.runnerQ = queue.NewSync[*pb.RunnerStat](sc.runnerGetter())
	sc.regionQ = queue.NewSync[*pb.RegionStat](sc.regionGetter())
	sc.definitionQ = queue.NewSync[*pb.DefinitionMeta](func(v *pb.DefinitionMeta) int64 {
		return v.StartRev
	})

	return sc
}

func (sc *Scheduler) runnerGetter() queue.ScoreFn[*pb.RunnerStat] {
	return func(stat *pb.RunnerStat) int64 {
		sc.rmu.RLock()
		runner, ok := sc.runners[stat.Id]
		sc.rmu.RUnlock()
		if !ok {
			return math.MaxInt64
		}
		cpus := float64(runner.Cpu)
		memory := float64(runner.Memory / 1024 / 1024)

		score := int((100-stat.CpuPer)/100*cpus)%30 +
			int((100-stat.MemoryPer)/100*memory)%30 +
			(sc.limit.RegionLimit-len(stat.Regions))%30 +
			(sc.limit.RegionLimit-len(stat.Leaders))%10

		return int64(score)
	}
}

func (sc *Scheduler) regionGetter() queue.ScoreFn[*pb.RegionStat] {
	return func(stat *pb.RegionStat) int64 {
		sc.rmu.RLock()
		_, ok := sc.regions[stat.Id]
		sc.rmu.RUnlock()
		if !ok {
			return math.MaxInt64
		}

		score := int64(float64(stat.Definitions)/float64(sc.limit.DefinitionLimit)*100)%70 +
			int64(stat.Replicas*10)%30

		return score
	}
}

func (sc *Scheduler) sync() error {
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
	key = runtime.DefaultRunnerRegion
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

func (sc *Scheduler) Start() error {
	if err := sc.sync(); err != nil {
		return err
	}

	go sc.run()
	return nil
}

func (sc *Scheduler) dispatchMessage(m imessage) {
	select {
	case <-sc.stopping:
		return
	case sc.messageCh <- m:
	}
}

func (sc *Scheduler) AllocRegion(ctx context.Context) (*pb.Region, error) {
	runners, err := sc.schedulingRunnerCycle(ctx)
	if err != nil {
		return nil, err
	}

	// allocates a new region id
	rid, err := sc.allocRegionId(ctx)
	if err != nil {
		return nil, err
	}
	rname := fmt.Sprintf("r%.2d", rid)

	region := &pb.Region{
		Id:           rid,
		Name:         rname,
		Members:      map[uint64]uint64{},
		Replicas:     map[uint64]*pb.RegionReplica{},
		ElectionRTT:  defaultRegionElectionTTL,
		HeartbeatRTT: defaultRegionHeartbeatTTL,
	}
	for i, runner := range runners {
		mid := uint64(i + 1)
		if i == 0 {
			region.Leader = mid
		}
		region.Members[mid] = runner.Id
		region.Replicas[mid] = &pb.RegionReplica{
			Runner: runner.Id,
			Region: rid,
			Id:     mid,
		}
	}

	key := path.Join(runtime.DefaultRunnerRegion, fmt.Sprintf("%d", region.Id))
	data, _ := region.Marshal()
	resp, err := sc.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return nil, err
	}
	region.Rev = resp.Header.Revision

	sc.rgmu.Lock()
	sc.regions[region.Id] = region
	sc.rgmu.Unlock()

	return region, nil
}

func (sc *Scheduler) allocRegionId(ctx context.Context) (uint64, error) {
	idGen, err := idutil.NewGenerator(ctx, runtime.DefaultMetaRegionRegistryId, sc.v3cli)
	if err != nil {
		return 0, err
	}
	return idGen.Next(), nil
}

func (sc *Scheduler) BindRegion(ctx context.Context, dm *pb.DefinitionMeta) (*pb.Region, bool, error) {
	region, has, err := sc.schedulingRegionCycle(ctx)
	if err != nil {
		return nil, false, err
	}
	if !has {
		go func() {
			//definition add to schedule queue
			sc.definitionQ.Set(dm)

			sc.rmu.RLock()
			regionTotal := len(sc.regions)
			sc.rmu.RUnlock()

			if regionTotal == 0 {
				sc.lg.Info("dispatch message, allocates a new region")
				sc.dispatchMessage(new(regionAllocMessage))
			}
		}()
		return nil, false, nil
	}

	// Expends the replica of region to 3
	if len(region.Members) < replicaNum && sc.runnerQ.Len() >= replicaNum {
		sc.dispatchMessage(new(regionExpendMessage))
	}

	dm.Region = region.Id
	key := path.Join(runtime.DefaultMetaDefinitionMeta, dm.Id)
	data, _ := dm.Marshal()
	_, err = sc.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return nil, false, err
	}

	return region, true, nil
}

func (sc *Scheduler) schedulingRunnerCycle(ctx context.Context) ([]*pb.Runner, error) {
	lg := sc.lg

	length := sc.runnerQ.Len()
	if length == 0 {
		return nil, ErrRunnerNotReady
	}

	rc := replicaNum
	if length < replicaNum {
		rc = length
	}
	runners := make([]*pb.Runner, rc)
	rts := make([]*pb.RunnerStat, rc)
	for i := 0; i < rc; i++ {
		rs, ok := sc.runnerQ.Pop()
		if !ok {
			sc.lg.Panic("call runnerQ.Pop()")
		}
		rts[i] = rs.(*pb.RunnerStat)
	}

	recycle := func(list *[]*pb.RunnerStat) {
		for i := range *list {
			rs := (*list)[i]
			sc.runnerQ.Set(rs)
		}
	}
	defer recycle(&rts)

	if score := sc.runnerGetter()(rts[0]); score > 100 {
		return nil, ErrRunnerBusy
	}

	sc.rmu.RLock()
	for i, rs := range rts {
		runner, ok := sc.runners[rs.Id]
		if !ok {
			lg.Warn("RunnerStat is invalid", zap.Uint64("id", rs.Id))
			continue
		}
		runners[i] = runner
	}
	sc.rmu.RUnlock()

	return runners, nil
}

func (sc *Scheduler) schedulingRegionCycle(ctx context.Context) (*pb.Region, bool, error) {
	if sc.regionQ.Len() == 0 {
		return nil, false, nil
	}
	x, ok := sc.regionQ.Pop()
	if !ok {
		return nil, false, nil
	}
	rs := x.(*pb.RegionStat)

	sc.rgmu.RLock()
	region, has := sc.regions[rs.Id]
	sc.rgmu.RUnlock()
	if !has {
		return nil, false, nil
	}

	if int(region.Definitions) >= sc.limit.DefinitionLimit {
		sc.dispatchMessage(new(regionAllocMessage))
		return nil, false, ErrRegionNoSpace
	}

	region.Definitions += 1

	key := path.Join(runtime.DefaultRunnerRegion, fmt.Sprintf("%d", region.Id))
	data, _ := region.Marshal()
	if _, err := sc.v3cli.Put(ctx, key, string(data)); err != nil {
		return nil, false, err
	}

	sc.rgmu.Lock()
	sc.regions[rs.Id] = region
	sc.rgmu.Unlock()

	return region, true, nil
}

func (sc *Scheduler) waitUtilLeader() bool {
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

func (sc *Scheduler) run() {
	tickDuration := time.Second
	ticker := time.NewTimer(tickDuration)
	defer ticker.Stop()

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

		ticker.Reset(time.Second)

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

			case message := <-sc.messageCh:
				sc.processMessage(message)

			case <-ticker.C:
				ticker.Reset(tickDuration)

				x, has := sc.definitionQ.Pop()
				if !has {
					break
				}
				dm := x.(*pb.DefinitionMeta)

				if _, _, err := sc.BindRegion(sc.ctx, dm); err != nil {
					sc.lg.Error("binding region",
						zap.String("definition", dm.Id),
						zap.Error(err))
				}
			}
		}
	}
}

func (sc *Scheduler) processEvent(event *clientv3.Event) {
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

func (sc *Scheduler) handleRunnerStat(stat *pb.RunnerStat) {
	sc.runnerQ.Set(stat)
}

func (sc *Scheduler) handleRegionStat(stat *pb.RegionStat) {
	sc.regionQ.Set(stat)
}

func (sc *Scheduler) handleRunner(runner *pb.Runner) {
	if runner.Id == 0 {
		return
	}

	sc.rmu.Lock()
	defer sc.rmu.Unlock()
	sc.runners[runner.Id] = runner
}

func (sc *Scheduler) processMessage(m imessage) {
	switch m.(type) {
	case *regionAllocMessage:
		sc.processAllocRegion(sc.ctx)
	case *regionExpendMessage:
	case *regionMigrateMessage:
	}
}

func (sc *Scheduler) processAllocRegion(ctx context.Context) {
	lg := sc.lg
	region, err := sc.AllocRegion(ctx)
	if err != nil {
		lg.Error("allocate region", zap.Error(err))
		return
	}

	lg.Info("allocate new region", zap.Stringer("region", region))
}
