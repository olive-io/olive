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

import "github.com/lni/dragonboat/v4/raftio"

type eventListener struct {
	ch chan raftio.LeaderInfo
}

func newEventListener(ch chan raftio.LeaderInfo) *eventListener {
	return &eventListener{ch: ch}
}

func (l *eventListener) LeaderUpdated(info raftio.LeaderInfo) {
	select {
	case l.ch <- info:
	}
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
