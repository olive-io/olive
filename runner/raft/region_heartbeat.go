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

package raft

import (
	"time"

	corev1 "github.com/olive-io/olive/apis/core/v1"
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

func (r *Region) stat() *corev1.RegionStat {
	info := r.getInfo()
	_ = info
	rs := &corev1.RegionStat{
		//Id:                 r.getID(),
		//Leader:             r.getLeader(),
		//Term:               r.getTerm(),
		//Definitions:        uint64(r.metric.definition.Get()),
		//RunningDefinitions: uint64(r.metric.runningDefinition.Get()),
		//BpmnProcesses:      uint64(r.metric.process.Get()),
		//BpmnEvents:         uint64(r.metric.event.Get()),
		//BpmnTasks:          uint64(r.metric.task.Get()),
		//Timestamp:          time.Now().Unix(),
	}

	return rs
}
