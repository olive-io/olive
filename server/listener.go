package server

import (
	"context"

	"github.com/lni/dragonboat/v4/raftio"
)

type raftEventListener struct {
	ch chan<- raftio.LeaderInfo
}

func newRaftEventListener(ch chan<- raftio.LeaderInfo) *raftEventListener {
	rel := &raftEventListener{ch: ch}
	return rel
}

func (rel *raftEventListener) LeaderUpdated(ri raftio.LeaderInfo) { rel.ch <- ri }

type systemEventListener struct{}

func newSystemEventListener() *systemEventListener {
	return &systemEventListener{}
}

func (sel *systemEventListener) NodeHostShuttingDown() {}

func (sel *systemEventListener) NodeUnloaded(info raftio.NodeInfo) {}

func (sel *systemEventListener) NodeDeleted(info raftio.NodeInfo) {}

func (sel *systemEventListener) NodeReady(info raftio.NodeInfo) {}

func (sel *systemEventListener) MembershipChanged(info raftio.NodeInfo) {}

func (sel *systemEventListener) ConnectionEstablished(info raftio.ConnectionInfo) {}

func (sel *systemEventListener) ConnectionFailed(info raftio.ConnectionInfo) {}

func (sel *systemEventListener) SendSnapshotStarted(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SendSnapshotCompleted(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SendSnapshotAborted(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SnapshotReceived(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SnapshotRecovered(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SnapshotCreated(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) SnapshotCompacted(info raftio.SnapshotInfo) {}

func (sel *systemEventListener) LogCompacted(info raftio.EntryInfo) {}

func (sel *systemEventListener) LogDBCompacted(info raftio.EntryInfo) {}

type replicaRequest struct {
	shardID     uint64
	nodeID      uint64
	staleRead   *replicaStaleRead
	syncRead    *replicaSyncRead
	syncPropose *replicaSyncPropose
}

type replicaStaleRead struct {
	query any
	rc    chan any
	ec    chan error
}

type replicaSyncRead struct {
	ctx   context.Context
	query any
	rc    chan any
	ec    chan error
}

type replicaSyncPropose struct {
	ctx  context.Context
	data []byte
	rc   chan any
	ec   chan error
}
