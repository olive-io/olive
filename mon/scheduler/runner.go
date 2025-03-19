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
	"math"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
)

type snapshot struct {
	*types.Runner
	stat *types.RunnerStat
}

func newSnapshot(runner *types.Runner, stat *types.RunnerStat) *snapshot {
	snap := &snapshot{
		Runner: runner,
		stat:   stat,
	}
	return snap
}

func (s *snapshot) ID() uint64 {
	return s.Id
}

func (s *snapshot) UpdateStat(stat *types.RunnerStat) {
	s.stat = stat
}

func (s *snapshot) Active() bool {
	if s.stat == nil && s.stat.Timestamp == 0 {
		return false
	}
	heartbeat := time.Millisecond * time.Duration(s.HeartbeatMs+30)
	expired := s.stat.Timestamp + heartbeat.Nanoseconds()
	return expired > time.Now().UnixNano()
}

func (s *snapshot) DeepCopy() *snapshot {
	out := &snapshot{
		Runner: proto.Clone(s.Runner).(*types.Runner),
		stat:   proto.Clone(s.stat).(*types.RunnerStat),
	}
	return out
}

func defaultSnapshotScore(snap *snapshot) int64 {
	stat := snap.stat
	if stat == nil || snap.stat.Timestamp == 0 {
		return math.MinInt64
	}

	score := int64((float64(snap.Cpu)-stat.CpuUsed)*10) +
		int64((float64(snap.Memory)-stat.MemoryUsed)*5) -
		int64(stat.BpmnProcesses+stat.BpmnTasks)

	return score
}

// addSnapshot updates *types.Runner and *types.RunnerStat for priority queue
func (sc *scheduler) addSnapshot(snap *snapshot) {
	sc.smu.Lock()
	sc.snapshots[snap.Id] = snap
	sc.smu.Unlock()
}

// removeSnapshot remove *types.Runner and *types.RunnerStat from priority queue
func (sc *scheduler) removeSnapshot(id uint64) {
	sc.smu.Lock()
	delete(sc.snapshots, id)
	sc.smu.Unlock()
}

func (sc *scheduler) updateSnapshot(runner *types.Runner, stat *types.RunnerStat) {
	sc.smu.Lock()
	defer sc.smu.Unlock()

	id := runner.Id
	if id == 0 {
		id = stat.Id
	}

	snap, ok := sc.snapshots[id]
	if !ok {
		return
	}

	if runner.Id != 0 {
		snap.Runner = runner
	}
	if stat.Id != 0 {
		snap.stat = stat
	}
}

func (sc *scheduler) watchRunners(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sc.loadAllRunners(ctx)

	for {

		opts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
		}

		w := sc.v3cli.Watch(ctx, api.RunnerPrefix, opts...)
		rw := sc.v3cli.Watch(ctx, api.RunnerStatPrefix, opts...)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case ch, ok := <-w:
				if !ok {
					break LOOP
				}

				if err := ch.Err(); err != nil {
					break LOOP
				}

				for _, ev := range ch.Events {
					switch ev.Type {
					case clientv3.EventTypePut:
						runner, err := types.RunnerFromKV(ev.Kv)
						if err == nil {
							sc.addSnapshot(newSnapshot(runner, new(types.RunnerStat)))
						}

					case clientv3.EventTypeDelete:
						runner, err := types.RunnerFromKV(ev.PrevKv)
						if err == nil {
							sc.removeSnapshot(runner.Id)
						}
					}
				}

			case ch, ok := <-rw:
				if !ok {
					break LOOP
				}

				if err := ch.Err(); err != nil {
					break LOOP
				}

				for _, ev := range ch.Events {
					switch ev.Type {
					case clientv3.EventTypePut:
						stat, err := types.StatFromKV(ev.Kv)
						if err == nil {
							if stat.Timestamp == 0 {
								stat.Timestamp = time.Now().UnixNano()
							}
							sc.updateSnapshot(new(types.Runner), stat)
						}
					}
				}
			}
		}
	}
}

func (sc *scheduler) loadAllRunners(ctx context.Context) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := sc.v3cli.Get(ctx, api.RunnerPrefix, opts...)
	if err != nil {
		return
	}

	for _, kv := range resp.Kvs {
		var runner types.Runner
		if err = proto.Unmarshal(kv.Value, &runner); err != nil {
			continue
		}

		snap := newSnapshot(&runner, new(types.RunnerStat))
		sc.addSnapshot(snap)
	}
}
