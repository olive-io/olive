// Copyright 2023 The olive Authors
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
