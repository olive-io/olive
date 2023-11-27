// Copyright 2023 Lack (xingyys@gmail.com).
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

package schedule

import (
	"container/heap"
)

type INode interface {
	ID() uint64
}

type PriorityRunnerQueue[T INode] struct {
	pq      priorityQueue[T]
	m       map[uint64]*item[T]
	gradeFn GradeFn[T]
}

func NewRunnerQueue[T INode](fn GradeFn[T]) *PriorityRunnerQueue[T] {
	queue := &PriorityRunnerQueue[T]{
		pq:      priorityQueue[T]{},
		m:       map[uint64]*item[T]{},
		gradeFn: fn,
	}

	return queue
}

func (q *PriorityRunnerQueue[T]) Push(val T) {
	it := &item[T]{
		value: val,
		fn:    q.gradeFn,
	}
	heap.Push(&q.pq, it)
	q.m[val.ID()] = it
}

func (q *PriorityRunnerQueue[T]) Pop() (any, bool) {
	if q.pq.Len() == 0 {
		return nil, false
	}
	x := heap.Pop(&q.pq)
	it := x.(*item[T])
	delete(q.m, it.value.ID())
	return it.value, true
}

func (q *PriorityRunnerQueue[T]) Set(val T) {
	v, ok := q.m[val.ID()]
	if !ok {
		q.Push(val)
		return
	}
	v.value = val
	q.pq.update(v, val, q.gradeFn)
	q.m[val.ID()] = v
}

func (q *PriorityRunnerQueue[T]) Len() int {
	return q.pq.Len()
}
