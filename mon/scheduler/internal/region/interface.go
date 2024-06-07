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

package region

import (
	"sync"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/queue"
)

type SchedulingQueue interface {
	Add(region *corev1.Region)
	Update(region *corev1.Region, limit int)
	Remove(region *corev1.Region)
	Len() int

	// Pop gets the highest score Region
	Pop(opts ...NextOption) (*Snapshot, bool)
	// Recycle recycles the popped region by Pop()
	Recycle()
	// Free clears the popped region by Pop()
	Free()

	// RegionToScale gets a scale Region
	RegionToScale() (*corev1.Region, bool)
}

type schedulingQueue struct {
	mu       sync.RWMutex
	maps     map[string]*Snapshot
	activeQ  *queue.PriorityQueue
	recycled []*Snapshot

	smu    sync.Mutex
	scaleQ *queue.PriorityQueue

	definitionLimit int
	replicaNum      int
}

func NewSchedulingQueue(definitionLimit, replicaNum int) SchedulingQueue {
	return &schedulingQueue{
		maps:            make(map[string]*Snapshot),
		activeQ:         queue.New(),
		recycled:        make([]*Snapshot, 0),
		scaleQ:          queue.New(),
		definitionLimit: definitionLimit,
		replicaNum:      replicaNum,
	}
}

func (sq *schedulingQueue) Add(region *corev1.Region) {
	name := region.Name
	snapshot := NewSnapshot(region, sq.definitionLimit)
	sq.mu.Lock()
	sq.maps[name] = snapshot
	sq.activeQ.Push(snapshot)
	sq.mu.Unlock()

	sq.scaledRegion(region)
}

func (sq *schedulingQueue) Update(region *corev1.Region, limit int) {
	name := region.Name
	snapshot := NewSnapshot(region, sq.definitionLimit)
	sq.mu.Lock()
	sq.definitionLimit = limit
	sq.maps[name] = snapshot
	sq.activeQ.Set(snapshot)
	sq.mu.Unlock()

	sq.scaledRegion(region)
}

func (sq *schedulingQueue) Remove(region *corev1.Region) {
	name := region.Name
	sq.mu.Lock()
	delete(sq.maps, name)
	sq.activeQ.Remove(name)
	sq.mu.Unlock()

	sq.smu.Lock()
	sq.scaleQ.Remove(name)
	sq.smu.Unlock()
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
		region, ok := sq.maps[*options.Name]
		sq.mu.RUnlock()
		return region, ok
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
		if selector.Select(snap.region) {
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

// Recycle recycles popped regions by Pop()
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

// Free clears popped regions by Pop()
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

func (sq *schedulingQueue) scaledRegion(region *corev1.Region) {
	if int(region.Status.Replicas) >= sq.replicaNum {
		return
	}
	sq.smu.Lock()
	sq.scaleQ.Set(newScaleRegion(region))
	sq.smu.Unlock()
}

func (sq *schedulingQueue) RegionToScale() (*corev1.Region, bool) {
	sq.smu.Lock()
	x, ok := sq.scaleQ.Pop()
	sq.smu.Unlock()
	if !ok {
		return nil, false
	}
	region := x.(*scaleRegion).region.DeepCopy()
	return region, true
}
