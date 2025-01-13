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
	Update(runner *corev1.Runner)
	Remove(runner *corev1.Runner)
	Len() int

	// Pop gets the highest score Runner
	Pop(opts ...NextOption) (*Snapshot, bool)
	// Recycle recycles popped runners by Pop()
	Recycle()
	// Free clears popped runners by Pop()
	Free()
}

type schedulingQueue struct {
	mu       sync.RWMutex
	maps     map[string]*Snapshot
	activeQ  *queue.PriorityQueue[*Snapshot]
	recycled []*Snapshot
}

func NewSchedulingQueue() SchedulingQueue {

	fn := func(v *Snapshot) int64 {
		return v.Score()
	}

	return &schedulingQueue{
		maps:     make(map[string]*Snapshot),
		activeQ:  queue.New[*Snapshot](fn),
		recycled: make([]*Snapshot, 0),
	}
}

func (sq *schedulingQueue) Add(runner *corev1.Runner) {
	name := runner.Name
	snapshot := NewSnapshot(runner)
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.maps[name] = snapshot
	sq.activeQ.Push(snapshot)
}

func (sq *schedulingQueue) Update(runner *corev1.Runner) {
	name := runner.Name
	snapshot := NewSnapshot(runner)
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.maps[name] = snapshot
	sq.activeQ.Set(snapshot)
}

func (sq *schedulingQueue) Remove(runner *corev1.Runner) {
	name := runner.Name
	sq.mu.Lock()
	defer sq.mu.Unlock()
	delete(sq.maps, name)
	sq.activeQ.Remove(name)
}

func (sq *schedulingQueue) Len() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.maps)
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

	if options.Name != nil && *options.Name != "" {
		sq.mu.RLock()
		runner, ok := sq.maps[*options.Name]
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

// Recycle recycles popped runners by Pop()
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

// Free clears popped runners by Pop()
//
// Examples:
//
//	queue := NewSchedulingQueue(32)
//	s1, _ := queue.Pop()
//	s2, _ := queue.Pop()
//	queue.free()
func (sq *schedulingQueue) Free() {
	sq.mu.Lock()
	defer sq.mu.Unlock()
	sq.recycled = make([]*Snapshot, 0)
}
