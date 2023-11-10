package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server/auth"
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/mvcc"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.uber.org/zap"
)

type applyResult struct {
	resp proto.Message
	err  error
	// physc signals the physical effect of the request has completed in addition
	// to being logically reflected by the node. Currently only used for
	// Compaction requests.
	physc <-chan struct{}
	trace *traceutil.Trace
}

// applier is the interface for processing rafe message
type applier interface {
	Apply(r *pb.InternalRaftRequest) *applyResult

	Put(ctx context.Context, txn mvcc.ITxnWrite, p *pb.PutRequest) (*pb.PutResponse, *traceutil.Trace, error)
	Range(ctx context.Context, txn mvcc.ITxnRead, r *pb.RangeRequest) (*pb.RangeResponse, error)
	DeleteRange(txn mvcc.ITxnWrite, dr *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error)
	Txn(ctx context.Context, rt *pb.TxnRequest) (*pb.TxnResponse, *traceutil.Trace, error)
	Compaction(compaction *pb.CompactionRequest) (*pb.CompactionResponse, <-chan struct{}, *traceutil.Trace, error)

	LeaseGrant(lc *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error)
	LeaseRevoke(lc *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error)

	LeaseCheckpoint(lc *pb.LeaseCheckpointRequest) (*pb.LeaseCheckpointResponse, error)

	Authenticate(r *pb.InternalAuthenticateRequest) (*pb.AuthenticateResponse, error)

	AuthEnable() (*pb.AuthEnableResponse, error)
	AuthDisable() (*pb.AuthDisableResponse, error)
	AuthStatus() (*pb.AuthStatusResponse, error)

	UserAdd(ua *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error)
	UserDelete(ua *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error)
	UserChangePassword(ua *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error)
	UserGrantRole(ua *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error)
	UserGet(ua *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error)
	UserRevokeRole(ua *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error)
	RoleAdd(ua *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error)
	RoleGrantPermission(ua *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error)
	RoleGet(ua *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error)
	RoleRevokePermission(ua *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error)
	RoleDelete(ua *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error)
	UserList(ua *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error)
	RoleList(ua *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error)
}

type checkReqFunc func(mvcc.IReadView, *pb.RequestOp) error

type applierBackend struct {
	ra *Replica

	checkPut   checkReqFunc
	checkRange checkReqFunc
}

func (ra *Replica) newApplierBackend() applier {
	base := &applierBackend{ra: ra}
	base.checkPut = func(rv mvcc.IReadView, req *pb.RequestOp) error {
		return base.checkRequestPut(rv, req)
	}
	base.checkRange = func(rv mvcc.IReadView, req *pb.RequestOp) error {
		return base.checkRequestRange(rv, req)
	}
	return base
}

func (ra *Replica) newApplier() applier {
	return newAuthApplier(
		ra.AuthStore(),
		ra.newApplierBackend(),
		ra.lessor,
	)
}

