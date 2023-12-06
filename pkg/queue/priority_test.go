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

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/stretchr/testify/assert"
)

func TestNewRunnerQueue(t *testing.T) {
	q := New[*pb.RunnerStat](func(v *pb.RunnerStat) int64 {
		return int64(((100-v.CpuPer)/100*4*2500 + (100-v.MemoryPer)/100*16) + float64(len(v.Regions))*-1)
	})

	r1 := &pb.RunnerStat{
		Id:        1,
		CpuPer:    30,
		MemoryPer: 50,
		Regions:   []uint64{1},
		Leaders:   []string{"1.1"},
	}
	r2 := &pb.RunnerStat{
		Id:        2,
		CpuPer:    30,
		MemoryPer: 60,
		Regions:   []uint64{2},
		Leaders:   []string{"1.3"},
	}
	r3 := &pb.RunnerStat{
		Id:        3,
		CpuPer:    30,
		MemoryPer: 70,
		Regions:   []uint64{3},
		Leaders:   []string{"3.1"},
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &pb.RunnerStat{
		Id:        4,
		CpuPer:    20,
		MemoryPer: 50,
		Regions:   []uint64{3},
		Leaders:   []string{"3.1"},
	}
	q.Set(r4)

	if !assert.Equal(t, q.Len(), 4) {
		return
	}

	runners := make([]uint64, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*pb.RunnerStat).Id)
		}
	}

	assert.Equal(t, []uint64{4, 1, 2, 3}, runners)

	q.Push(r1)
	q.Push(r2)
	q.Push(r3)
	q.Push(r4)
	q.Set(&pb.RunnerStat{
		Id:        4,
		CpuPer:    30,
		MemoryPer: 80,
		Regions:   []uint64{3},
		Leaders:   []string{"3.1"},
	})

	runners = make([]uint64, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*pb.RunnerStat).Id)
		}
	}

	assert.Equal(t, []uint64{1, 2, 3, 4}, runners)
}

func TestNewRunnerQueue_Get_And_Remove(t *testing.T) {
	q := New[*pb.RunnerStat](func(v *pb.RunnerStat) int64 {
		return int64(((100-v.CpuPer)/100*4*2500 + (100-v.MemoryPer)/100*16) + float64(len(v.Regions))*-1)
	})

	r1 := &pb.RunnerStat{
		Id:        1,
		CpuPer:    30,
		MemoryPer: 50,
		Regions:   []uint64{1},
		Leaders:   []string{"1.1"},
	}
	r2 := &pb.RunnerStat{
		Id:        2,
		CpuPer:    30,
		MemoryPer: 60,
		Regions:   []uint64{2},
		Leaders:   []string{"1.3"},
	}
	r3 := &pb.RunnerStat{
		Id:        3,
		CpuPer:    30,
		MemoryPer: 70,
		Regions:   []uint64{3},
		Leaders:   []string{"3.1"},
	}
	q.Set(r2)
	q.Set(r1)
	q.Set(r3)

	r4 := &pb.RunnerStat{
		Id:        4,
		CpuPer:    20,
		MemoryPer: 50,
		Regions:   []uint64{3},
		Leaders:   []string{"3.1"},
	}
	q.Set(r4)

	v, ok := q.Get(2)
	if !assert.True(t, ok) {
		return
	}

	if !assert.Equal(t, r2, v) {
		return
	}

	v, ok = q.Remove(2)
	if !assert.True(t, ok) {
		return
	}

	if !assert.Equal(t, r2, v) {
		return
	}

	if !assert.Equal(t, q.Len(), 3) {
		return
	}

	runners := make([]uint64, 0)
	for q.Len() != 0 {
		if v, ok := q.Pop(); ok {
			runners = append(runners, v.(*pb.RunnerStat).Id)
		}
	}

	assert.Equal(t, []uint64{4, 1, 3}, runners)
}
