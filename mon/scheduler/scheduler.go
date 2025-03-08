/*
Copyright 2023 The olive Authors

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

package scheduler

import (
	"context"
	"fmt"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/pkg/queue"
)

const processSchPrefix = "/olive/mon/schedule/process"

type Process struct {
	*types.ProcessSnapshot
}

func NewProcess(ps *types.ProcessSnapshot) *Process {
	p := &Process{
		ProcessSnapshot: ps,
	}
	return p
}

func (p *Process) ID() int64 {
	return int64(p.Id)
}

func DefaultProcessStore(p *Process) int64 {
	return p.Priority
}

type Scheduler interface {
	AddProcess(ctx context.Context, pi *types.ProcessInstance)
	NextOne(ctx context.Context) (*types.Runner, bool)
	Start(ctx context.Context) error
}

type scheduler struct {
	lg *zap.Logger

	v3cli    *clientv3.Client
	notifier leader.Notifier

	leaderFlag *atomic.Bool

	smu       sync.RWMutex
	snapshots map[uint64]*snapshot

	processQ *queue.SyncPriorityQueue[*Process]
}

func New(lg *zap.Logger, v3cli *clientv3.Client, notifier leader.Notifier) (Scheduler, error) {

	processQ := queue.NewSync[*Process](DefaultProcessStore)

	isLeader := new(atomic.Bool)
	isLeader.Store(false)

	sch := &scheduler{
		lg: lg,

		v3cli:      v3cli,
		notifier:   notifier,
		leaderFlag: isLeader,

		snapshots: make(map[uint64]*snapshot),

		processQ: processQ,
	}

	return sch, nil
}

func (sc *scheduler) AddProcess(ctx context.Context, pi *types.ProcessInstance) {
	ps := pi.ToSnapshot()
	if sc.isLeader() {
		sc.processQ.Push(NewProcess(ps))
	} else {
		sc.addProcessSnapshot(ctx, ps)
	}
}

// NextOne returns an *types.Runner with the highest score
func (sc *scheduler) NextOne(ctx context.Context) (*types.Runner, bool) {
	snapshots := make([]*snapshot, 0, len(sc.snapshots))
	sc.smu.RLock()
	for _, rs := range sc.snapshots {
		if rs.Active() {
			snapshots = append(snapshots, rs.DeepCopy())
		}
	}
	sc.smu.RUnlock()

	if len(snapshots) == 0 {
		return nil, false
	}

	sort.Slice(snapshots, func(i, j int) bool {
		s1 := snapshots[i]
		s2 := snapshots[j]

		return defaultSnapshotScore(s1) > defaultSnapshotScore(s2)
	})

	return snapshots[0].Runner, true
}

func (sc *scheduler) Start(ctx context.Context) error {
	go sc.watchRunners(ctx)
	go sc.process(ctx)
	return nil
}

func (sc *scheduler) process(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interval := time.Millisecond * 50
	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		if !sc.waitLeaderLoop(ctx) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		sc.leaderFlag.Store(true)

		tick.Reset(interval)

		sc.fetchPrepares(ctx)

		opts := []clientv3.OpOption{
			clientv3.WithPrefix(),
		}

		prefix := processSchPrefix
		w := sc.v3cli.Watch(ctx, prefix, opts...)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case <-sc.notifier.ChangeNotify():
				// follow to leader
				if sc.waitLeaderLoop(ctx) {
					sc.lg.Info("become the new term leader, fetch all prepare process")
					sc.fetchPrepares(ctx)
				}
				break LOOP

			case wch, ok := <-w:
				if !ok {
					break LOOP
				}

				if wch.Err() != nil {
					break LOOP
				}

				for _, ev := range wch.Events {
					switch ev.Type {
					case mvccpb.PUT:
					}
				}

			case <-tick.C:

				pv, exists := sc.processQ.Pop()
				if !exists {
					continue
				}
				process := pv.(*Process)

				runner, exists := sc.NextOne(ctx)
				if !exists {
					sc.processQ.Push(process)
					continue
				}

				event := &types.RunnerEvent{
					Type: types.EventType_ExecuteProcess,
					ExecuteProcess: &types.ExecuteProcessEvent{
						Process: process.ID(),
					},
				}
				sc.dispatchEvent(ctx, runner.Id, event)

				ps := process.ProcessSnapshot
				ps.Status = types.ProcessStatus_Ready
				sc.removeProcessSnapshot(ctx, ps)
			}
		}
	}
}

func (sc *scheduler) addProcessSnapshot(ctx context.Context, ps *types.ProcessSnapshot) {
	value, err := proto.Marshal(ps)
	if err != nil {
		return
	}

	key := path.Join(processSchPrefix, fmt.Sprintf("%d", ps.Id))
	_, err = sc.v3cli.Put(ctx, key, string(value))
	if err != nil {
		sc.lg.Error("failed to save process snapshot",
			zap.Int64("process", ps.Id),
			zap.Error(err))
	}
}

func (sc *scheduler) removeProcessSnapshot(ctx context.Context, ps *types.ProcessSnapshot) {
	key := path.Join(processSchPrefix, fmt.Sprintf("%d", ps.Id))
	_, err := sc.v3cli.Delete(ctx, key)
	if err != nil {
		sc.lg.Error("failed to remove process snapshot",
			zap.Int64("process", ps.Id),
			zap.Error(err))
	}
}

func (sc *scheduler) fetchPrepares(ctx context.Context) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	prefix := processSchPrefix
	resp, err := sc.v3cli.Get(ctx, prefix, opts...)
	if err != nil {
		return
	}

	for _, kv := range resp.Kvs {
		ps, err := parsePSnapKV(kv)
		if err != nil {
			continue
		}

		if ps.Status != types.ProcessStatus_Prepare {
			continue
		}

		key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", ps.Id))
		resp, err = sc.v3cli.Get(ctx, key, clientv3.WithSerializable())
		if err != nil {
			continue
		}
		pi, err := parseProcessKV(resp.Kvs[0])
		if err != nil {
			continue
		}
		if pi.Status != types.ProcessStatus_Prepare {
			continue
		}

		sc.processQ.Push(NewProcess(ps))
	}
}

func (sc *scheduler) dispatchEvent(ctx context.Context, runner uint64, event *types.RunnerEvent) {
	key := path.Join(api.RunnerTopic, fmt.Sprintf("%d", runner))
	value, err := proto.Marshal(event)
	if err != nil {
		sc.lg.Error("failed to marshal event", zap.Error(err))
	}
	_, err = sc.v3cli.Put(ctx, key, string(value))
	if err != nil {
		sc.lg.Error("failed to dispatch event", zap.Error(err))
	}
}

func (sc *scheduler) waitLeaderOnce(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-sc.notifier.ReadyNotify():
	}

	return sc.notifier.IsLeader()
}

func (sc *scheduler) waitLeaderLoop(ctx context.Context) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-sc.notifier.ReadyNotify():
		}

		if sc.notifier.IsLeader() {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-sc.notifier.ChangeNotify():
		}
	}
}

func (sc *scheduler) isLeader() bool {
	return sc.leaderFlag.Load()
}