func (a *applierBackend) Apply(r *pb.InternalRaftRequest) *applyResult {
	op := "unknown"
	ar := &applyResult{}
	defer func(start time.Time) {
		success := ar.err == nil || errors.Is(ar.err, mvcc.ErrCompacted)
		applySec.WithLabelValues("v1", op, strconv.FormatBool(success)).Observe(time.Since(start).Seconds())
		warnOfExpensiveRequest(a.ra.lg, a.ra.WarningApplyDuration, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		if !success {
			warnOfFailedRequest(a.ra.lg, start, &pb.InternalRaftStringer{Request: r}, ar.resp, ar.err)
		}
	}(time.Now())

	// call into a.ra.apply.F instead of a.F so upper appliers can check individual calls
	switch {
	case r.Range != nil:
		op = "Range"
		ar.resp, ar.err = a.ra.apply.Range(context.TODO(), nil, r.Range)
	case r.Put != nil:
		op = "Put"
		ar.resp, ar.trace, ar.err = a.ra.apply.Put(context.TODO(), nil, r.Put)
	case r.DeleteRange != nil:
		op = "DeleteRange"
		ar.resp, ar.err = a.ra.apply.DeleteRange(nil, r.DeleteRange)
	case r.Txn != nil:
		op = "Txn"
		ar.resp, ar.trace, ar.err = a.ra.apply.Txn(context.TODO(), r.Txn)
	case r.Compaction != nil:
		op = "Compaction"
		ar.resp, ar.physc, ar.trace, ar.err = a.ra.apply.Compaction(r.Compaction)
	case r.LeaseGrant != nil:
		op = "LeaseGrant"
		ar.resp, ar.err = a.ra.apply.LeaseGrant(r.LeaseGrant)
	case r.LeaseRevoke != nil:
		op = "LeaseRevoke"
		ar.resp, ar.err = a.ra.apply.LeaseRevoke(r.LeaseRevoke)
	case r.LeaseCheckpoint != nil:
		op = "LeaseCheckpoint"
		ar.resp, ar.err = a.ra.apply.LeaseCheckpoint(r.LeaseCheckpoint)
	case r.Authenticate != nil:
		op = "Authenticate"
		ar.resp, ar.err = a.ra.apply.Authenticate(r.Authenticate)
	case r.AuthEnable != nil:
		op = "AuthEnable"
		ar.resp, ar.err = a.ra.apply.AuthEnable()
	case r.AuthDisable != nil:
		op = "AuthDisable"
		ar.resp, ar.err = a.ra.apply.AuthDisable()
	case r.AuthStatus != nil:
		ar.resp, ar.err = a.ra.apply.AuthStatus()
	case r.AuthUserAdd != nil:
		op = "AuthUserAdd"
		ar.resp, ar.err = a.ra.apply.UserAdd(r.AuthUserAdd)
	case r.AuthUserDelete != nil:
		op = "AuthUserDelete"
		ar.resp, ar.err = a.ra.apply.UserDelete(r.AuthUserDelete)
	case r.AuthUserChangePassword != nil:
		op = "AuthUserChangePassword"
		ar.resp, ar.err = a.ra.apply.UserChangePassword(r.AuthUserChangePassword)
	case r.AuthUserGrantRole != nil:
		op = "AuthUserGrantRole"
		ar.resp, ar.err = a.ra.apply.UserGrantRole(r.AuthUserGrantRole)
	case r.AuthUserGet != nil:
		op = "AuthUserGet"
		ar.resp, ar.err = a.ra.apply.UserGet(r.AuthUserGet)
	case r.AuthUserRevokeRole != nil:
		op = "AuthUserRevokeRole"
		ar.resp, ar.err = a.ra.apply.UserRevokeRole(r.AuthUserRevokeRole)
	case r.AuthRoleAdd != nil:
		op = "AuthRoleAdd"
		ar.resp, ar.err = a.ra.apply.RoleAdd(r.AuthRoleAdd)
	case r.AuthRoleGrantPermission != nil:
		op = "AuthRoleGrantPermission"
		ar.resp, ar.err = a.ra.apply.RoleGrantPermission(r.AuthRoleGrantPermission)
	case r.AuthRoleGet != nil:
		op = "AuthRoleGet"
		ar.resp, ar.err = a.ra.apply.RoleGet(r.AuthRoleGet)
	case r.AuthRoleRevokePermission != nil:
		op = "AuthRoleRevokePermission"
		ar.resp, ar.err = a.ra.apply.RoleRevokePermission(r.AuthRoleRevokePermission)
	case r.AuthRoleDelete != nil:
		op = "AuthRoleDelete"
		ar.resp, ar.err = a.ra.apply.RoleDelete(r.AuthRoleDelete)
	case r.AuthUserList != nil:
		op = "AuthUserList"
		ar.resp, ar.err = a.ra.apply.UserList(r.AuthUserList)
	case r.AuthRoleList != nil:
		op = "AuthRoleList"
		ar.resp, ar.err = a.ra.apply.RoleList(r.AuthRoleList)
	default:
		a.ra.lg.Panic("not implemented apply", zap.Stringer("raft-request", r))
	}
	return ar
}

func (a *applierBackend) Put(ctx context.Context, txn mvcc.ITxnWrite, p *pb.PutRequest) (resp *pb.PutResponse, trace *traceutil.Trace, err error) {
	resp = &pb.PutResponse{}
	resp.Header = &pb.ResponseHeader{}
	trace = traceutil.Get(ctx)
	// create put tracing if the trace in context is empty
	if trace.IsEmpty() {
		trace = traceutil.New("put",
			a.ra.lg,
			traceutil.Field{Key: "key", Value: string(p.Key)},
			traceutil.Field{Key: "req_size", Value: p.XSize()},
		)
	}
	val, leaseID := p.Value, lease.LeaseID(p.Lease)
	if txn == nil {
		if leaseID != lease.NoLease {
			if l := a.ra.lessor.Lookup(leaseID); l == nil {
				return nil, nil, lease.ErrLeaseNotFound
			}
		}
		txn = a.ra.KV().Write(trace)
		defer txn.End()
	}

	var rr *mvcc.RangeResult
	if p.IgnoreValue || p.PrevKv {
		trace.StepWithFunction(func() {
			rr, err = txn.Range(context.TODO(), p.Key, nil, mvcc.RangeOptions{})
		}, "get previous kv pair")

		if err != nil {
			return nil, nil, err
		}
	}
	if p.IgnoreValue {
		if rr == nil || len(rr.KVs) == 0 {
			// ignore_{lease,value} flag expects previous key-value pair
			return nil, nil, ErrKeyNotFound
		}
	}
	if p.IgnoreValue {
		val = rr.KVs[0].Value
	}
	if p.IgnoreLease {
		leaseID = lease.LeaseID(rr.KVs[0].Lease)
	}
	if p.PrevKv {
		if rr != nil && len(rr.KVs) != 0 {
			resp.PrevKv = &rr.KVs[0]
		}
	}

	resp.Header.Revision = txn.Put(p.Key, val, leaseID)
	trace.AddField(traceutil.Field{Key: "response_revision", Value: resp.Header.Revision})
	return resp, trace, nil
}

func (a *applierBackend) Range(ctx context.Context, txn mvcc.ITxnRead, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	trace := traceutil.Get(ctx)

	resp := &pb.RangeResponse{}
	resp.Header = &pb.ResponseHeader{}

	if txn == nil {
		txn = a.ra.kv.Read(mvcc.ConcurrentReadTxMode, trace)
		defer txn.End()
	}

	limit := r.Limit
	if r.SortOrder != pb.RangeRequest_NONE ||
		r.MinModRevision != 0 || r.MaxModRevision != 0 ||
		r.MinCreateRevision != 0 || r.MaxCreateRevision != 0 {
		// fetch everything; sort and truncate afterwards
		limit = 0
	}
	if limit > 0 {
		// fetch one extra for 'more' flag
		limit = limit + 1
	}

	ro := mvcc.RangeOptions{
		Limit: limit,
		Rev:   r.Revision,
		Count: r.CountOnly,
	}

	rr, err := txn.Range(ctx, r.Key, mkGteRange(r.RangeEnd), ro)
	if err != nil {
		return nil, err
	}

	if r.MultiVersion && len(r.RangeEnd) == 0 {
		revs, err := txn.Versions(ctx, r.Key)
		if err != nil {
			return nil, err
		}
		resp.Versions = revs
	}

	if r.MaxModRevision != 0 {
		f := func(kv *pb.KeyValue) bool { return kv.ModRevision > r.MaxModRevision }
		pruneKVs(rr, f)
	}
	if r.MinModRevision != 0 {
		f := func(kv *pb.KeyValue) bool { return kv.ModRevision < r.MinModRevision }
		pruneKVs(rr, f)
	}
	if r.MaxCreateRevision != 0 {
		f := func(kv *pb.KeyValue) bool { return kv.CreateRevision > r.MaxCreateRevision }
		pruneKVs(rr, f)
	}
	if r.MinCreateRevision != 0 {
		f := func(kv *pb.KeyValue) bool { return kv.CreateRevision < r.MinCreateRevision }
		pruneKVs(rr, f)
	}

	sortOrder := r.SortOrder
	if r.SortTarget != pb.RangeRequest_KEY && sortOrder == pb.RangeRequest_NONE {
		// Since current mvcc.Range implementation returns results
		// sorted by keys in lexiographically ascending order,
		// sort ASCEND by default only when target is not 'KEY'
		sortOrder = pb.RangeRequest_ASCEND
	}
	if sortOrder != pb.RangeRequest_NONE {
		var sorter sort.Interface
		switch {
		case r.SortTarget == pb.RangeRequest_KEY:
			sorter = &kvSortByKey{&kvSort{rr.KVs}}
		case r.SortTarget == pb.RangeRequest_VERSION:
			sorter = &kvSortByVersion{&kvSort{rr.KVs}}
		case r.SortTarget == pb.RangeRequest_CREATE:
			sorter = &kvSortByCreate{&kvSort{rr.KVs}}
		case r.SortTarget == pb.RangeRequest_MOD:
			sorter = &kvSortByMod{&kvSort{rr.KVs}}
		case r.SortTarget == pb.RangeRequest_VALUE:
			sorter = &kvSortByValue{&kvSort{rr.KVs}}
		}
		switch {
		case sortOrder == pb.RangeRequest_ASCEND:
			sort.Sort(sorter)
		case sortOrder == pb.RangeRequest_DESCEND:
			sort.Sort(sort.Reverse(sorter))
		}
	}

	if r.Limit > 0 && len(rr.KVs) > int(r.Limit) {
		rr.KVs = rr.KVs[:r.Limit]
		resp.More = true
	}
	trace.Step("filter and sort the key-value pairs")
	resp.Header.Revision = rr.Rev
	resp.Count = int64(rr.Count)
	resp.Kvs = make([]*pb.KeyValue, len(rr.KVs))
	for i := range rr.KVs {
		if r.KeysOnly {
			rr.KVs[i].Value = nil
		}
		resp.Kvs[i] = &rr.KVs[i]
	}
	trace.Step("assemble the response")
	return resp, nil
}

func (a *applierBackend) DeleteRange(txn mvcc.ITxnWrite, dr *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	resp := &pb.DeleteRangeResponse{}
	resp.Header = &pb.ResponseHeader{}
	end := mkGteRange(dr.RangeEnd)

	if txn == nil {
		txn = a.ra.kv.Write(traceutil.TODO())
		defer txn.End()
	}

	if dr.PrevKv {
		rr, err := txn.Range(context.TODO(), dr.Key, end, mvcc.RangeOptions{})
		if err != nil {
			return nil, err
		}
		if rr != nil {
			resp.PrevKvs = make([]*pb.KeyValue, len(rr.KVs))
			for i := range rr.KVs {
				resp.PrevKvs[i] = &rr.KVs[i]
			}
		}
	}

	resp.Deleted, resp.Header.Revision = txn.DeleteRange(dr.Key, end)
	return resp, nil
}

func (a *applierBackend) Txn(ctx context.Context, rt *pb.TxnRequest) (*pb.TxnResponse, *traceutil.Trace, error) {
	lg := a.ra.lg
	trace := traceutil.Get(ctx)
	if trace.IsEmpty() {
		trace = traceutil.New("transaction", a.ra.lg)
		ctx = context.WithValue(ctx, traceutil.TraceKey, trace)
	}
	isWrite := !isTxnReadonly(rt)

	// When the transaction contains write operations, we use ReadTx instead of
	// ConcurrentReadTx to avoid extra overhead of copying buffer.
	var txn mvcc.ITxnWrite
	if isWrite && a.ra.TxnModeWriteWithSharedBuffer {
		txn = mvcc.NewReadOnlyTxnWrite(a.ra.KV().Read(mvcc.SharedBufReadTxMode, trace))
	} else {
		txn = mvcc.NewReadOnlyTxnWrite(a.ra.KV().Read(mvcc.ConcurrentReadTxMode, trace))
	}

	var txnPath []bool
	trace.StepWithFunction(
		func() {
			txnPath = compareToPath(txn, rt)
		},
		"compare",
	)

	if isWrite {
		trace.AddField(traceutil.Field{Key: "read_only", Value: false})
		if _, err := checkRequests(txn, rt, txnPath, a.checkPut); err != nil {
			txn.End()
			return nil, nil, err
		}
	}
	if _, err := checkRequests(txn, rt, txnPath, a.checkRange); err != nil {
		txn.End()
		return nil, nil, err
	}
	trace.Step("check requests")
	txnResp, _ := newTxnResp(rt, txnPath)

	// When executing mutable txn ops, olive must hold the txn lock so
	// readers do not see any intermediate results. Since writes are
	// serialized on the raft loop, the revision in the read view will
	// be the revision of the write txn.
	if isWrite {
		txn.End()
		txn = a.ra.KV().Write(trace)
	}
	_, err := a.applyTxn(ctx, txn, rt, txnPath, txnResp)
	if err != nil {
		if isWrite {
			// end txn to release locks before panic
			txn.End()
			// When txn with write operations starts it has to be successful
			// We don't have a way to recover state in case of write failure
			lg.Panic("unexpected error during txn with writes", zap.Error(err))
		} else {
			lg.Error("unexpected error during readonly txn", zap.Error(err))
		}
	}
	rev := txn.Rev()
	if len(txn.Changes()) != 0 {
		rev++
	}
	txn.End()

	txnResp.Header.Revision = rev
	trace.AddField(
		traceutil.Field{Key: "number_of_response", Value: len(txnResp.Responses)},
		traceutil.Field{Key: "response_revision", Value: txnResp.Header.Revision},
	)
	return txnResp, trace, err
}

// newTxnResp allocates a txn response for a txn request given a path.
func newTxnResp(rt *pb.TxnRequest, txnPath []bool) (txnResp *pb.TxnResponse, txnCount int) {
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}
	resps := make([]*pb.ResponseOp, len(reqs))
	txnResp = &pb.TxnResponse{
		Responses: resps,
		Succeeded: txnPath[0],
		Header:    &pb.ResponseHeader{},
	}
	for i, req := range reqs {
		switch tv := req.Request.(type) {
		case *pb.RequestOp_RequestRange:
			resps[i] = &pb.ResponseOp{Response: &pb.ResponseOp_ResponseRange{}}
		case *pb.RequestOp_RequestPut:
			resps[i] = &pb.ResponseOp{Response: &pb.ResponseOp_ResponsePut{}}
		case *pb.RequestOp_RequestDeleteRange:
			resps[i] = &pb.ResponseOp{Response: &pb.ResponseOp_ResponseDeleteRange{}}
		case *pb.RequestOp_RequestTxn:
			resp, txns := newTxnResp(tv.RequestTxn, txnPath[1:])
			resps[i] = &pb.ResponseOp{Response: &pb.ResponseOp_ResponseTxn{ResponseTxn: resp}}
			txnPath = txnPath[1+txns:]
			txnCount += txns + 1
		default:
		}
	}
	return txnResp, txnCount
}

