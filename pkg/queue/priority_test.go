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
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/api/types"
)

type runnerStat struct {
	*types.RunnerStat
}

func (rs *runnerStat) ID() int64 {
	id := crc32.ChecksumIEEE([]byte(rs.Name))
	return int64(id)
}

func TestNewRunnerQueue(t *testing.T) {
	q := New[*runnerStat](func(v *runnerStat) int64 {
		return int64((100-v.CpuUsed)/100*4*2500 + (100-v.MemoryUsed)/100*16)
	})

	r1 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner1",
			CpuUsed:    30,
			MemoryUsed: 50,
		},
	}
	r2 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner2",
			CpuUsed:    30,
			MemoryUsed: 60,
		},
	}
	r3 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner3",
			CpuUsed:    30,
			MemoryUsed: 70,
		},
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner4",
			CpuUsed:    20,
			MemoryUsed: 50,
		},
	}
	q.Set(r4)

	if !assert.Equal(t, q.Len(), 4) {
		return
	}

	runners := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*runnerStat).Name)
		}
	}

	assert.Equal(t, []string{"runner4", "runner1", "runner2", "runner3"}, runners)

	q.Push(r1)
	q.Push(r2)
	q.Push(r3)
	q.Push(r4)
	q.Set(&runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner4",
			CpuUsed:    30,
			MemoryUsed: 80,
		},
	})

	runners = make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*runnerStat).Name)
		}
	}

	assert.Equal(t, []string{"runner1", "runner2", "runner3", "runner4"}, runners)
}

func TestNewRunnerQueue_Get_And_Remove(t *testing.T) {
	q := New[*runnerStat](func(v *runnerStat) int64 {
		return int64((100-v.CpuUsed)/100*4*2500 + (100-v.MemoryUsed)/100*16)
	})

	r1 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner1",
			CpuUsed:    30,
			MemoryUsed: 50,
		},
	}
	r2 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner2",
			CpuUsed:    30,
			MemoryUsed: 60,
		},
	}
	r3 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner3",
			CpuUsed:    30,
			MemoryUsed: 70,
		},
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &runnerStat{
		RunnerStat: &types.RunnerStat{
			Name:       "runner4",
			CpuUsed:    20,
			MemoryUsed: 50,
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

	runners := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*runnerStat).Name)
		}
	}

	assert.Equal(t, []string{"runner4", "runner1", "runner3"}, runners)
}
