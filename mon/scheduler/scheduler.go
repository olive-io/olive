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

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/pkg/queue"
)

const (
	processSchPrefix = "/olive/mon/schedule/process"
	// processInterval means the interval of *types.Process from olive-mon scheduled
	// until olive-runner executes
	processInterval = time.Minute
)

var (
	gs   Scheduler
	once sync.Once
)

// StartScheduler start global Scheduler
func StartScheduler(ctx context.Context, lg *zap.Logger, v3cli *clientv3.Client, notifier leader.Notifier) error {
	var err error
	once.Do(func() {
		gs, err = New(lg, v3cli, notifier)
		if err != nil {
			return
		}

		err = gs.Start(ctx)
	})
	return err
}

// AddProcess add bpmn process for global Scheduler
func AddProcess(ctx context.Context, pi *types.Process) {
	gs.AddProcess(ctx, pi)
}

// NextOne returns the *types.Runner with the highest score from global scheduler
func NextOne(ctx context.Context) (*types.Runner, bool) {
	return gs.NextOne(ctx)
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
	AddProcess(ctx context.Context, pi *types.Process)
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

	pmu         sync.RWMutex
	processTree *rbt.Tree

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

		processTree: rbt.NewWith(utils.Int64Comparator),

		processQ: processQ,
	}

	return sch, nil
}

func (sc *scheduler) AddProcess(ctx context.Context, pi *types.Process) {
	ps := pi.ToSnapshot()
	sc.saveProcessSnapshot(ctx, ps)
	if sc.isLeader() {
		sc.processQ.Push(NewProcess(ps))
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
	go sc.watchProcess(ctx)
	return nil
}

func (sc *scheduler) process(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interval := time.Millisecond * 50
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

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

		ticker.Reset(interval)

		sc.fetchPrepares(ctx)

		opts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
		}

		key := processSchPrefix
		w := sc.v3cli.Watch(ctx, key, opts...)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case <-sc.notifier.ChangeNotify():
				break LOOP

			case wch, ok := <-w:
				if !ok {
					break LOOP
				}

				if wch.Err() != nil {
					break LOOP
				}

				for _, ev := range wch.Events {
					switch {
					case ev.Type == mvccpb.PUT && ev.IsCreate():
						// this event from other olive-mon node
						ps, err := parsePSnapKV(ev.Kv)
						if err == nil {
							sc.pmu.RLock()
							_, found := sc.processTree.Get(ps.Id)
							sc.pmu.RUnlock()
							if !found {
								sc.processQ.Push(NewProcess(ps))
							}
						}
					case ev.Type == mvccpb.DELETE:
						ps, err := parsePSnapKV(ev.PrevKv)
						if err == nil {
							sc.pmu.Lock()
							sc.processTree.Remove(ps.Id)
							sc.pmu.Unlock()

							sc.processQ.Remove(ps.Id)
						}
					}
				}

			case <-ticker.C:

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
					ExecuteProcess: &types.ExecuteProcessMsg{
						Process: process.ID(),
					},
				}
				sc.dispatchEvent(ctx, runner.Id, event)

				ps := process.ProcessSnapshot
				ps.Status = types.ProcessStatus_Ready
				ps.ReadyAt = time.Now().UnixNano()
				sc.saveProcessSnapshot(ctx, ps)
			}
		}
	}
}

// watchProcess watches events of *types.Process
func (sc *scheduler) watchProcess(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interval := time.Second * 30
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if !sc.waitLeaderLoop(ctx) {
			select {
			case <-ctx.Done():
				return
			default:
			}
			continue
		}

		ticker.Reset(interval)

		opts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
		}

		key := api.ProcessPrefix
		w := sc.v3cli.Watch(ctx, key, opts...)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case <-sc.notifier.ChangeNotify():
				break LOOP

			case wch, ok := <-w:
				if !ok {
					break LOOP
				}

				if wch.Err() != nil {
					break LOOP
				}

				for _, ev := range wch.Events {
					switch {
					case ev.Type == mvccpb.PUT && ev.IsModify():
						pi, err := parseProcessKV(ev.Kv)
						if err == nil {
							if pi.Executed() {
								// reallocates Process to another olive-runner
								sc.pmu.Lock()
								sc.processTree.Remove(pi.Id)
								sc.pmu.Unlock()
							}
							if pi.Finished() {
								sc.removeProcessSnapshot(ctx, pi.ToSnapshot())
							}
						}
					case ev.Type == mvccpb.DELETE:
						pi, err := parseProcessKV(ev.PrevKv)
						if err == nil {
							sc.pmu.Lock()
							sc.processTree.Remove(pi.Id)
							sc.pmu.Unlock()
						}
					}
				}

			case <-ticker.C:

				sc.pmu.RLock()
				values := sc.processTree.Values()
				sc.pmu.RUnlock()

				for _, ev := range values {
					ps, ok := ev.(*types.ProcessSnapshot)
					if !ok {
						continue
					}

					if ps.ExecuteExpired(processInterval) {
						sc.lg.Info("discovery an expired process, dispatches another runner",
							zap.Int64("process", ps.Id))
						newSnap := &types.ProcessSnapshot{
							Id:       ps.Id,
							Priority: ps.Priority,
							Status:   types.ProcessStatus_Prepare,
						}

						sc.pmu.Lock()
						sc.processTree.Remove(ps.Id)
						sc.pmu.Unlock()

						sc.processQ.Push(NewProcess(newSnap))
					}
				}
			}
		}
	}
}

func (sc *scheduler) saveProcessSnapshot(ctx context.Context, ps *types.ProcessSnapshot) {
	// saves scheduled ProcessSnapshot
	sc.pmu.Lock()
	sc.processTree.Put(ps.Id, ps)
	sc.pmu.Unlock()

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
	ts := time.Now().UnixNano()
	key := path.Join(api.RunnerTopic, fmt.Sprintf("%d", runner), fmt.Sprintf("%d", ts))
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