func compareToPath(rv mvcc.IReadView, rt *pb.TxnRequest) []bool {
	txnPath := make([]bool, 1)
	ops := rt.Success
	if txnPath[0] = applyCompares(rv, rt.Compare); !txnPath[0] {
		ops = rt.Failure
	}
	for _, op := range ops {
		tv, ok := op.Request.(*pb.RequestOp_RequestTxn)
		if !ok || tv.RequestTxn == nil {
			continue
		}
		txnPath = append(txnPath, compareToPath(rv, tv.RequestTxn)...)
	}
	return txnPath
}

func applyCompares(rv mvcc.IReadView, cmps []*pb.Compare) bool {
	for _, c := range cmps {
		if !applyCompare(rv, c) {
			return false
		}
	}
	return true
}

// applyCompare applies the compare request.
// If the comparison succeeds, it returns true. Otherwise, returns false.
func applyCompare(rv mvcc.IReadView, c *pb.Compare) bool {
	// TODO: possible optimizations
	// * chunk reads for large ranges to conserve memory
	// * rewrite rules for common patterns:
	//	ex. "[a, b) createrev > 0" => "limit 1 /\ kvs > 0"
	// * caching
	rr, err := rv.Range(context.TODO(), c.Key, mkGteRange(c.RangeEnd), mvcc.RangeOptions{})
	if err != nil {
		return false
	}
	if len(rr.KVs) == 0 {
		if c.Target == pb.Compare_VALUE {
			// Always fail if comparing a value on a key/keys that doesn't exist;
			// nil == empty string in grpc; no way to represent missing value
			return false
		}
		return compareKV(c, pb.KeyValue{})
	}
	for _, kv := range rr.KVs {
		if !compareKV(c, kv) {
			return false
		}
	}
	return true
}

