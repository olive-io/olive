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

type INode interface {
	ID() uint64
}

type PriorityQueue[T INode] struct {
	pq      priorityQueue[T]
	m       map[uint64]*item[T]
	scoreFn ScoreFn[T]
}

func New[T INode](fn ScoreFn[T]) *PriorityQueue[T] {
	queue := &PriorityQueue[T]{
		pq:      priorityQueue[T]{},
		m:       map[uint64]*item[T]{},
		scoreFn: fn,
	}

	return queue
}

func (q *PriorityQueue[T]) Push(val T) {
	it := &item[T]{
		value: val,
		fn:    q.scoreFn,
	}
	heap.Push(&q.pq, it)
	q.m[val.ID()] = it
}

func (q *PriorityQueue[T]) Pop() (any, bool) {
	if q.pq.Len() == 0 {
		return nil, false
	}
	x := heap.Pop(&q.pq)
	it := x.(*item[T])
	delete(q.m, it.value.ID())
	return it.value, true
}

func (q *PriorityQueue[T]) Get(id uint64) (any, bool) {
	v, ok := q.m[id]
	if !ok {
		return nil, false
	}
	return v.value, true
}

func (q *PriorityQueue[T]) Remove(id uint64) (any, bool) {
	v, ok := q.m[id]
	if !ok {
		return nil, false
	}
	idx := v.index
	x := heap.Remove(&q.pq, idx)
	v = x.(*item[T])
	delete(q.m, id)
	return v.value, true
}

func (q *PriorityQueue[T]) Set(val T) {
	v, ok := q.m[val.ID()]
	if !ok {
		q.Push(val)
		return
	}
	v.value = val
	q.pq.update(v, val, q.scoreFn)
	q.m[val.ID()] = v
}

func (q *PriorityQueue[T]) Len() int {
	return q.pq.Len()
}

type SyncPriorityQueue[T INode] struct {
	mu sync.RWMutex
	pq *PriorityQueue[T]
}

func NewSync[T INode](fn ScoreFn[T]) *SyncPriorityQueue[T] {
	pq := New(fn)
	queue := &SyncPriorityQueue[T]{pq: pq}

	return queue
}

func (q *SyncPriorityQueue[T]) Push(val T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pq.Push(val)
}

func (q *SyncPriorityQueue[T]) Pop() (any, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	result, ok := q.pq.Pop()
	return result, ok
}

func (q *SyncPriorityQueue[T]) Get(id uint64) (any, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	result, ok := q.pq.Get(id)
	return result, ok
}

func (q *SyncPriorityQueue[T]) Remove(id uint64) (any, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	result, ok := q.pq.Remove(id)
	return result, ok
}

func (q *SyncPriorityQueue[T]) Set(val T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pq.Set(val)
}

func (q *SyncPriorityQueue[T]) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.pq.Len()
}
