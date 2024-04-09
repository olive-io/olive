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
	"github.com/lni/dragonboat/v4/raftio"
	"github.com/olive-io/bpmn/tracing"
)

type eventListener struct {
	tracer tracing.ITracer
}

func newEventListener(tracer tracing.ITracer) *eventListener {
	return &eventListener{tracer: tracer}
}

func (l *eventListener) LeaderUpdated(info raftio.LeaderInfo) {
	l.tracer.Trace(leaderTrace(info))
}

type systemListener struct{}

func newSystemListener() *systemListener {
	return &systemListener{}
}

func (s systemListener) NodeHostShuttingDown() {}

func (s systemListener) NodeUnloaded(info raftio.NodeInfo) {}

func (s systemListener) NodeDeleted(info raftio.NodeInfo) {}

func (s systemListener) NodeReady(info raftio.NodeInfo) {}

func (s systemListener) MembershipChanged(info raftio.NodeInfo) {}

func (s systemListener) ConnectionEstablished(info raftio.ConnectionInfo) {}

func (s systemListener) ConnectionFailed(info raftio.ConnectionInfo) {}

func (s systemListener) SendSnapshotStarted(info raftio.SnapshotInfo) {}

func (s systemListener) SendSnapshotCompleted(info raftio.SnapshotInfo) {}

func (s systemListener) SendSnapshotAborted(info raftio.SnapshotInfo) {}

func (s systemListener) SnapshotReceived(info raftio.SnapshotInfo) {}

func (s systemListener) SnapshotRecovered(info raftio.SnapshotInfo) {}

func (s systemListener) SnapshotCreated(info raftio.SnapshotInfo) {}

func (s systemListener) SnapshotCompacted(info raftio.SnapshotInfo) {}

func (s systemListener) LogCompacted(info raftio.EntryInfo) {}

func (s systemListener) LogDBCompacted(info raftio.EntryInfo) {}
