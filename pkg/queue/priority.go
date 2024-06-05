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

package queue

import (
	"container/heap"
	"sync"
)

type PriorityQueue struct {
	pq priorityQueue
	m  map[string]*item
}

func New() *PriorityQueue {
	queue := &PriorityQueue{
		pq: priorityQueue{},
		m:  map[string]*item{},
	}

	return queue
}

func (q *PriorityQueue) Push(val IScoreGetter) {
	it := &item{
		value: val,
	}
	heap.Push(&q.pq, it)
	q.m[val.UID()] = it
}

func (q *PriorityQueue) Pop() (IScoreGetter, bool) {
	if q.pq.Len() == 0 {
		return nil, false
	}
	x := heap.Pop(&q.pq)
	it := x.(*item)
	delete(q.m, it.value.UID())
	return it.value, true
}

func (q *PriorityQueue) Get(id string) (IScoreGetter, bool) {
	v, ok := q.m[id]
	if !ok {
		return nil, false
	}
	return v.value, true
}

func (q *PriorityQueue) Next(id string) (IScoreGetter, bool) {
	v, ok := q.m[id]
	if !ok {
		return nil, false
	}
	idx := v.index
	if idx >= q.pq.Len()-1 {
		return nil, false
	}
	return q.pq.list[idx+1].value, true
}

func (q *PriorityQueue) Remove(id string) (IScoreGetter, bool) {
	v, ok := q.m[id]
	if !ok {
		return nil, false
	}
	idx := v.index
	x := heap.Remove(&q.pq, idx)
	v = x.(*item)
	delete(q.m, id)
	return v.value, true
}

func (q *PriorityQueue) Set(val IScoreGetter) {
	v, ok := q.m[val.UID()]
	if !ok {
		q.Push(val)
		return
	}
	v.value = val
	q.pq.update(v, val)
	q.m[val.UID()] = v
}

func (q *PriorityQueue) Len() int {
	return q.pq.Len()
}

type SyncPriorityQueue struct {
	mu sync.RWMutex
	pq *PriorityQueue
}

func NewSync() *SyncPriorityQueue {
	pq := New()
	queue := &SyncPriorityQueue{pq: pq}

	return queue
}

func (q *SyncPriorityQueue) Push(val IScoreGetter) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pq.Push(val)
}

func (q *SyncPriorityQueue) Pop() (IScoreGetter, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	result, ok := q.pq.Pop()
	return result, ok
}

func (q *SyncPriorityQueue) Get(id string) (IScoreGetter, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result, ok := q.pq.Get(id)
	return result, ok
}

func (q *SyncPriorityQueue) Next(id string) (IScoreGetter, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	result, ok := q.pq.Next(id)
	return result, ok
}

func (q *SyncPriorityQueue) Remove(id string) (IScoreGetter, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	result, ok := q.pq.Remove(id)
	return result, ok
}

func (q *SyncPriorityQueue) Set(val IScoreGetter) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pq.Set(val)
}

func (q *SyncPriorityQueue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.pq.Len()
}
