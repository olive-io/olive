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
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

//func (r *Runner) register() (runner *pb.Runner, err error) {
//	return runner, nil
//}

func (r *Runner) process() {

	//ctx := r.ctx
	//runner := r.pr
	//lg := r.Logger()
	//cfg := r.cfg
	//
	//rKey := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", runner.Id))
	//data, _ := proto.Marshal(runner)
	//_, err := r.oct.Put(ctx, rKey, string(data))
	//if err != nil {
	//	lg.Panic("olive-runner register", zap.Error(err))
	//}
	//
	//lg.Info("olive-runner registered",
	//	zap.Uint64("id", runner.Id),
	//	zap.String("listen-client-url", runner.ListenClientURL),
	//	zap.String("listen-peer-url", runner.ListenPeerURL),
	//	zap.Uint64("cpu-total", runner.Cpu),
	//	zap.String("memory", humanize.IBytes(runner.Memory)),
	//	zap.String("version", runner.Version))
	//
	//ticker := time.NewTicker(time.Duration(cfg.HeartbeatMs) * time.Millisecond)
	//defer ticker.Stop()
	//for {
	//	select {
	//	case <-r.StoppingNotify():
	//		return
	//	case trace := <-r.traces:
	//		switch tt := trace.(type) {
	//		case *raft.RegionStatTrace:
	//			stat := tt.Stat
	//			lg.Debug("update region stat", zap.Stringer("stat", stat))
	//
	//			key := path.Join(runtime.DefaultMetaRegionStat, fmt.Sprintf("%d", stat.Id))
	//			data, _ = proto.Marshal(stat)
	//			_, err = r.oct.Put(ctx, key, string(data))
	//			if err != nil {
	//				lg.Error("olive-runner update region stat", zap.Error(err))
	//			}
	//		}
	//	case <-ticker.C:
	//		stat := r.processRunnerStat()
	//		stat.Timestamp = time.Now().Unix()
	//
	//		lg.Debug("update runner stat", zap.Stringer("stat", stat))
	//		key := path.Join(runtime.DefaultMetaRunnerStat, fmt.Sprintf("%d", stat.Id))
	//		data, _ = proto.Marshal(stat)
	//		_, err = r.oct.Put(ctx, key, string(data))
	//		if err != nil {
	//			lg.Error("olive-runner update runner stat", zap.Error(err))
	//		}
	//	}
	//}
}

func (r *Runner) processRunnerStat() (stat *corev1.RunnerStatistics) {
	//lg := r.Logger()
	//stat := &pb.RunnerStat{
	//	Id:            r.pr.Id,
	//	Definitions:   uint64(raft.DefinitionsCounter.Get()),
	//	BpmnProcesses: uint64(raft.ProcessCounter.Get()),
	//	BpmnEvents:    uint64(raft.EventCounter.Get()),
	//	BpmnTasks:     uint64(raft.TaskCounter.Get()),
	//}
	//interval := time.Millisecond * 300
	//percents, err := cpu.Percent(interval, false)
	//if err != nil {
	//	lg.Error("current cpu percent", zap.Error(err))
	//}
	//if len(percents) > 0 {
	//	stat.CpuPer = percents[0]
	//}
	//
	//vm, err := mem.VirtualMemory()
	//if err != nil {
	//	lg.Error("current memory percent", zap.Error(err))
	//}
	//if vm != nil {
	//	stat.MemoryPer = vm.UsedPercent
	//}
	//stat.Regions, stat.Leaders = r.controller.RunnerStat()
	//
	//return stat
	return
}
