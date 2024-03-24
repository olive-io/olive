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

package raft

import (
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
)

func (r *Region) heartbeat() {
	duration := time.Duration(r.cfg.StatHeartBeatMs) * time.Millisecond
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		if !r.waitUtilLeader() {
			select {
			case <-r.stopc:
				return
			default:
			}
			continue
		}

		timer.Reset(duration)

	LOOP:
		for {
			select {
			case <-r.stopc:
				return
			case <-r.changeC:
				break LOOP
			case <-timer.C:
				timer.Reset(duration)
				r.tracer.Trace(&RegionStatTrace{Stat: r.stat()})
			}
		}
	}
}

func (r *Region) stat() *pb.RegionStat {
	info := r.getInfo()
	replicas := int32(len(info.Replicas))
	rs := &pb.RegionStat{
		Id:                 r.getID(),
		Leader:             r.getLeader(),
		Term:               r.getTerm(),
		Replicas:           replicas,
		Definitions:        uint64(r.metric.definition.Get()),
		RunningDefinitions: uint64(r.metric.runningDefinition.Get()),
		BpmnProcesses:      uint64(r.metric.process.Get()),
		BpmnEvents:         uint64(r.metric.event.Get()),
		BpmnTasks:          uint64(r.metric.task.Get()),
		Timestamp:          time.Now().Unix(),
	}

	return rs
}
