// Copyright 2023 Lack (xingyys@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runner

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dustin/go-humanize"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/idutil"
	"github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
)

func (r *Runner) register() (*pb.Runner, error) {
	key := []byte("runner")
	bucket := buckets.Meta
	runner := new(pb.Runner)

	readTx := r.be.ReadTx()
	readTx.RLock()
	data, _ := readTx.UnsafeGet(bucket, key)
	readTx.RUnlock()
	if len(data) != 0 {
		_ = runner.Unmarshal(data)
	}

	cpus, err := cpu.Counts(false)
	if err != nil {
		return nil, errors.Wrap(err, "read system cpu")
	}
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, errors.Wrap(err, "read system memory")
	}

	runner.AdvertiseListen = r.AdvertiseListen
	runner.PeerListen = r.PeerListen
	runner.HeartbeatMs = r.HeartbeatMs
	runner.Hostname, _ = os.Hostname()
	runner.Cpus = uint32(cpus)
	runner.Memory = vm.Total
	runner.Version = version.Version

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	if runner.Id == 0 {
		idGen, err := idutil.NewGenerator(ctx, runtime.DefaultMetaRunnerRegistryId, r.oct.ActiveEtcdClient())
		if err != nil {
			return nil, errors.Wrap(err, "create new id generator")
		}
		runner.Id = idGen.Next()
	}

	tx := r.be.BatchTx()
	tx.Lock()
	defer tx.Unlock()
	data, _ = runner.Marshal()
	tx.UnsafePut(bucket, key, data)

	return runner, nil
}

func (r *Runner) registry() {
	ctx := r.ctx
	runner := r.pr
	lg := r.Logger

	rKey := path.Join(runtime.DefaultMetaRunnerRegistry, fmt.Sprintf("%d", runner.Id))
	data, _ := runner.Marshal()
	_, err := r.oct.Put(ctx, rKey, string(data))
	if err != nil {
		lg.Panic("olive-runner register", zap.Error(err))
	}

	r.Logger.Info("olive-runner registered",
		zap.Uint64("id", runner.Id),
		zap.String("advertise-listen", runner.AdvertiseListen),
		zap.String("peer-listen", runner.PeerListen),
		zap.Uint32("cpu-core", runner.Cpus),
		zap.String("memory", humanize.IBytes(runner.Memory)),
		zap.String("version", runner.Version))

	ticker := time.NewTicker(time.Duration(r.HeartbeatMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopping:
			return
		case <-ticker.C:
		}

		stat := r.processRunnerStat()
		stat.Timestamp = time.Now().Unix()

		key := path.Join(runtime.DefaultMetaRunnerStat, fmt.Sprintf("%d", stat.Id))
		data, _ = stat.Marshal()
		_, err = r.oct.Put(ctx, key, string(data))
		if err != nil {
			lg.Error("olive-runner update runner stat", zap.Error(err))
		}
	}
}

func (r *Runner) processRunnerStat() *pb.RunnerStat {
	lg := r.Logger
	stat := &pb.RunnerStat{
		Id:            r.pr.Id,
		CpuPer:        0,
		MemoryPer:     0,
		Definitions:   uint64(definitionsCounter.Get()),
		BpmnProcesses: uint64(processCounter.Get()),
		BpmnEvents:    uint64(eventCounter.Get()),
		BpmnTasks:     uint64(taskCounter.Get()),
	}
	interval := time.Millisecond * 100
	percents, err := cpu.Percent(interval, false)
	if err != nil {
		lg.Error("current cpu percent", zap.Error(err))
	}
	if len(percents) > 0 {
		stat.CpuPer = percents[0]
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		lg.Error("current memory percent", zap.Error(err))
	}
	if vm != nil {
		stat.MemoryPer = vm.UsedPercent
	}
	stat.Regions, stat.Leaders = r.raftGroup.runnerStat()

	return stat
}
