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

package scheduler

import (
	"sync"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type RunnerInfo struct {
	Runner      *corev1.Runner
	RegionLimit int
}

func (ri *RunnerInfo) UID() string {
	return string(ri.Runner.UID)
}

func (ri *RunnerInfo) Score() int64 {
	status := ri.Runner.Status
	stat := status.Stat
	cpus := status.CpuTotal
	memoryTotal := status.MemoryTotal

	score := int64(int(cpus-stat.CpuUsed)%50) + int64(int(memoryTotal-stat.MemoryUsed)/1024/1024%50)

	return score
}

type RunnerFilter func(*corev1.Runner) bool

type RunnerMap struct {
	mu    sync.RWMutex
	store map[string]*corev1.Runner
}

func NewRunnerMap() *RunnerMap {
	return &RunnerMap{
		store: make(map[string]*corev1.Runner),
	}
}

func (m *RunnerMap) Get(uid string) (*corev1.Runner, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rt, ok := m.store[uid]
	return rt, ok
}

func (m *RunnerMap) Hit(filters ...RunnerFilter) []*corev1.Runner {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runners := make([]*corev1.Runner, 0)
	for _, runner := range m.store {
		match := true
		for _, fn := range filters {
			if !fn(runner) {
				match = false
				break
			}
		}
		if match {
			runners = append(runners, runner)
		}
	}

	return runners
}

func (m *RunnerMap) Put(rt *corev1.Runner) {
	uid := string(rt.UID)
	m.mu.Lock()
	m.store[uid] = rt
	m.mu.Unlock()
}

func (m *RunnerMap) Del(uid string) (*corev1.Runner, bool) {
	m.mu.Lock()
	rt, ok := m.store[uid]
	if !ok {
		m.mu.Unlock()
		return nil, false
	}
	m.store[uid] = nil
	delete(m.store, uid)
	m.mu.Unlock()
	return rt, true
}

func (m *RunnerMap) Len() int {
	m.mu.RLock()
	length := len(m.store)
	m.mu.RUnlock()
	return length
}
