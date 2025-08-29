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

type ClientFactory struct {
	rmu     sync.RWMutex
	runners map[uint64]*types.Runner

	smu   sync.RWMutex
	stats map[uint64]*types.RunnerStat

	pmu   sync.RWMutex
	pipes map[string]*StreamPipe
}

func NewFactory() *ClientFactory {
	rm := ClientFactory{
		runners: make(map[uint64]*types.Runner),
		stats:   make(map[uint64]*types.RunnerStat),
		pipes:   make(map[string]*StreamPipe),
	}
	return &rm
}

func (cf *ClientFactory) SetRunner(runner *types.Runner) {
	cf.rmu.Lock()
	defer cf.rmu.Unlock()
	cf.runners[runner.Id] = runner
}

func (cf *ClientFactory) SetStat(stat *types.RunnerStat) {
	cf.smu.Lock()
	defer cf.smu.Unlock()
	cf.stats[stat.Id] = stat
}

func (cf *ClientFactory) GetRunner(id uint64) (*types.Runner, bool) {
	cf.rmu.RLock()
	defer cf.rmu.RUnlock()

	runner, ok := cf.runners[id]
	return runner, ok
}

func (cf *ClientFactory) AddPipe(uid string, pipe *StreamPipe) {
	cf.pmu.Lock()
	defer cf.pmu.Unlock()

	cf.pipes[uid] = pipe
}

func (cf *ClientFactory) GetPipe(uid string) (*StreamPipe, bool) {
	cf.pmu.RLock()
	defer cf.pmu.RUnlock()

	pipe, ok := cf.pipes[uid]
	return pipe, ok
}

func (cf *ClientFactory) RemovePipe(uid string) {
	cf.pmu.Lock()
	defer cf.pmu.Unlock()
	delete(cf.pipes, uid)
}
