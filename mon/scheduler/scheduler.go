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
	"path/filepath"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/pkg/queue"
)

const processSchPrefix = "/olive/mon/schedule/process"

type Snapshot struct {
	runner *types.Runner
	stat   *types.RunnerStat
}

func NewSnapshot(runner *types.Runner, stat *types.RunnerStat) *Snapshot {
	snapshot := &Snapshot{
		runner: runner,
		stat:   stat,
	}
	return snapshot
}

func (s *Snapshot) ID() int64 {
	return int64(s.runner.Id)
}

func (s *Snapshot) Active() bool {
	heartbeat := time.Millisecond * time.Duration(s.runner.HeartbeatMs)
	return s.stat.Timestamp+heartbeat.Nanoseconds() > time.Now().UnixNano()
}

func DefaultSnapshotScore(snap *Snapshot) int64 {
	runner := snap.runner
	stat := snap.stat

	score := int64(float64(runner.Cpu)-stat.CpuUsed) +
		int64(float64(runner.Memory)-stat.MemoryUsed) -
		int64(stat.BpmnProcesses+stat.BpmnTasks)

	return score
}

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
	UpdateSnapshot(snap *Snapshot)
	AddProcess(ctx context.Context, pi *types.ProcessInstance)
	NextOne(ctx context.Context) (*types.Runner, bool)
	Start(ctx context.Context) error
}

type scheduler struct {
	lg *zap.Logger

	v3cli    *clientv3.Client
	notifier leader.Notifier

	rmu     sync.RWMutex
	runners map[uint64]*types.Runner

	runnerQ *queue.SyncPriorityQueue[*Snapshot]

	processQ *queue.SyncPriorityQueue[*Process]
}

func New(lg *zap.Logger, v3cli *clientv3.Client, notifier leader.Notifier) (Scheduler, error) {

	runnerQ := queue.NewSync[*Snapshot](DefaultSnapshotScore)
	processQ := queue.NewSync[*Process](DefaultProcessStore)

	sch := &scheduler{
		lg: lg,

		v3cli:    v3cli,
		notifier: notifier,

		rmu:     sync.RWMutex{},
		runners: make(map[uint64]*types.Runner),

		runnerQ: runnerQ,

		processQ: processQ,
	}

	return sch, nil
}

// UpdateSnapshot updates *types.Runner and *types.RunnerStat for priority queue
func (sc *scheduler) UpdateSnapshot(snap *Snapshot) {
	sc.rmu.Lock()
	sc.runners[snap.runner.Id] = snap.runner
	sc.rmu.Unlock()

	sc.runnerQ.Set(snap)
}

func (sc *scheduler) AddProcess(ctx context.Context, pi *types.ProcessInstance) {
	ps := pi.ToSnapshot()
	sc.processQ.Push(NewProcess(ps))
	sc.addProcessSnapshot(ctx, ps)
}

// NextOne returns an *types.Runner with the highest score
func (sc *scheduler) NextOne(ctx context.Context) (*types.Runner, bool) {
	value, exists := sc.runnerQ.Pop()
	if !exists {
		return nil, false
	}

	snap := value.(*Snapshot)
	if !snap.Active() {
		return nil, false
	}

	return snap.runner, true
}

func (sc *scheduler) Start(ctx context.Context) error {
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

		tick.Reset(interval)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case <-sc.notifier.ChangeNotify():
				// follow to leader
				if sc.waitLeaderLoop(ctx) {
					sc.fetchPrepares(ctx)
				}
				break LOOP

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

				snap := process.ProcessSnapshot
				snap.Status = types.ProcessStatus_Ready
				sc.removeProcessSnapshot(ctx, snap)
			}
		}
	}
}

func (sc *scheduler) addProcessSnapshot(ctx context.Context, ps *types.ProcessSnapshot) {
	value, err := proto.Marshal(ps)
	if err != nil {
		return
	}

	key := filepath.Join(processSchPrefix, fmt.Sprintf("%d", ps.Id))
	_, err = sc.v3cli.Put(ctx, key, string(value))
	if err != nil {
		sc.lg.Error("failed to save process snapshot",
			zap.Int64("process", ps.Id),
			zap.Error(err))
	}
}

func (sc *scheduler) removeProcessSnapshot(ctx context.Context, ps *types.ProcessSnapshot) {
	key := filepath.Join(processSchPrefix, fmt.Sprintf("%d", ps.Id))
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
		var process types.ProcessSnapshot
		if err := proto.Unmarshal(kv.Value, &process); err != nil {
			return
		}

		if process.Status == types.ProcessStatus_Prepare {
			sc.processQ.Push(&Process{ProcessSnapshot: &process})
		}
	}
}

func (sc *scheduler) dispatchEvent(ctx context.Context, runner uint64, event *types.RunnerEvent) {
	key := filepath.Join(api.RunnerTopic, fmt.Sprintf("%d", runner))
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
