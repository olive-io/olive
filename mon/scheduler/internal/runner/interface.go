/*
Copyright 2024 The olive Authors

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

package runner

import (
	"sync"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/queue"
)

type SchedulingQueue interface {
	Add(runner *corev1.Runner)
	Update(runner *corev1.Runner, limit int)
	Remove(runner *corev1.Runner)

	// Pop gets the highest score Runner
	Pop(opts ...NextOption) (*Snapshot, bool)
}

type schedulingQueue struct {
	mu          sync.RWMutex
	maps        map[string]*Snapshot
	activeQ     *queue.PriorityQueue
	inactive    []*corev1.Runner
	recycled    []*Snapshot
	regionLimit int
}

func NewSchedulingQueue(regionLimit int) SchedulingQueue {
	return &schedulingQueue{
		maps:        make(map[string]*Snapshot),
		activeQ:     queue.New(),
		inactive:    make([]*corev1.Runner, 0),
		recycled:    make([]*Snapshot, 0),
		regionLimit: regionLimit,
	}
}

func (sq *schedulingQueue) Add(runner *corev1.Runner) {
	id := string(runner.UID)
	snapshot := NewSnapshot(runner, sq.regionLimit)
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.maps[id] = snapshot
	sq.activeQ.Push(snapshot)
}

func (sq *schedulingQueue) Update(runner *corev1.Runner, limit int) {
	id := string(runner.UID)
	snapshot := NewSnapshot(runner, sq.regionLimit)
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.regionLimit = limit
	sq.maps[id] = snapshot
	sq.activeQ.Set(snapshot)
}

func (sq *schedulingQueue) Remove(runner *corev1.Runner) {
	id := string(runner.UID)
	sq.mu.Lock()
	defer sq.mu.Unlock()
	delete(sq.maps, id)
	sq.activeQ.Remove(id)
}

func (sq *schedulingQueue) Pop(opts ...NextOption) (*Snapshot, bool) {
	var options NextOptions
	for _, opt := range opts {
		opt(&options)
	}

	sq.mu.RLock()
	length := len(sq.maps)
	sq.mu.RUnlock()
	if length == 0 {
		return nil, false
	}

	if options.UID != nil && *options.UID != "" {
		sq.mu.RLock()
		runner, ok := sq.maps[*options.UID]
		sq.mu.RUnlock()
		return runner, ok
	}

	selector := NewSelector(options)

	sq.mu.Lock()
	defer sq.mu.Unlock()
	recycles := make([]*Snapshot, 0)
	var matched *Snapshot
	for {
		item, ok := sq.activeQ.Pop()
		if !ok {
			break
		}
		snap := item.(*Snapshot)
		if selector.Select(snap.runner) {
			matched = snap
			sq.recycled = append(sq.recycled, snap)
			break
		}

		recycles = append(recycles, snap)
	}

	for _, recycle := range recycles {
		sq.activeQ.Push(recycle)
	}

	return matched, matched != nil
}

// Recycle reset Runner Snapshot to priority queue and clear recycled
//
// Examples:
//
//	queue := NewSchedulingQueue(32)
//	s1, _ := queue.Pop()
//	s2, _ := queue.Pop()
//	queue.Recycle()
func (sq *schedulingQueue) Recycle() {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	for _, recycle := range sq.recycled {
		sq.activeQ.Push(recycle)
	}
	sq.recycled = make([]*Snapshot, 0)
}
