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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

func (s *Shard) heartbeat() {
	duration := time.Duration(s.cfg.StatHeartBeatMs) * time.Millisecond
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		if !s.waitUtilLeader() {
			select {
			case <-s.stopc:
				return
			default:
			}
			continue
		}

		timer.Reset(duration)

	LOOP:
		for {
			select {
			case <-s.stopc:
				return
			case <-s.changeC:
				break LOOP
			case <-timer.C:
				timer.Reset(duration)
				s.tracer.Trace(&regionStatTrace{Id: s.getID(), stat: s.stat()})
			}
		}
	}
}

func (s *Shard) stat() *corev1.RegionStat {
	rs := &corev1.RegionStat{
		TypeMeta: metav1.TypeMeta{},
		Bpmn: &corev1.BpmnStat{
			Definitions: int64(s.metric.definition.Get()),
			Processes:   int64(s.metric.process.Get()),
			Events:      int64(s.metric.event.Get()),
			Tasks:       int64(s.metric.task.Get()),
		},
		RunningDefinitions: int64(s.metric.runningDefinition.Get()),
		Leader:             int64(s.getLeader()),
		Term:               int64(s.getTerm()),
		Timeout:            time.Now().Unix(),
	}

	return rs
}
