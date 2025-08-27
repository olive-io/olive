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

package runner

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/version"
)

type Runner struct {
	cfg *Config

	tr *types.Runner

	smu  sync.RWMutex
	stat *types.RunnerStat
}

func New(cfg *Config) (*Runner, error) {

	cpuTotal := uint64(0)
	cpus, err := cpu.Counts(false)
	if err != nil {
		return nil, fmt.Errorf("read system cpu: %w", err)
	}
	cpuInfos, _ := cpu.Info()
	if len(cpuInfos) > 0 {
		cpuTotal = uint64(cpus) * uint64(cpuInfos[0].Mhz)
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("read system memory: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("read system hostname: %w", err)
	}

	tr := &types.Runner{
		Id:          0,
		Uid:         cfg.UID,
		Name:        cfg.Name,
		Version:     version.GitTag,
		HeartbeatMs: cfg.HeartbeatDuration.Milliseconds(),
		Hostname:    hostname,
		Metadata:    map[string]string{},
		Features:    map[string]string{},
		Cpu:         cpuTotal,
		Memory:      vm.Total,
	}

	runner := &Runner{
		cfg: cfg,
		tr:  tr,
	}

	return runner, nil
}

func (r *Runner) generateRunnerStat() *types.RunnerStat {
	rs := &types.RunnerStat{
		Id:            r.tr.Id,
		Uid:           r.tr.Uid,
		Timestamp:     time.Now().UnixNano(),
		Steps:         uint64(stepCounter.Get()),
		CommitCount:   uint64(stepCommitCounter.Get()),
		RollbackCount: uint64(stepRollbackCounter.Get()),
		DestroyCount:  uint64(stepDestroyCounter.Get()),
	}
	interval := time.Millisecond * 300
	percents, _ := cpu.Percent(interval, false)
	if len(percents) > 0 {
		rs.CpuUsed = percents[0] * float64(r.tr.Cpu) / 100
	}

	vm, err := mem.VirtualMemory()
	if err == nil {
		rs.MemoryUsed = vm.UsedPercent * float64(r.tr.Memory)
	}

	return rs
}

func (r *Runner) GetStat() *types.RunnerStat {
	r.smu.RLock()
	defer r.smu.RUnlock()
	out := new(types.RunnerStat)
	*out = *r.stat
	return out
}

func (r *Runner) process(stop <-chan struct{}) {
	internal := r.cfg.HeartbeatDuration
	timer := time.NewTicker(internal)
	defer timer.Stop()

	for {
		select {
		case <-stop:
			return
		case <-timer.C:
			timer.Reset(internal)

			r.smu.Lock()
			r.stat = r.generateRunnerStat()
			r.smu.Unlock()
		}
	}
}
