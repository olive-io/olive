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

func newHeader(s *server.KVServer) header {
	cluster, err := s.InternalCluster()
	if err != nil {
		panic(err)
	}

	ra := cluster.(*server.Replica)

	return header{
		shardID: cluster.ShardID(),
		nodeID:  cluster.NodeID(),
		sg:      cluster,
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
