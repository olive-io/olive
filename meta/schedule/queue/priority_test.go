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

	if q.Len() != 3 {
		if !assert.Equal(t, q.Len(), 4) {
			return
		}
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