func compareKV(c *pb.Compare, ckv pb.KeyValue) bool {
	var result int
	rev := int64(0)
	switch c.Target {
	case pb.Compare_VALUE:
		v := []byte{}
		if tv, _ := c.TargetUnion.(*pb.Compare_Value); tv != nil {
			v = tv.Value
		}
		result = bytes.Compare(ckv.Value, v)
	case pb.Compare_CREATE:
		if tv, _ := c.TargetUnion.(*pb.Compare_CreateRevision); tv != nil {
			rev = tv.CreateRevision
		}
		result = compareInt64(ckv.CreateRevision, rev)
	case pb.Compare_MOD:
		if tv, _ := c.TargetUnion.(*pb.Compare_ModRevision); tv != nil {
			rev = tv.ModRevision
		}
		result = compareInt64(ckv.ModRevision, rev)
	case pb.Compare_VERSION:
		if tv, _ := c.TargetUnion.(*pb.Compare_Version); tv != nil {
			rev = tv.Version
		}
		result = compareInt64(ckv.Version, rev)
	}
	switch c.Result {
	case pb.Compare_EQUAL:
		return result == 0
	case pb.Compare_NOT_EQUAL:
		return result != 0
	case pb.Compare_GREATER:
		return result > 0
	case pb.Compare_LESS:
		return result < 0
	}
	return true
}

