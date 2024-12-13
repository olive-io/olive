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

package gather

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/olive-io/olive/api/meta"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/metrics"
)

const runnerPrefix = "/runner"

type Config struct {
	Name        string
	ListenURL   string
	HeartbeatMs int64
}

// Gather gets the information of the runner.
type Gather struct {
	ctx context.Context
	cfg *Config
	be  backend.IBackend

	runner *types.Runner
}

func NewGather(ctx context.Context, cfg *Config, be backend.IBackend) (*Gather, error) {
	runner := new(types.Runner)
	resp, err := be.Get(ctx, runnerPrefix)
	if err != nil {
		if !errors.Is(err, backend.ErrNotFound) {
			return nil, err
		}

		runner = &types.Runner{
			ObjectMeta: meta.ObjectMeta{
				Name:   cfg.Name,
				Labels: map[string]string{},
			},
		}

	} else {
		if err = resp.Unmarshal(runner); err != nil {
			return nil, err
		}
	}

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

	runner.ListenURL = cfg.ListenURL
	runner.HeartbeatMs = cfg.HeartbeatMs
	runner.Hostname, _ = os.Hostname()
	runner.Features = metrics.GetFeatures()
	runner.CPU = int32(cpuTotal)
	runner.Memory = int64(vm.Total)
	runner.Version = version.Version

	if err = be.Put(ctx, runnerPrefix, runner); err != nil {
		return nil, err
	}

	delegate := &Gather{
		ctx: ctx,
		cfg: cfg,
		be:  be,

		runner: runner,
	}

	return delegate, nil
}

func (d *Gather) GetRunner() *types.Runner {
	in := d.runner
	out := &types.Runner{
		ObjectMeta: meta.ObjectMeta{
			Name:   in.Name,
			Labels: in.Labels,
		},
		ListenURL:   in.ListenURL,
		Version:     in.Version,
		HeartbeatMs: in.HeartbeatMs,
		Hostname:    in.Hostname,
		CPU:         in.CPU,
		Memory:      in.Memory,
		DiskSize:    in.DiskSize,
	}
	return out
}

func (d *Gather) GetStat() (*types.RunnerStatistics, error) {
	rs := &types.RunnerStatistics{
		Name:          d.runner.Name,
		BpmnProcesses: uint64(metrics.ProcessCounter.Get()),
		BpmnEvents:    uint64(metrics.EventCounter.Get()),
		BpmnTasks:     uint64(metrics.TaskCounter.Get()),
	}
	interval := time.Millisecond * 300
	percents, err := cpu.Percent(interval, false)
	if err != nil {
		return nil, fmt.Errorf("current cpu percent: %w", err)
	}
	if len(percents) > 0 {
		rs.CPUUsed = percents[0] * float64(d.runner.CPU)
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("current memory percent: %w", err)
	}
	if vm != nil {
		rs.MemoryUsed = vm.UsedPercent * float64(d.runner.Memory)
	}

	return rs, nil
}
