package rpc

import (
	"context"

	"github.com/olive-io/olive/api/rpctypes"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server"
	"go.etcd.io/etcd/pkg/v3/adt"
)

type kvServer struct {
	hdr header
	kv  server.IRaftKV
	// maxTxnOps is the max operations per txn.
	// e.g suppose maxTxnOps = 128.
	// Txn.Success can have at most 128 operations,
	// and Txn.Failure can have at most 128 operations.
	maxTxnOps uint
}

func NewKVServer(s *server.KVServer) pb.KVServer {
	return &kvServer{hdr: newHeader(s), kv: s, maxTxnOps: s.MaxTxnOps}
}

func (s *kvServer) Range(ctx context.Context, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	if err := checkRangeRequest(r); err != nil {
		return nil, err
	}

	resp, err := s.kv.Range(ctx, s.hdr.shardID, r)
	if err != nil {
		return nil, togRPCError(err)
	}

	s.hdr.fill(resp.Header)
	return resp, nil
}

func (s *kvServer) Put(ctx context.Context, r *pb.PutRequest) (*pb.PutResponse, error) {
	if err := checkPutRequest(r); err != nil {
		return nil, err
	}

	resp, err := s.kv.Put(ctx, s.hdr.shardID, r)
	if err != nil {
		return nil, togRPCError(err)
	}

	s.hdr.fill(resp.Header)
	return resp, nil
}

func (s *kvServer) DeleteRange(ctx context.Context, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	if err := checkDeleteRequest(r); err != nil {
		return nil, err
	}

	resp, err := s.kv.DeleteRange(ctx, s.hdr.shardID, r)
	if err != nil {
		return nil, togRPCError(err)
	}

	s.hdr.fill(resp.Header)
	return resp, nil
}

func (s *kvServer) Txn(ctx context.Context, r *pb.TxnRequest) (*pb.TxnResponse, error) {
	if err := checkTxnRequest(r, int(s.maxTxnOps)); err != nil {
		return nil, err
	}
	// check for forbidden put/del overlaps after checking request to avoid quadratic blowup
	if _, _, err := checkIntervals(r.Success); err != nil {
		return nil, err
	}
	if _, _, err := checkIntervals(r.Failure); err != nil {
		return nil, err
	}

	resp, err := s.kv.Txn(ctx, s.hdr.shardID, r)
	if err != nil {
		return nil, togRPCError(err)
	}

	s.hdr.fill(resp.Header)
	return resp, nil
}

func (s *kvServer) Compact(ctx context.Context, r *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	resp, err := s.kv.Compact(ctx, s.hdr.shardID, r)
	if err != nil {
		return nil, togRPCError(err)
	}

	s.hdr.fill(resp.Header)
	return resp, nil
}

func checkRangeRequest(r *pb.RangeRequest) error {
	if len(r.Key) == 0 {
		return rpctypes.ErrGRPCEmptyKey
	}
	return nil
}

func checkPutRequest(r *pb.PutRequest) error {
	if len(r.Key) == 0 {
		return rpctypes.ErrGRPCEmptyKey
	}
	if r.IgnoreValue && len(r.Value) != 0 {
		return rpctypes.ErrGRPCValueProvided
	}
	if r.IgnoreLease && r.Lease != 0 {
		return rpctypes.ErrGRPCLeaseProvided
	}
	return nil
}

func checkDeleteRequest(r *pb.DeleteRangeRequest) error {
	if len(r.Key) == 0 {
		return rpctypes.ErrGRPCEmptyKey
	}
	return nil
}

func checkTxnRequest(r *pb.TxnRequest, maxTxnOps int) error {
	opc := len(r.Compare)
	if opc < len(r.Success) {
		opc = len(r.Success)
	}
	if opc < len(r.Failure) {
		opc = len(r.Failure)
	}
	if opc > maxTxnOps {
		return rpctypes.ErrGRPCTooManyOps
	}

	for _, c := range r.Compare {
		if len(c.Key) == 0 {
			return rpctypes.ErrGRPCEmptyKey
		}
	}
	for _, u := range r.Success {
		if err := checkRequestOp(u, maxTxnOps-opc); err != nil {
			return err
		}
	}
	for _, u := range r.Failure {
		if err := checkRequestOp(u, maxTxnOps-opc); err != nil {
			return err
		}
	}

	return nil
}