func (a *applierBackend) applyTxn(ctx context.Context, txn mvcc.ITxnWrite, rt *pb.TxnRequest, txnPath []bool, tresp *pb.TxnResponse) (txns int, err error) {
	trace := traceutil.Get(ctx)
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}

	for i, req := range reqs {
		respi := tresp.Responses[i].Response
		switch tv := req.Request.(type) {
		case *pb.RequestOp_RequestRange:
			trace.StartSubTrace(
				traceutil.Field{Key: "req_type", Value: "range"},
				traceutil.Field{Key: "range_begin", Value: string(tv.RequestRange.Key)},
				traceutil.Field{Key: "range_end", Value: string(tv.RequestRange.RangeEnd)})
			resp, err := a.Range(ctx, txn, tv.RequestRange)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed Range: %w", err)
			}
			respi.(*pb.ResponseOp_ResponseRange).ResponseRange = resp
			trace.StopSubTrace()
		case *pb.RequestOp_RequestPut:
			trace.StartSubTrace(
				traceutil.Field{Key: "req_type", Value: "put"},
				traceutil.Field{Key: "key", Value: string(tv.RequestPut.Key)},
				traceutil.Field{Key: "req_size", Value: tv.RequestPut.XSize()})
			resp, _, err := a.Put(ctx, txn, tv.RequestPut)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed Put: %w", err)
			}
			respi.(*pb.ResponseOp_ResponsePut).ResponsePut = resp
			trace.StopSubTrace()
		case *pb.RequestOp_RequestDeleteRange:
			resp, err := a.DeleteRange(txn, tv.RequestDeleteRange)
			if err != nil {
				return 0, fmt.Errorf("applyTxn: failed DeleteRange: %w", err)
			}
			respi.(*pb.ResponseOp_ResponseDeleteRange).ResponseDeleteRange = resp
		case *pb.RequestOp_RequestTxn:
			resp := respi.(*pb.ResponseOp_ResponseTxn).ResponseTxn
			applyTxns, err := a.applyTxn(ctx, txn, tv.RequestTxn, txnPath[1:], resp)
			if err != nil {
				// don't wrap the error. It's a recursive call and err should be already wrapped
				return 0, err
			}
			txns += applyTxns + 1
			txnPath = txnPath[applyTxns+1:]
		default:
			// empty union
		}
	}
	return txns, nil
}

