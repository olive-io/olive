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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/api/types"
)

type process struct {
	*types.Process
}

func (p *process) ID() int64 {
	return p.Id
}

func TestNewProcessQueue(t *testing.T) {
	q := New[*process](func(v *process) int64 {
		return int64(v.Priority)
	})

	p1 := &process{
		Process: &types.Process{
			Id:       1,
			Name:     "p1",
			Priority: 2,
		},
	}
	p2 := &process{
		Process: &types.Process{
			Id:       2,
			Name:     "p2",
			Priority: 4,
		},
	}
	p3 := &process{
		Process: &types.Process{
			Id:       3,
			Name:     "p3",
			Priority: 3,
		},
	}
	q.Set(p2)
	q.Set(p1)
	q.Set(p3)

	p4 := &process{
		Process: &types.Process{
			Id:       4,
			Name:     "p4",
			Priority: 100,
		},
	}
	q.Set(p4)

	if !assert.Equal(t, q.Len(), 4) {
		return
	}

	processes := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			processes = append(processes, v.(*process).Name)
		}
	}

	assert.Equal(t, []string{"p4", "p2", "p3", "p1"}, processes)

	q.Push(p1)
	q.Push(p2)
	q.Push(p3)
	q.Push(p4)
	q.Set(&process{
		Process: &types.Process{
			Id:       4,
			Name:     "p4",
			Priority: 1,
		},
	})

	processes = make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			processes = append(processes, v.(*process).Name)
		}
	}

	assert.Equal(t, []string{"p2", "p3", "p1", "p4"}, processes)
}

func TestNewProcessQueue_Get_And_Remove(t *testing.T) {
	q := New[*process](func(v *process) int64 {
		return v.Priority
	})

	r1 := &process{
		Process: &types.Process{
			Id:       1,
			Name:     "p1",
			Priority: 2,
		},
	}
	r2 := &process{
		Process: &types.Process{
			Id:       2,
			Name:     "p2",
			Priority: 1,
		},
	}
	r3 := &process{
		Process: &types.Process{
			Id:       3,
			Name:     "p3",
			Priority: 1,
		},
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &process{
		Process: &types.Process{
			Id:       4,
			Name:     "p4",
			Priority: 10,
		},
	}
	q.Set(r4)

	v, ok := q.Get(r2.ID())
	if !assert.True(t, ok) {
		return
	}

	if !assert.Equal(t, r2, v) {
		return
	}

	v, ok = q.Remove(r2.ID())
	if !assert.True(t, ok) {
		return
	}

	if !assert.Equal(t, r2, v) {
		return
	}

	if !assert.Equal(t, q.Len(), 3) {
		return
	}

	processes := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			processes = append(processes, v.(*process).Name)
		}
	}

	assert.Equal(t, []string{"p4", "p1", "p3"}, processes)
}
