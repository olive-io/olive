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
)

type ScoreFn[T any] func(v T) int64

type item[T any] struct {
	value T
	fn    ScoreFn[T]
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
	value := x.(*item[T])
	value.index = n
	pq.list = append(pq.list, value)
}

func (pq *priorityQueue[T]) Pop() any {
	old := *pq
	n := len(old.list)
	value := old.list[n-1]
	old.list[n-1] = nil
	value.index = -1
	pq.list = old.list[0 : n-1]
	return value
}

func (pq *priorityQueue[T]) update(item *item[T], value T, fn ScoreFn[T]) {
	item.value = value
	if fn != nil {
		item.fn = fn
	}
	heap.Fix(pq, item.index)
}
