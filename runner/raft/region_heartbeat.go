// MIT License
//
// Copyright (c) 2023 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package raft

import (
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
)

func (r *Region) heartbeat() {
	duration := time.Duration(r.heartbeatMs) * time.Millisecond
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		if !r.waitUtilLeader() {
			select {
			case <-r.stopping:
				return
			default:
			}
			continue
		}

		timer.Reset(duration)

	LOOP:
		for {
			select {
			case <-r.stopping:
				return
			case <-r.changeC:
				break LOOP
			case <-timer.C:
				timer.Reset(duration)
			}
		}
	}
}

func (r *Region) stat() *pb.RegionStat {
	replicas := int32(len(r.getInfo().Replicas))
	rs := &pb.RegionStat{
		Id:            r.getID(),
		Leader:        r.getLeader(),
		Term:          r.getTerm(),
		Replicas:      replicas,
		Definitions:   0,
		BpmnProcesses: 0,
		BpmnEvents:    0,
		BpmnTasks:     0,
		Timestamp:     time.Now().Unix(),
	}

	return rs
}
