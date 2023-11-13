package rpc

import (
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server"
)

type header struct {
	shardID uint64
	nodeID  uint64
	sg      server.RaftStatusGetter
	rev     func() int64
}

func newHeader(ra *server.Replica) header {
	return header{
		shardID: ra.ShardID(),
		nodeID:  ra.NodeID(),
		sg:      ra,
		rev:     func() int64 { return ra.KV().Rev() },
	}
}

// fill populates pb.ResponseHeader using olive-meta information
func (h *header) fill(rh *pb.ResponseHeader) {
	if rh == nil {
		panic("unexpected nil resp.Header")
	}
	rh.ShardId = h.shardID
	rh.NodeId = h.nodeID
	rh.RaftTerm = h.sg.Term()
	if rh.Revision == 0 {
		rh.Revision = h.rev()
	}
}
