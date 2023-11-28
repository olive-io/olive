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

type ChaosFn[T any] func(v T) int

type item[T any] struct {
	value T
	fn    ChaosFn[T]
	index int
}

type priorityQueue[T any] struct {
	list []*item[T]
}

func (pq *priorityQueue[T]) Len() int { return len(pq.list) }

func (pq *priorityQueue[T]) Less(i, j int) bool {
	x, y := pq.list[i], pq.list[j]
	return x.fn(x.value) > y.fn(y.value)
}

func (pq *priorityQueue[T]) Swap(i, j int) {
	pq.list[i], pq.list[j] = pq.list[j], pq.list[i]
	pq.list[i].index = i
	pq.list[j].index = j
}

func (pq *priorityQueue[T]) Push(x any) {
	n := len(pq.list)
	item := x.(*item[T])
	item.index = n
	pq.list = append(pq.list, item)
}

func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old.list)
	item := old.list[n-1]
	old.list[n-1] = nil
	item.index = -1
	pq.list = old.list[0 : n-1]
	return item
}

func (pq *priorityQueue[T]) update(item *item[T], value T, fn ChaosFn[T]) {
	item.value = value
	if fn != nil {
		item.fn = fn
	}
	heap.Fix(pq, item.index)
}
