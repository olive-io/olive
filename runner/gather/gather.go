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
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner/metrics"
	"github.com/olive-io/olive/runner/storage"
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
	bs  storage.Storage

	runner *types.Runner
}

func NewGather(ctx context.Context, cfg *Config, bs storage.Storage) (*Gather, error) {
	runner := new(types.Runner)
	resp, err := bs.Get(ctx, runnerPrefix)
	if err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return nil, err
		}

		runner = &types.Runner{}
		runner.Name = cfg.Name

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

	runner.Name = cfg.Name
	runner.ListenURL = cfg.ListenURL
	runner.HeartbeatMs = cfg.HeartbeatMs
	runner.Hostname, _ = os.Hostname()
	runner.Features = metrics.GetFeatures()
	runner.Cpu = cpuTotal
	runner.Memory = vm.Total
	runner.Version = version.Version

	delegate := &Gather{
		ctx: ctx,
		cfg: cfg,
		bs:  bs,

		runner: runner,
	}

	if err = delegate.SaveRunner(ctx, runner); err != nil {
		return delegate, err
	}

	return delegate, nil
}

func (d *Gather) GetRunner() *types.Runner {
	return proto.Clone(d.runner).(*types.Runner)
}

func (d *Gather) SaveRunner(ctx context.Context, runner *types.Runner) error {
	d.runner = runner
	if err := d.bs.Put(ctx, runnerPrefix, runner); err != nil {
		return err
	}
	return nil
}

func (d *Gather) GetStat() *types.RunnerStat {
	rs := &types.RunnerStat{
		Id:            d.runner.Id,
		BpmnProcesses: uint64(metrics.ProcessCounter.Get()),
		BpmnTasks:     uint64(metrics.TaskCounter.Get()),
		Timestamp:     time.Now().UnixNano(),
	}
	interval := time.Millisecond * 300
	percents, err := cpu.Percent(interval, false)
	if err == nil {
		rs.CpuUsed = percents[0] * float64(d.runner.Cpu)
	}
	if len(percents) > 0 {
		rs.CpuUsed = percents[0] * float64(d.runner.Cpu)
	}

	vm, err := mem.VirtualMemory()
	if err == nil {
		rs.MemoryUsed = vm.UsedPercent * float64(d.runner.Memory)
	}

	return rs
}
