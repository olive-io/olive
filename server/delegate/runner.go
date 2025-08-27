/*
Copyright 2025 The olive Authors

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

package delegate

import (
	"sync"

	"github.com/olive-io/olive/api/types"
)

type RunnerMap struct {
	rmu     sync.RWMutex
	runners map[uint64]*types.Runner

	smu   sync.RWMutex
	stats map[uint64]*types.RunnerStat
}

func NewRunnerMap() *RunnerMap {
	rm := RunnerMap{
		runners: make(map[uint64]*types.Runner),
		stats:   make(map[uint64]*types.RunnerStat),
	}
	return &rm
}

func (rm *RunnerMap) SetRunner(runner *types.Runner) {
	rm.rmu.Lock()
	defer rm.rmu.Unlock()
	rm.runners[runner.Id] = runner
}

func (rm *RunnerMap) SetStat(stat *types.RunnerStat) {
	rm.smu.Lock()
	defer rm.smu.Unlock()
	rm.stats[stat.Id] = stat
}

func (rm *RunnerMap) GetRunner(id uint64) (*types.Runner, bool) {
	rm.rmu.RLock()
	defer rm.rmu.RUnlock()

	runner, ok := rm.runners[id]
	return runner, ok
}
