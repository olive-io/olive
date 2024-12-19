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
	"testing"

	"github.com/stretchr/testify/assert"

	pb "github.com/olive-io/olive/api/rpc/planepb"
)

func TestNewRunnerQueue(t *testing.T) {
	q := New[*pb.RunnerStatistics](func(v *pb.RunnerStatistics) int64 {
		return int64((100-v.CpuUsed)/100*4*2500 + (100-v.MemoryUsed)/100*16)
	})

	r1 := &pb.RunnerStatistics{
		Id:         "runner1",
		CpuUsed:    30,
		MemoryUsed: 50,
	}
	r2 := &pb.RunnerStatistics{
		Id:         "runner2",
		CpuUsed:    30,
		MemoryUsed: 60,
	}
	r3 := &pb.RunnerStatistics{
		Id:         "runner3",
		CpuUsed:    30,
		MemoryUsed: 70,
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &pb.RunnerStatistics{
		Id:         "runner4",
		CpuUsed:    20,
		MemoryUsed: 50,
	}
	q.Set(r4)

	if !assert.Equal(t, q.Len(), 4) {
		return
	}

	runners := make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*pb.RunnerStatistics).Id)
		}
	}

	assert.Equal(t, []string{"runner4", "runner1", "runner2", "runner3"}, runners)

	q.Push(r1)
	q.Push(r2)
	q.Push(r3)
	q.Push(r4)
	q.Set(&pb.RunnerStatistics{
		Id:         "runner4",
		CpuUsed:    30,
		MemoryUsed: 80,
	})

	runners = make([]string, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*pb.RunnerStatistics).Id)
		}
	}

	assert.Equal(t, []string{"runner1", "runner2", "runner3", "runner4"}, runners)
}

func TestNewRunnerQueue_Get_And_Remove(t *testing.T) {
	q := New[*pb.RunnerStatistics](func(v *pb.RunnerStatistics) int64 {
		return int64((100-v.CpuUsed)/100*4*2500 + (100-v.MemoryUsed)/100*16)
	})

	r1 := &pb.RunnerStatistics{
		Id:         "runner1",
		CpuUsed:    30,
		MemoryUsed: 50,
	}
	r2 := &pb.RunnerStatistics{
		Id:         "runner2",
		CpuUsed:    30,
		MemoryUsed: 60,
	}
	r3 := &pb.RunnerStatistics{
		Id:         "runner3",
		CpuUsed:    30,
		MemoryUsed: 70,
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &pb.RunnerStatistics{
		Id:         "runner4",
		CpuUsed:    20,
		MemoryUsed: 50,
	}
	q.Set(r4)

	v, ok := q.Get("runner2")
	if !assert.True(t, ok) {
		return
	}

	if !assert.Equal(t, r2, v) {
		return
	}

	v, ok = q.Remove("runner2")
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
			runners = append(runners, v.(*pb.RunnerStatistics).Id)
		}
	}

	assert.Equal(t, []string{"runner4", "runner1", "runner3"}, runners)
}
