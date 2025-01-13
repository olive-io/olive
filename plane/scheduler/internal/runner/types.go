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
	"sync/atomic"

	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type Snapshot struct {
	runner *corev1.Runner
	// the score of Runner, calculation by calScore()
	score atomic.Int64
}

func (s *Snapshot) ID() string {
	return s.UID()
}

func (s *Snapshot) UID() string {
	return s.runner.Name
}

func (s *Snapshot) Score() int64 {
	return s.score.Load()
}

func (s *Snapshot) Get() *corev1.Runner {
	return s.runner.DeepCopy()
}

func (s *Snapshot) Set(r *corev1.Runner) {
	s.runner = r
}

func (s *Snapshot) DeepCopy() *Snapshot {
	return &Snapshot{
		runner: s.runner.DeepCopy(),
	}
}

func NewSnapshot(runner *corev1.Runner) *Snapshot {
	snapshot := &Snapshot{
		runner: runner,
		score:  atomic.Int64{},
	}
	snapshot.score.Store(snapshot.calScore())
	return snapshot
}

func (s *Snapshot) calScore() int64 {
	runner := s.runner
	status := runner.Status
	stat := status.Stat
	cpus := status.CpuTotal
	memoryTotal := status.MemoryTotal

	score := int64(int(cpus-stat.CpuUsed)%30) + int64(int(memoryTotal-stat.MemoryUsed)/1024/1024%30)
	return score
}

type runnerSelector func(*corev1.Runner) bool

type Selector []runnerSelector

func NewSelector(options NextOptions) Selector {
	selector := Selector{}
	selector = append(selector, func(r *corev1.Runner) bool {
		return r.Status.Phase == corev1.RunnerActive
	})
	if options.Name != nil && *options.Name != "" {
		sfn := func(r *corev1.Runner) bool { return *options.Name != r.Name }
		selector = append(selector, sfn)
	}
	if len(options.Ignores) > 0 {
		sfn := func(region *corev1.Runner) bool {
			idSets := sets.NewString(options.Ignores...)
			return !idSets.Has(region.Name)
		}
		selector = append(selector, sfn)
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
	// runner name
	Name *string
	// ignores the given runners name
	Ignores []string
}

type NextOption func(*NextOptions)

func WithName(name string) NextOption {
	return func(o *NextOptions) {
		o.Name = &name
	}
}

func WithIgnores(names ...string) NextOption {
	return func(o *NextOptions) {
		o.Ignores = names
	}
}
