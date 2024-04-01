// Copyright 2023 The olive Authors
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
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/idutil"
	"github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/pkg/version"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/olive-io/olive/runner/raft"
)

func (r *Runner) register() (*pb.Runner, error) {
	cfg := r.cfg
	key := []byte("runner")
	bucket := buckets.Meta
	runner := new(pb.Runner)

	readTx := r.be.ReadTx()
	readTx.RLock()
	data, _ := readTx.UnsafeGet(bucket, key)
	readTx.RUnlock()
	if len(data) != 0 {
		_ = proto.Unmarshal(data, runner)
	}

	cpuTotal := uint64(0)
	cpus, err := cpu.Counts(false)
	if err != nil {
		return nil, errors.Wrap(err, "read system cpu")
	}
	cpuInfos, _ := cpu.Info()
	if len(cpuInfos) > 0 {
		cpuTotal = uint64(cpus) * uint64(cpuInfos[0].Mhz)
	}

	vm, err := mem.VirtualMemory()
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

	runner.ListenPeerURL = listenPeerURL
	runner.ListenClientURL = listenClientURL
	runner.HeartbeatMs = cfg.HeartbeatMs
	runner.Hostname, _ = os.Hostname()
	runner.Cpu = cpuTotal
	runner.Memory = vm.Total
	runner.Version = version.Version

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	if runner.Id == 0 {
		idGen, err := idutil.NewGenerator(ctx, runtime.DefaultMetaRunnerRegistrarId, r.oct.ActiveEtcdClient())
		if err != nil {
			return nil, errors.Wrap(err, "create new id generator")
		}
		runner.Id = idGen.Next()
	}

	tx := r.be.BatchTx()
	tx.Lock()
	defer tx.Unlock()
	data, _ = proto.Marshal(runner)
	tx.UnsafePut(bucket, key, data)

	return runner, nil
}

func (r *Runner) process() {

	ctx := r.ctx
	runner := r.pr
	lg := r.Logger()
	cfg := r.cfg

	rKey := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", runner.Id))
	data, _ := proto.Marshal(runner)
	_, err := r.oct.Put(ctx, rKey, string(data))
	if err != nil {
		lg.Panic("olive-runner register", zap.Error(err))
	}

	lg.Info("olive-runner registered",
		zap.Uint64("id", runner.Id),
		zap.String("listen-client-url", runner.ListenClientURL),
		zap.String("listen-peer-url", runner.ListenPeerURL),
		zap.Uint64("cpu-total", runner.Cpu),
		zap.String("memory", humanize.IBytes(runner.Memory)),
		zap.String("version", runner.Version))

	ticker := time.NewTicker(time.Duration(cfg.HeartbeatMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.StoppingNotify():
			return
		case trace := <-r.traces:
			switch tt := trace.(type) {
			case *raft.RegionStatTrace:
				stat := tt.Stat
				lg.Debug("update region stat", zap.Stringer("stat", stat))

				key := path.Join(runtime.DefaultMetaRegionStat, fmt.Sprintf("%d", stat.Id))
				data, _ = proto.Marshal(stat)
				_, err = r.oct.Put(ctx, key, string(data))
				if err != nil {
					lg.Error("olive-runner update region stat", zap.Error(err))
				}
			}
		case <-ticker.C:
			stat := r.processRunnerStat()
			stat.Timestamp = time.Now().Unix()

			lg.Debug("update runner stat", zap.Stringer("stat", stat))
			key := path.Join(runtime.DefaultMetaRunnerStat, fmt.Sprintf("%d", stat.Id))
			data, _ = proto.Marshal(stat)
			_, err = r.oct.Put(ctx, key, string(data))
			if err != nil {
				lg.Error("olive-runner update runner stat", zap.Error(err))
			}
		}
	}
}

func (r *Runner) processRunnerStat() *pb.RunnerStat {
	lg := r.Logger()
	stat := &pb.RunnerStat{
		Id:            r.pr.Id,
		Definitions:   uint64(raft.DefinitionsCounter.Get()),
		BpmnProcesses: uint64(raft.ProcessCounter.Get()),
		BpmnEvents:    uint64(raft.EventCounter.Get()),
		BpmnTasks:     uint64(raft.TaskCounter.Get()),
	}
	interval := time.Millisecond * 300
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
	stat.Regions, stat.Leaders = r.controller.RunnerStat()

	return stat
}
