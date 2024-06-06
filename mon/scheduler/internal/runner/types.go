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

package runner

import (
	"math"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type Snapshot struct {
	sync.RWMutex
	runner *corev1.Runner
	// Sets the limit of region number in which Runner
	regionLimit int
	// the score of Runner, calculation by calScore()
	score int64
}

func (s *Snapshot) UID() string {
	s.RLock()
	defer s.RUnlock()
	return string(s.runner.UID)
}

func (s *Snapshot) Score() int64 {
	s.RLock()
	defer s.RUnlock()
	return s.score
}

func (s *Snapshot) Get() *corev1.Runner {
	s.RLock()
	defer s.RUnlock()
	return s.runner.DeepCopy()
}

func (s *Snapshot) Set(r *corev1.Runner) {
	s.Lock()
	defer s.Unlock()
	s.runner = r
}

func (s *Snapshot) DeepCopy() *Snapshot {
	s.RLock()
	defer s.RUnlock()
	return &Snapshot{
		runner:      s.runner.DeepCopy(),
		regionLimit: s.regionLimit,
	}
}

func NewSnapshot(runner *corev1.Runner, limit int) *Snapshot {
	snapshot := &Snapshot{
		runner:      runner,
		regionLimit: limit,
		score:       0,
	}
	snapshot.score = snapshot.calScore()
	return snapshot
}

func (s *Snapshot) calScore() int64 {
	s.RLock()
	defer s.RUnlock()
	runner := s.runner
	limit := s.regionLimit
	status := runner.Status
	stat := status.Stat
	cpus := status.CpuTotal
	memoryTotal := status.MemoryTotal

	if len(status.Regions) >= s.regionLimit {
		return -math.MaxInt64
	}

	score := int64(int(cpus-stat.CpuUsed)%30) + int64(int(memoryTotal-stat.MemoryUsed)/1024/1024%30) +
		int64((limit-len(status.Regions))%30) +
		int64((limit-len(status.Leaders))%10)
	return score
}

type runnerSelector func(*corev1.Runner) bool

type Selector []runnerSelector

func NewSelector(options NextOptions) Selector {
	selector := Selector{}
	if len(options.Ignores) > 0 {
		selector = append(selector, func(runner *corev1.Runner) bool {
			idSets := sets.NewString(options.Ignores...)
			return idSets.Has(string(runner.UID))
		})
	}

	return selector
}

func (s Selector) Select(runner *corev1.Runner) bool {
	for _, item := range s {
		if !item(runner) {
			return false
		}
	}
	return true
}

type NextOptions struct {
	// runner uid
	UID *string
	// ignores the given runners id
	Ignores []string
}

type NextOption func(*NextOptions)

func WithUID(uid string) NextOption {
	return func(o *NextOptions) {
		o.UID = &uid
	}
}

func WithIgnores(ids ...string) NextOption {
	return func(o *NextOptions) {
		o.Ignores = ids
	}
}
