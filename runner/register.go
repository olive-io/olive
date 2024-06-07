/*
Copyright 2023 The olive Authors

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
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dustin/go-humanize"
	pscpu "github.com/shirou/gopsutil/v3/cpu"
	psmem "github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/apis/version"
	ort "github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/olive-io/olive/runner/raft"
)

func (r *Runner) getRunner() *corev1.Runner {
	r.prMu.RLock()
	defer r.prMu.RUnlock()
	return r.pr.DeepCopy()
}

func (r *Runner) setRunner(runner *corev1.Runner) {
	r.prMu.Lock()
	r.pr = runner
	r.prMu.Unlock()
}

func (r *Runner) persistRunner(runner *corev1.Runner) {
	r.setRunner(runner)

	key := []byte("runner")
	bucket := buckets.Meta

	tx := r.be.BatchTx()
	tx.Lock()
	defer tx.Unlock()
	data, _ := runner.Marshal()
	_ = tx.UnsafePut(bucket, key, data)
}

func (r *Runner) register() (*corev1.Runner, error) {
	cfg := r.cfg
	key := []byte("runner")
	bucket := buckets.Meta
	runner := &corev1.Runner{}
	r.scheme.Default(runner)

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	readTx := r.be.ReadTx()
	readTx.RLock()
	data, _ := readTx.UnsafeGet(bucket, key)
	readTx.RUnlock()
	if len(data) != 0 {
		if e1 := runner.Unmarshal(data); e1 != nil {
			r.Logger().Error("Unmarshal runner", zap.Error(e1))
		}
		latest, err := r.oct.CoreV1().Runners().Get(ctx, runner.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("sync runner: %w", err)
		}
		if latest.Spec.ID == 0 {
			runner.ResourceVersion = latest.ResourceVersion
		} else {
			runner = latest
		}
	}

	cpuTotal := float64(0)
	cpus, err := pscpu.Counts(false)
	if err != nil {
		return nil, errors.Wrap(err, "read system cpu")
	}
	cpuInfos, _ := pscpu.Info()
	if len(cpuInfos) > 0 {
		cpuTotal = float64(cpus) * cpuInfos[0].Mhz
	}

	vm, err := psmem.VirtualMemory()
	if err != nil {
		return nil, errors.Wrap(err, "read system memory")
	}

	listenPeerURL := cfg.AdvertisePeerURL
	if len(listenPeerURL) == 0 {
		listenPeerURL = cfg.ListenPeerURL
	}
	listenClientURL := cfg.AdvertiseClientURL
	if len(listenClientURL) == 0 {
		listenClientURL = cfg.ListenClientURL
	}

	runner.Spec.PeerURL = listenPeerURL
	runner.Spec.ClientURL = listenClientURL
	runner.Spec.Hostname, _ = os.Hostname()
	runner.Spec.VersionRef = version.Version
	if runner.Status.Stat == nil {
		runner.Status.Stat = &corev1.RunnerStat{}
	}
	runner.Status.CpuTotal = cpuTotal
	runner.Status.MemoryTotal = float64(vm.Total)

	if runner.Spec.ID == 0 {
		runner.Name = "runner_default" // runner not be empty
		runner, err = r.oct.CoreV1().Runners().Create(ctx, runner, metav1.CreateOptions{})
	} else {
		runner, err = r.oct.CoreV1().Runners().Update(ctx, runner, metav1.UpdateOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("register runner: %w", err)
	}

	return runner, nil
}

func (r *Runner) process() {

	ctx := r.ctx
	runner := r.getRunner()
	lg := r.Logger()
	cfg := r.cfg

	rKey := path.Join(ort.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", runner.Spec.ID))
	data, _ := runner.Marshal()
	_, err := r.oct.Put(ctx, rKey, string(data))
	if err != nil {
		lg.Panic("olive-runner register", zap.Error(err))
	}

	lg.Info("olive-runner registered",
		zap.Int64("id", runner.Spec.ID),
		zap.String("listen-client-url", runner.Spec.ClientURL),
		zap.String("listen-peer-url", runner.Spec.PeerURL),
		zap.Float64("cpu", runner.Status.CpuTotal),
		zap.String("memory", humanize.IBytes(uint64(runner.Status.MemoryTotal))),
		zap.String("version", runner.Spec.VersionRef))

	ticker := time.NewTicker(time.Duration(cfg.HeartbeatMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.StoppingNotify():
			return
		case trace := <-r.traces:
			switch tt := trace.(type) {
			case *raft.RegionStatTrace:
				_ = tt
				//stat := tt.Stat
				//lg.Debug("update region stat", zap.Stringer("stat", stat))
				//
				//key := path.Join(ort.DefaultMetaRegionStat, fmt.Sprintf("%d", stat.Id))
				//data, _ = proto.Marshal(stat)
				//_, err = r.oct.Put(ctx, key, string(data))
				//if err != nil {
				//	lg.Error("olive-runner update region stat", zap.Error(err))
				//}
			}
		case <-ticker.C:
			applied, updateOptions := r.updateRunnerStat()

			lg.Debug("update runner stat", zap.Stringer("stat", applied.Status.Stat))
			runner, err = r.oct.CoreV1().Runners().UpdateStatus(ctx, applied, updateOptions)
			if err != nil {
				lg.Error("olive-runner update runner stat", zap.Error(err))
			} else {
				r.setRunner(runner)
			}
		}
	}
}

func (r *Runner) updateRunnerStat() (*corev1.Runner, metav1.UpdateOptions) {
	lg := r.Logger()
	runner := r.getRunner()
	stat := runner.Status.Stat
	if stat == nil {
		stat = &corev1.RunnerStat{}
	}

	updateOptions := metav1.UpdateOptions{}

	runner.Status.Phase = corev1.RunnerActive
	runner.Status.Definitions = int64(raft.DefinitionsCounter.Get())

	stat.Bpmn = &corev1.BpmnStat{
		Processes: int64(raft.ProcessCounter.Get()),
		Events:    int64(raft.EventCounter.Get()),
		Tasks:     int64(raft.TaskCounter.Get()),
	}
	regions, leaders := r.controller.RunnerStat()
	runner.Status.Regions = regions
	runner.Status.Leaders = leaders

	interval := time.Millisecond * 500
	percents, err := pscpu.Percent(interval, false)
	if err != nil {
		lg.Error("current cpu percent", zap.Error(err))
	}
	if len(percents) > 0 && percents[0] > 0 {
		stat.CpuUsed = percents[0] / 100 * runner.Status.CpuTotal
	}

	vm, err := psmem.VirtualMemory()
	if err != nil {
		lg.Error("current memory percent", zap.Error(err))
	}
	if vm != nil {
		stat.MemoryUsed = float64(vm.Used)
	}
	stat.Timeout = time.Now().Unix()

	return runner, updateOptions
}
