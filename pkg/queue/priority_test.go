// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

func (ri *RunnerInfo) Score() float64 {
	v := ri.r
	return 100 - v.CpuUsed + 100 - v.MemoryUsed
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