func (a *applierBackend) Compaction(compaction *pb.CompactionRequest) (*pb.CompactionResponse, <-chan struct{}, *traceutil.Trace, error) {
	resp := &pb.CompactionResponse{}
	resp.Header = &pb.ResponseHeader{}
	trace := traceutil.New("compact",
		a.ra.lg,
		traceutil.Field{Key: "revision", Value: compaction.Revision},
	)

	ch, err := a.ra.KV().Compact(trace, compaction.Revision)
	if err != nil {
		return nil, ch, nil, err
	}
	// get the current revision. which key to get is not important.
	rr, _ := a.ra.KV().Range(context.TODO(), []byte("compaction"), nil, mvcc.RangeOptions{})
	resp.Header.Revision = rr.Rev
	return resp, ch, trace, err
}

func (a *applierBackend) LeaseGrant(lc *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error) {
	l, err := a.ra.lessor.Grant(lease.LeaseID(lc.ID), lc.TTL)
	resp := &pb.LeaseGrantResponse{}
	if err == nil {
		resp.ID = int64(l.ID)
		resp.TTL = l.TTL()
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) LeaseRevoke(lc *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error) {
	err := a.ra.lessor.Revoke(lease.LeaseID(lc.ID))
	return &pb.LeaseRevokeResponse{Header: newHeader(a.ra)}, err
}

func (a *applierBackend) LeaseCheckpoint(lc *pb.LeaseCheckpointRequest) (*pb.LeaseCheckpointResponse, error) {
	for _, c := range lc.Checkpoints {
		err := a.ra.lessor.Checkpoint(lease.LeaseID(c.ID), c.Remaining_TTL)
		if err != nil {
			return &pb.LeaseCheckpointResponse{Header: newHeader(a.ra)}, err
		}
	}
	return &pb.LeaseCheckpointResponse{Header: newHeader(a.ra)}, nil
}

func (a *applierBackend) AuthEnable() (*pb.AuthEnableResponse, error) {
	err := a.ra.AuthStore().AuthEnable()
	if err != nil {
		return nil, err
	}
	return &pb.AuthEnableResponse{Header: newHeader(a.ra)}, nil
}

func (a *applierBackend) AuthDisable() (*pb.AuthDisableResponse, error) {
	a.ra.AuthStore().AuthDisable()
	return &pb.AuthDisableResponse{Header: newHeader(a.ra)}, nil
}

func (a *applierBackend) AuthStatus() (*pb.AuthStatusResponse, error) {
	enabled := a.ra.AuthStore().IsAuthEnabled()
	authRevision := a.ra.AuthStore().Revision()
	return &pb.AuthStatusResponse{Header: newHeader(a.ra), Enabled: enabled, AuthRevision: authRevision}, nil
}

func (a *applierBackend) Authenticate(r *pb.InternalAuthenticateRequest) (*pb.AuthenticateResponse, error) {
	index := a.ra.consistIndex.ConsistentIndex()
	ctx := context.WithValue(context.WithValue(a.ra.ctx, auth.AuthenticateParamIndex{}, index), auth.AuthenticateParamSimpleTokenPrefix{}, r.SimpleToken)
	resp, err := a.ra.AuthStore().Authenticate(ctx, r.Name, r.Password)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserAdd(r *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error) {
	resp, err := a.ra.AuthStore().UserAdd(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserDelete(r *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error) {
	resp, err := a.ra.AuthStore().UserDelete(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserChangePassword(r *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error) {
	resp, err := a.ra.AuthStore().UserChangePassword(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserGrantRole(r *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error) {
	resp, err := a.ra.AuthStore().UserGrantRole(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserGet(r *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	resp, err := a.ra.AuthStore().UserGet(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserRevokeRole(r *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error) {
	resp, err := a.ra.AuthStore().UserRevokeRole(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleAdd(r *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error) {
	resp, err := a.ra.AuthStore().RoleAdd(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleGrantPermission(r *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error) {
	resp, err := a.ra.AuthStore().RoleGrantPermission(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleGet(r *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	resp, err := a.ra.AuthStore().RoleGet(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleRevokePermission(r *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error) {
	resp, err := a.ra.AuthStore().RoleRevokePermission(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleDelete(r *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error) {
	resp, err := a.ra.AuthStore().RoleDelete(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) UserList(r *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error) {
	resp, err := a.ra.AuthStore().UserList(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

func (a *applierBackend) RoleList(r *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error) {
	resp, err := a.ra.AuthStore().RoleList(r)
	if resp != nil {
		resp.Header = newHeader(a.ra)
	}
	return resp, err
}

type kvSort struct{ kvs []pb.KeyValue }

func (s *kvSort) Swap(i, j int) {
	t := s.kvs[i]
	s.kvs[i] = s.kvs[j]
	s.kvs[j] = t
}
func (s *kvSort) Len() int { return len(s.kvs) }

type kvSortByKey struct{ *kvSort }

func (s *kvSortByKey) Less(i, j int) bool {
	return bytes.Compare(s.kvs[i].Key, s.kvs[j].Key) < 0
}

type kvSortByVersion struct{ *kvSort }

func (s *kvSortByVersion) Less(i, j int) bool {
	return (s.kvs[i].Version - s.kvs[j].Version) < 0
}

type kvSortByCreate struct{ *kvSort }

func (s *kvSortByCreate) Less(i, j int) bool {
	return (s.kvs[i].CreateRevision - s.kvs[j].CreateRevision) < 0
}

type kvSortByMod struct{ *kvSort }

func (s *kvSortByMod) Less(i, j int) bool {
	return (s.kvs[i].ModRevision - s.kvs[j].ModRevision) < 0
}

type kvSortByValue struct{ *kvSort }

func (s *kvSortByValue) Less(i, j int) bool {
	return bytes.Compare(s.kvs[i].Value, s.kvs[j].Value) < 0
}

func checkRequests(rv mvcc.IReadView, rt *pb.TxnRequest, txnPath []bool, f checkReqFunc) (int, error) {
	txnCount := 0
	reqs := rt.Success
	if !txnPath[0] {
		reqs = rt.Failure
	}
	for _, req := range reqs {
		if tv, ok := req.Request.(*pb.RequestOp_RequestTxn); ok && tv.RequestTxn != nil {
			txns, err := checkRequests(rv, tv.RequestTxn, txnPath[1:], f)
			if err != nil {
				return 0, err
			}
			txnCount += txns + 1
			txnPath = txnPath[txns+1:]
			continue
		}
		if err := f(rv, req); err != nil {
			return 0, err
		}
	}
	return txnCount, nil
}

func (a *applierBackend) checkRequestPut(rv mvcc.IReadView, reqOp *pb.RequestOp) error {
	tv, ok := reqOp.Request.(*pb.RequestOp_RequestPut)
	if !ok || tv.RequestPut == nil {
		return nil
	}
	req := tv.RequestPut
	if req.IgnoreValue {
		// expects previous key-value, error if not exist
		rr, err := rv.Range(context.TODO(), req.Key, nil, mvcc.RangeOptions{})
		if err != nil {
			return err
		}
		if rr == nil || len(rr.KVs) == 0 {
			return ErrKeyNotFound
		}
	}
	if lease.LeaseID(req.Lease) != lease.NoLease {
		if l := a.ra.lessor.Lookup(lease.LeaseID(req.Lease)); l == nil {
			return lease.ErrLeaseNotFound
		}
	}
	return nil
}

func (a *applierBackend) checkRequestRange(rv mvcc.IReadView, reqOp *pb.RequestOp) error {
	tv, ok := reqOp.Request.(*pb.RequestOp_RequestRange)
	if !ok || tv.RequestRange == nil {
		return nil
	}
	req := tv.RequestRange
	switch {
	case req.Revision == 0:
		return nil
	case req.Revision > rv.Rev():
		return mvcc.ErrFutureRev
	case req.Revision < rv.FirstRev():
		return mvcc.ErrCompacted
	}
	return nil
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// mkGteRange determines if the range end is a >= range. This works around grpc
// sending empty byte strings as nil; >= is encoded in the range end as '\0'.
// If it is a GTE range, then []byte{} is returned to indicate the empty byte
// string (vs nil being no byte string).
func mkGteRange(rangeEnd []byte) []byte {
	if len(rangeEnd) == 1 && rangeEnd[0] == 0 {
		return []byte{}
	}
	return rangeEnd
}

func noSideEffect(r *pb.InternalRaftRequest) bool {
	return r.Range != nil
}

func pruneKVs(rr *mvcc.RangeResult, isPrunable func(*pb.KeyValue) bool) {
	j := 0
	for i := range rr.KVs {
		rr.KVs[j] = rr.KVs[i]
		if !isPrunable(&rr.KVs[i]) {
			j++
		}
	}
	rr.KVs = rr.KVs[:j]
}

func newHeader(ra *Replica) *pb.ResponseHeader {
	return &pb.ResponseHeader{
		ShardId:  ra.ShardID(),
		NodeId:   ra.NodeID(),
		Revision: ra.KV().Rev(),
		RaftTerm: ra.Term(),
	}
}