// checkIntervals tests whether puts and deletes overlap for a list of ops. If
// there is an overlap, returns an error. If no overlap, return put and delete
// sets for recursive evaluation.
func checkIntervals(reqs []*pb.RequestOp) (map[string]struct{}, adt.IntervalTree, error) {
	dels := adt.NewIntervalTree()

	// collect deletes from this level; build first to check lower level overlapped puts
	for _, req := range reqs {
		tv, ok := req.Request.(*pb.RequestOp_RequestDeleteRange)
		if !ok {
			continue
		}
		dreq := tv.RequestDeleteRange
		if dreq == nil {
			continue
		}
		var iv adt.Interval
		if len(dreq.RangeEnd) != 0 {
			iv = adt.NewStringAffineInterval(string(dreq.Key), string(dreq.RangeEnd))
		} else {
			iv = adt.NewStringAffinePoint(string(dreq.Key))
		}
		dels.Insert(iv, struct{}{})
	}

	// collect children puts/deletes
	puts := make(map[string]struct{})
	for _, req := range reqs {
		tv, ok := req.Request.(*pb.RequestOp_RequestTxn)
		if !ok {
			continue
		}
		putsThen, delsThen, err := checkIntervals(tv.RequestTxn.Success)
		if err != nil {
			return nil, dels, err
		}
		putsElse, delsElse, err := checkIntervals(tv.RequestTxn.Failure)
		if err != nil {
			return nil, dels, err
		}
		for k := range putsThen {
			if _, ok := puts[k]; ok {
				return nil, dels, rpctypes.ErrGRPCDuplicateKey
			}
			if dels.Intersects(adt.NewStringAffinePoint(k)) {
				return nil, dels, rpctypes.ErrGRPCDuplicateKey
			}
			puts[k] = struct{}{}
		}
		for k := range putsElse {
			if _, ok := puts[k]; ok {
				// if key is from putsThen, overlap is OK since
				// either then/else are mutually exclusive
				if _, isSafe := putsThen[k]; !isSafe {
					return nil, dels, rpctypes.ErrGRPCDuplicateKey
				}
			}
			if dels.Intersects(adt.NewStringAffinePoint(k)) {
				return nil, dels, rpctypes.ErrGRPCDuplicateKey
			}
			puts[k] = struct{}{}
		}
		dels.Union(delsThen, adt.NewStringAffineInterval("\x00", ""))
		dels.Union(delsElse, adt.NewStringAffineInterval("\x00", ""))
	}

	// collect and check this level's puts
	for _, req := range reqs {
		tv, ok := req.Request.(*pb.RequestOp_RequestPut)
		if !ok || tv.RequestPut == nil {
			continue
		}
		k := string(tv.RequestPut.Key)
		if _, ok := puts[k]; ok {
			return nil, dels, rpctypes.ErrGRPCDuplicateKey
		}
		if dels.Intersects(adt.NewStringAffinePoint(k)) {
			return nil, dels, rpctypes.ErrGRPCDuplicateKey
		}
		puts[k] = struct{}{}
	}
	return puts, dels, nil
}

func checkRequestOp(u *pb.RequestOp, maxTxnOps int) error {
	// TODO: ensure only one of the field is set.
	switch uv := u.Request.(type) {
	case *pb.RequestOp_RequestRange:
		return checkRangeRequest(uv.RequestRange)
	case *pb.RequestOp_RequestPut:
		return checkPutRequest(uv.RequestPut)
	case *pb.RequestOp_RequestDeleteRange:
		return checkDeleteRequest(uv.RequestDeleteRange)
	case *pb.RequestOp_RequestTxn:
		return checkTxnRequest(uv.RequestTxn, maxTxnOps)
	default:
		// empty op / nil entry
		return rpctypes.ErrGRPCKeyNotFound
	}
}