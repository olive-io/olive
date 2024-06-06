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

package queue

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type RunnerInfo struct {
	Id int64
	r  *corev1.RunnerStat
}

func (ri *RunnerInfo) UID() string {
	return fmt.Sprintf("%d", ri.Id)
}

func (ri *RunnerInfo) Score() int64 {
	v := ri.r
	return int64(100 - v.CpuUsed + 100 - v.MemoryUsed)
}

func TestNewRunnerQueue(t *testing.T) {
	q := New()

	r1 := &RunnerInfo{Id: 1, r: &corev1.RunnerStat{
		CpuUsed:    10,
		MemoryUsed: 20,
	}}
	r2 := &RunnerInfo{Id: 2, r: &corev1.RunnerStat{
		CpuUsed:    40,
		MemoryUsed: 20,
	}}
	r3 := &RunnerInfo{Id: 3, r: &corev1.RunnerStat{
		CpuUsed:    20,
		MemoryUsed: 30,
	}}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &RunnerInfo{Id: 4, r: &corev1.RunnerStat{
		CpuUsed:    60,
		MemoryUsed: 60,
	}}
	q.Set(r4)

	if !assert.Equal(t, q.Len(), 4) {
		return
	}

	value, _ := q.Next("2")
	assert.Equal(t, "3", value.UID())

	runners := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*RunnerInfo).UID())
		}
	}

	assert.Equal(t, []string{"4", "2", "3", "1"}, runners)

	q.Push(r1)
	q.Push(r2)
	q.Push(r3)
	q.Push(r4)
	q.Set(&RunnerInfo{Id: 4, r: &corev1.RunnerStat{
		CpuUsed:    5,
		MemoryUsed: 5,
	}})

	runners = make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*RunnerInfo).UID())
		}
	}

	assert.Equal(t, []string{"2", "3", "1", "4"}, runners)
}
