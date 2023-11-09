package client

import pb "github.com/olive-io/olive/api/serverpb"

// CompactOp represents a compact operation.
type CompactOp struct {
	revision int64
	physical bool
}

// CompactOption configures compact operation.
type CompactOption func(*CompactOp)

func (op *CompactOp) applyCompactOpts(opts []CompactOption) {
	for _, opt := range opts {
		opt(op)
	}
}

// OpCompact wraps slice CompactOption to create a CompactOp.
func OpCompact(rev int64, opts ...CompactOption) CompactOp {
	ret := CompactOp{revision: rev}
	ret.applyCompactOpts(opts)
	return ret
}

func (op CompactOp) toRequest() *pb.CompactionRequest {
	return &pb.CompactionRequest{Revision: op.revision, Physical: op.physical}
}

// WithCompactPhysical makes Compact wait until all compacted entries are
// removed from the etcd server's storage.
func WithCompactPhysical() CompactOption {
	return func(op *CompactOp) { op.physical = true }
}
