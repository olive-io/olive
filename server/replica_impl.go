package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/lni/dragonboat/v4"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server/auth"
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/mvcc"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	// In the health case, there might be a raall gap (10s of entries) between
	// the applied index and committed index.
	// However, if the committed entries are very heavy to apply, the gap might grow.
	// We should stop accepting new proposals if the gap growing to a certain point.
	maxGapBetweenApplyAndCommitIndex = 5000
	traceThreshold                   = 100 * time.Millisecond

	queryTimeout = time.Second * 3

	// The timeout for the node to catch up its applied index, and is used in
	// lease related operations, such as LeaseRenew and LeaseTimeToLive.
	applyTimeout = time.Second
)

type IRaftKV interface {
	Range(ctx context.Context, r *pb.RangeRequest) (*pb.RangeResponse, error)
	Put(ctx context.Context, r *pb.PutRequest) (*pb.PutResponse, error)
	DeleteRange(ctx context.Context, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error)
	Txn(ctx context.Context, r *pb.TxnRequest) (*pb.TxnResponse, error)
	Compact(ctx context.Context, r *pb.CompactionRequest) (*pb.CompactionResponse, error)
}

type ILessor interface {
	// LeaseGrant sends LeaseGrant request to raft and apply it after committed.
	LeaseGrant(ctx context.Context, r *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error)
	// LeaseRevoke sends LeaseRevoke request to raft and apply it after committed.
	LeaseRevoke(ctx context.Context, r *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error)

	// LeaseRenew renews the lease with given ID. The renewed TTL is returned. Or an error
	// is returned.
	LeaseRenew(ctx context.Context, id lease.LeaseID) (int64, error)

	// LeaseTimeToLive retrieves lease information.
	LeaseTimeToLive(ctx context.Context, r *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error)

	// LeaseLeases lists all leases.
	LeaseLeases(ctx context.Context, r *pb.LeaseLeasesRequest) (*pb.LeaseLeasesResponse, error)
}

type Authenticator interface {
	AuthEnable(ctx context.Context, r *pb.AuthEnableRequest) (*pb.AuthEnableResponse, error)
	AuthDisable(ctx context.Context, r *pb.AuthDisableRequest) (*pb.AuthDisableResponse, error)
	AuthStatus(ctx context.Context, r *pb.AuthStatusRequest) (*pb.AuthStatusResponse, error)
	Authenticate(ctx context.Context, r *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error)
	UserAdd(ctx context.Context, r *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error)
	UserDelete(ctx context.Context, r *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error)
	UserChangePassword(ctx context.Context, r *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error)
	UserGrantRole(ctx context.Context, r *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error)
	UserGet(ctx context.Context, r *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error)
	UserRevokeRole(ctx context.Context, r *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error)
	RoleAdd(ctx context.Context, r *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error)
	RoleGrantPermission(ctx context.Context, r *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error)
	RoleGet(ctx context.Context, r *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error)
	RoleRevokePermission(ctx context.Context, r *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error)
	RoleDelete(ctx context.Context, r *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error)
	UserList(ctx context.Context, r *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error)
	RoleList(ctx context.Context, r *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error)
}

func (ra *Replica) Range(ctx context.Context, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	trace := traceutil.New("range",
		ra.lg,
		traceutil.Field{Key: "range_begin", Value: string(r.Key)},
		traceutil.Field{Key: "range_end", Value: string(r.RangeEnd)},
		traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", ra.ShardID())},
	)
	ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

	var resp *pb.RangeResponse
	var err error
	defer func(start time.Time) {
		warnOfExpensiveReadOnlyRangeRequest(ra.lg, ra.WarningApplyDuration, start, r, resp, err)
		if resp != nil {
			trace.AddField(
				traceutil.Field{Key: "response_count", Value: len(resp.Kvs)},
				traceutil.Field{Key: "response_revision", Value: resp.Header.Revision},
				traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", ra.ShardID())},
			)
		}
		trace.LogIfLong(traceThreshold)
	}(time.Now())

	var req *replicaRequest
	rc := make(chan any, 1)
	ec := make(chan error, 1)

	if !r.Serializable {
		req = &replicaRequest{
			shardID: ra.ShardID(),
			nodeID:  ra.NodeID(),
			staleRead: &replicaStaleRead{
				query: r,
				rc:    rc,
				ec:    ec,
			},
		}
		trace.Step("agreement among raft nodes before linearized reading")
	} else {
		tctx, cancel := context.WithTimeout(ctx, queryTimeout)
		defer cancel()
		req = &replicaRequest{
			shardID: ra.ShardID(),
			nodeID:  ra.NodeID(),
			syncRead: &replicaSyncRead{
				ctx:   tctx,
				query: r,
				rc:    rc,
				ec:    ec,
			},
		}
	}

	chk := func(ai *auth.AuthInfo) error {
		return ra.AuthStore().IsRangePermitted(ai, r.Key, r.RangeEnd)
	}

	get := func() {
		ra.ReplicaRequest(req)
		select {
		case result := <-rc:
			var ok bool
			resp, ok = result.(*pb.RangeResponse)
			if !ok {
				ra.lg.Panic("not match raft read", zap.Stringer("request", r))
			}
		case err = <-ec:
		}
	}

	if serr := doSerialize(ctx, ra, chk, get); serr != nil {
		err = serr
		return nil, err
	}
	return resp, err
}

func (ra *Replica) Put(ctx context.Context, r *pb.PutRequest) (*pb.PutResponse, error) {
	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{Put: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.PutResponse), nil
}

func (ra *Replica) DeleteRange(ctx context.Context, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{DeleteRange: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.DeleteRangeResponse), nil
}

func (ra *Replica) Txn(ctx context.Context, r *pb.TxnRequest) (*pb.TxnResponse, error) {
	if isTxnReadonly(r) {
		trace := traceutil.New("transaction",
			ra.lg,
			traceutil.Field{Key: "read_only", Value: true},
			traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", ra.ShardID())},
		)
		ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

		var get func()
		var resp *pb.TxnResponse
		var err error

		var req *replicaRequest
		rc := make(chan any, 1)
		ec := make(chan error, 1)

		defer func() {
			close(rc)
			close(ec)
		}()

		if !isTxnSerializable(r) {
			get = func() {
				req = &replicaRequest{
					shardID: ra.ShardID(),
					nodeID:  ra.NodeID(),
					staleRead: &replicaStaleRead{
						query: r,
						rc:    rc,
						ec:    ec,
					},
				}
			}
			trace.Step("agreement among raft nodes before linearized reading")
		} else {
			tctx, cancel := context.WithTimeout(ctx, queryTimeout)
			defer cancel()
			req = &replicaRequest{
				shardID: ra.ShardID(),
				nodeID:  ra.NodeID(),
				syncRead: &replicaSyncRead{
					ctx:   tctx,
					query: r,
					rc:    rc,
					ec:    ec,
				},
			}
		}
		chk := func(ai *auth.AuthInfo) error {
			return checkTxnAuth(ra.authStore, ai, r)
		}

		get = func() {
			ra.ReplicaRequest(req)
			select {
			case result := <-rc:
				var ok bool
				resp, ok = result.(*pb.TxnResponse)
				if !ok {
					ra.lg.Panic("not match raft read", zap.Stringer("request", r))
				}
			case err = <-ec:
			}
		}

		defer func(start time.Time) {
			warnOfExpensiveReadOnlyTxnRequest(ra.lg, ra.WarningApplyDuration, start, r, resp, err)
			trace.LogIfLong(traceThreshold)
		}(time.Now())

		if serr := doSerialize(ctx, ra, chk, get); serr != nil {
			return nil, serr
		}
		return resp, err
	}

	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{Txn: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.TxnResponse), nil
}

func isTxnSerializable(rt *pb.TxnRequest) bool {
	for _, u := range rt.Success {
		if r := u.GetRequestRange(); r == nil || !r.Serializable {
			return false
		}
	}
	for _, u := range rt.Failure {
		if r := u.GetRequestRange(); r == nil || !r.Serializable {
			return false
		}
	}
	return true
}

func isTxnReadonly(tr *pb.TxnRequest) bool {
	for _, u := range tr.Success {
		if r := u.GetRequestRange(); r == nil {
			return false
		}
	}
	for _, u := range tr.Failure {
		if r := u.GetRequestRange(); r == nil {
			return false
		}
	}
	return true
}

func (ra *Replica) Compact(ctx context.Context, r *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	startTime := time.Now()
	result, err := ra.processInternalRaftRequestOnce(ctx, pb.InternalRaftRequest{Compaction: r})
	trace := traceutil.TODO()
	if result != nil && result.trace != nil {
		trace = result.trace
		defer func() {
			trace.LogIfLong(traceThreshold)
		}()
		applyStart := result.trace.GetStartTime()
		result.trace.SetStartTime(startTime)
		trace.InsertStep(0, applyStart, "process raft request")
	}
	if r.Physical && result != nil && result.physc != nil {
		<-result.physc
		// The compaction is done deleting keys; the hash is now settled
		// but the data is not necessarily committed. If there's a crash,
		// the hash may revert to a hash prior to compaction completing
		// if the compaction resumes. Force the finished compaction to
		// commit, so it won't resume following a crash.
		ra.Backend().ForceCommit()
		trace.Step("physically apply compaction")
	}
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	resp := result.resp.(*pb.CompactionResponse)
	if resp == nil {
		resp = &pb.CompactionResponse{}
	}
	if resp.Header == nil {
		resp.Header = &pb.ResponseHeader{}
	}
	resp.Header.Revision = ra.KV().Rev()
	trace.AddField(traceutil.Field{Key: "response_revision", Value: resp.Header.Revision})
	trace.AddField(traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", ra.ShardID())})
	return resp, nil
}

func (ra *Replica) LeaseGrant(ctx context.Context, r *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error) {
	// no id given? choose one
	for r.ID == int64(lease.NoLease) {
		// only use positive int64 id's
		r.ID = int64(ra.reqIDGen.Next() & ((1 << 63) - 1))
	}

	resp, err := ra.raftRequestOnce(ctx, pb.InternalRaftRequest{LeaseGrant: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.LeaseGrantResponse), nil
}

func (ra *Replica) waitAppliedIndex() error {
	select {
	case <-ra.ApplyWait():
	case <-ra.stopping:
		return ErrStopped
	case <-time.After(applyTimeout):
		return ErrTimeoutWaitAppliedIndex
	}

	return nil
}

func (ra *Replica) LeaseRevoke(ctx context.Context, r *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error) {
	resp, err := ra.raftRequestOnce(ctx, pb.InternalRaftRequest{LeaseRevoke: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.LeaseRevokeResponse), nil
}

func (ra *Replica) LeaseRenew(ctx context.Context, id lease.LeaseID) (int64, error) {
	if ra.isLeader() {
		if err := ra.waitAppliedIndex(); err != nil {
			return 0, err
		}

		ttl, err := ra.lessor.Renew(id)
		if err == nil { // already requested to primary lessor(leader)
			return ttl, nil
		}
		if !errors.Is(err, lease.ErrNotPrimary) {
			return -1, err
		}
	}

	cctx, cancel := context.WithTimeout(ctx, ra.ReqTimeout())
	defer cancel()

	// renewals don't go through raft; forward to leader manually
	for cctx.Err() == nil {
		//leader, lerr := s.waitLeader(cctx)
		//if lerr != nil {
		//	return -1, lerr
		//}
		//for _, url := range leader.PeerURLs {
		//	lurl := url + leasehttp.LeasePrefix
		//	ttl, err := leasehttp.RenewHTTP(cctx, id, lurl, s.peerRt)
		//	if err == nil || err == lease.ErrLeaseNotFound {
		//		return ttl, err
		//	}
		//}
		// Throttle in case of e.g. connection problems.
		time.Sleep(50 * time.Millisecond)
	}

	if errors.Is(cctx.Err(), context.DeadlineExceeded) {
		return -1, ErrTimeout
	}
	return -1, ErrCanceled
}

func (ra *Replica) checkLeaseTimeToLive(ctx context.Context, leaseID lease.LeaseID) (uint64, error) {
	rev := ra.AuthStore().Revision()
	if !ra.AuthStore().IsAuthEnabled() {
		return rev, nil
	}
	authInfo, err := ra.AuthInfoFromCtx(ctx)
	if err != nil {
		return rev, err
	}
	if authInfo == nil {
		return rev, auth.ErrUserEmpty
	}

	l := ra.lessor.Lookup(leaseID)
	if l != nil {
		for _, key := range l.Keys() {
			if err := ra.AuthStore().IsRangePermitted(authInfo, []byte(key), []byte{}); err != nil {
				return 0, err
			}
		}
	}

	return rev, nil
}

func (ra *Replica) leaseTimeToLive(ctx context.Context, r *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error) {
	//if s.isLeader() {
	//	if err := s.waitAppliedIndex(); err != nil {
	//		return nil, err
	//	}
	//	// primary; timetolive directly from leader
	//	le := s.lessor.Lookup(lease.LeaseID(r.ID))
	//	if le == nil {
	//		return nil, lease.ErrLeaseNotFound
	//	}
	//	// TODO: fill out ResponseHeader
	//	resp := &pb.LeaseTimeToLiveResponse{Header: &pb.ResponseHeader{}, ID: r.ID, TTL: int64(le.Remaining().Seconds()), GrantedTTL: le.TTL()}
	//	if r.Keys {
	//		ks := le.Keys()
	//		kbs := make([][]byte, len(ks))
	//		for i := range ks {
	//			kbs[i] = []byte(ks[i])
	//		}
	//		resp.Keys = kbs
	//	}
	//	return resp, nil
	//}
	//
	//cctx, cancel := context.WithTimeout(ctx, s.Cfg.ReqTimeout())
	//defer cancel()
	//
	//// forward to leader
	//for cctx.Err() == nil {
	//	leader, err := s.waitLeader(cctx)
	//	if err != nil {
	//		return nil, err
	//	}
	//	for _, url := range leader.PeerURLs {
	//		lurl := url + leasehttp.LeaseInternalPrefix
	//		resp, err := leasehttp.TimeToLiveHTTP(cctx, lease.LeaseID(r.ID), r.Keys, lurl, s.peerRt)
	//		if err == nil {
	//			return resp.LeaseTimeToLiveResponse, nil
	//		}
	//		if err == lease.ErrLeaseNotFound {
	//			return nil, err
	//		}
	//	}
	//}
	//
	//if cctx.Err() == context.DeadlineExceeded {
	//	return nil, ErrTimeout
	//}
	return nil, ErrCanceled
}

func (ra *Replica) LeaseTimeToLive(ctx context.Context, r *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error) {
	var rev uint64
	var err error
	if r.Keys {
		// check RBAC permission only if Keys is true
		rev, err = ra.checkLeaseTimeToLive(ctx, lease.LeaseID(r.ID))
		if err != nil {
			return nil, err
		}
	}

	resp, err := ra.leaseTimeToLive(ctx, r)
	if err != nil {
		return nil, err
	}

	if r.Keys {
		if ra.AuthStore().IsAuthEnabled() && rev != ra.AuthStore().Revision() {
			return nil, auth.ErrAuthOldRevision
		}
	}
	return resp, nil
}

// LeaseLeases is really ListLeases !???
func (ra *Replica) LeaseLeases(ctx context.Context, r *pb.LeaseLeasesRequest) (*pb.LeaseLeasesResponse, error) {
	ls := ra.lessor.Leases()
	lss := make([]*pb.LeaseStatus, len(ls))
	for i := range ls {
		lss[i] = &pb.LeaseStatus{ID: int64(ls[i].ID)}
	}
	return &pb.LeaseLeasesResponse{Header: newHeader(ra), Leases: lss}, nil
}

func (ra *Replica) AuthEnable(ctx context.Context, r *pb.AuthEnableRequest) (*pb.AuthEnableResponse, error) {
	resp, err := ra.raftRequestOnce(ctx, pb.InternalRaftRequest{AuthEnable: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthEnableResponse), nil
}

func (ra *Replica) AuthDisable(ctx context.Context, r *pb.AuthDisableRequest) (*pb.AuthDisableResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthDisable: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthDisableResponse), nil
}

func (ra *Replica) AuthStatus(ctx context.Context, r *pb.AuthStatusRequest) (*pb.AuthStatusResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthStatus: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthStatusResponse), nil
}

func (ra *Replica) Authenticate(ctx context.Context, r *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	//if err := s.linearizableReadNotify(ctx); err != nil {
	//	return nil, err
	//}
	lg := ra.lg

	// fix https://nvd.nist.gov/vuln/detail/CVE-2021-28235
	defer func() {
		if r != nil {
			r.Password = ""
		}
	}()

	var resp proto.Message
	for {
		checkedRevision, err := ra.AuthStore().CheckPassword(r.Name, r.Password)
		if err != nil {
			if !errors.Is(err, auth.ErrAuthNotEnabled) {
				lg.Warn(
					"invalid authentication was requested",
					zap.String("user", r.Name),
					zap.Error(err),
				)
			}
			return nil, err
		}

		st, err := ra.AuthStore().GenTokenPrefix()
		if err != nil {
			return nil, err
		}

		// internalReq doesn't need to have Password because the above s.AuthStore().CheckPassword() already did it.
		// In addition, it will let a WAL entry not record password as a plain text.
		internalReq := &pb.InternalAuthenticateRequest{
			Name:        r.Name,
			SimpleToken: st,
		}

		resp, err = ra.raftRequestOnce(ctx, pb.InternalRaftRequest{Authenticate: internalReq})
		if err != nil {
			return nil, err
		}
		if checkedRevision == ra.AuthStore().Revision() {
			break
		}

		lg.Info("revision when password checked became stale; retrying")
	}

	return resp.(*pb.AuthenticateResponse), nil
}

func (ra *Replica) UserAdd(ctx context.Context, r *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error) {
	if r.Options == nil || !r.Options.NoPassword {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), ra.authStore.BcryptCost())
		if err != nil {
			return nil, err
		}
		r.HashedPassword = base64.StdEncoding.EncodeToString(hashedPassword)
		r.Password = ""
	}

	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserAdd: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserAddResponse), nil
}

func (ra *Replica) UserDelete(ctx context.Context, r *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserDelete: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserDeleteResponse), nil
}

func (ra *Replica) UserChangePassword(ctx context.Context, r *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error) {
	if r.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), ra.authStore.BcryptCost())
		if err != nil {
			return nil, err
		}
		r.HashedPassword = base64.StdEncoding.EncodeToString(hashedPassword)
		r.Password = ""
	}

	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserChangePassword: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserChangePasswordResponse), nil
}

func (ra *Replica) UserGrantRole(ctx context.Context, r *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserGrantRole: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserGrantRoleResponse), nil
}

func (ra *Replica) UserGet(ctx context.Context, r *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserGet: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserGetResponse), nil
}

func (ra *Replica) UserList(ctx context.Context, r *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserList: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserListResponse), nil
}

func (ra *Replica) UserRevokeRole(ctx context.Context, r *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthUserRevokeRole: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserRevokeRoleResponse), nil
}

func (ra *Replica) RoleAdd(ctx context.Context, r *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleAdd: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleAddResponse), nil
}

func (ra *Replica) RoleGrantPermission(ctx context.Context, r *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleGrantPermission: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleGrantPermissionResponse), nil
}

func (ra *Replica) RoleGet(ctx context.Context, r *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleGet: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleGetResponse), nil
}

func (ra *Replica) RoleList(ctx context.Context, r *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleList: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleListResponse), nil
}

func (ra *Replica) RoleRevokePermission(ctx context.Context, r *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleRevokePermission: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleRevokePermissionResponse), nil
}

func (ra *Replica) RoleDelete(ctx context.Context, r *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error) {
	resp, err := ra.raftRequest(ctx, pb.InternalRaftRequest{AuthRoleDelete: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleDeleteResponse), nil
}

func (ra *Replica) raftRequestOnce(ctx context.Context, r pb.InternalRaftRequest) (proto.Message, error) {
	result, err := ra.processInternalRaftRequestOnce(ctx, r)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	if startTime, ok := ctx.Value(traceutil.StartTimeKey).(time.Time); ok && result.trace != nil {
		applyStart := result.trace.GetStartTime()
		// The trace object is created in apply. Here reset the start time to trace
		// the raft request time by the difference between the request start time
		// and apply start time
		result.trace.SetStartTime(startTime)
		result.trace.InsertStep(0, applyStart, "process raft request")
		result.trace.LogIfLong(traceThreshold)
	}
	return result.resp, nil
}

func (ra *Replica) raftRequest(ctx context.Context, r pb.InternalRaftRequest) (proto.Message, error) {
	return ra.raftRequestOnce(ctx, r)
}

// doSerialize handles the auth logic, with permissions checked by "chk", for a serialized request "get". Returns a non-nil error on authentication failure.
func doSerialize(ctx context.Context, ra *Replica, chk func(*auth.AuthInfo) error, get func()) error {
	trace := traceutil.Get(ctx)
	ai, err := ra.AuthInfoFromCtx(ctx)
	if err != nil {
		return err
	}
	if ai == nil {
		// chk expects non-nil AuthInfo; use empty credentials
		ai = &auth.AuthInfo{}
	}
	if err = chk(ai); err != nil {
		return err
	}
	trace.Step("get authentication metadata")
	// fetch response for serialized request
	get()
	// check for stale token revision in case the auth store was updated while
	// the request has been handled.
	if ai.Revision != 0 && ai.Revision != ra.authStore.Revision() {
		return auth.ErrAuthOldRevision
	}
	return nil
}

func (ra *Replica) processInternalRaftRequestOnce(ctx context.Context, r pb.InternalRaftRequest) (*applyResult, error) {
	ai := ra.getAppliedIndex()
	ci := ra.getCommittedIndex()
	if ci > ai+maxGapBetweenApplyAndCommitIndex {
		return nil, ErrTooManyRequests
	}

	r.Header = &pb.RequestHeader{
		ID: ra.reqIDGen.Next(),
	}

	// check authinfo if it is not InternalAuthenticateRequest
	if r.Authenticate == nil {
		authInfo, err := ra.AuthInfoFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		if authInfo != nil {
			r.Header.Username = authInfo.Username
			r.Header.AuthRevision = authInfo.Revision
		}
	}

	data, err := r.Marshal()
	if err != nil {
		return nil, err
	}

	if len(data) > int(ra.MaxRequestBytes) {
		return nil, ErrRequestTooLarge
	}

	id := r.Header.ID
	ch := ra.w.Register(id)

	cctx, cancel := context.WithTimeout(ctx, ra.ReqTimeout())
	defer cancel()

	start := time.Now()

	rc := make(chan any, 1)
	ec := make(chan error, 1)

	req := &replicaRequest{
		shardID: ra.ShardID(),
		nodeID:  ra.NodeID(),
		syncPropose: &replicaSyncPropose{
			ctx:  cctx,
			data: data,
			rc:   rc,
			ec:   ec,
		},
	}

	ra.ReplicaRequest(req)
	select {
	case <-rc:
	case err = <-ec:
		if err != nil {
			proposalsFailed.Inc()
			ra.w.Trigger(id, nil) // GC wait
			return nil, err
		}
	}

	proposalsPending.Inc()
	defer proposalsPending.Dec()

	select {
	case x := <-ch:
		return x.(*applyResult), nil
	case <-cctx.Done():
		proposalsFailed.Inc()
		ra.w.Trigger(id, nil) // GC wait
		return nil, ra.parseProposeCtxErr(cctx.Err(), start)
	case <-ra.done:
		return nil, ErrStopped
	}
}

func (ra *Replica) ReplicaRequest(r *replicaRequest) {
	select {
	case <-ra.stopping:
	case ra.replicaRequestC <- r:
	}
}

func (ra *Replica) AuthInfoFromCtx(ctx context.Context) (*auth.AuthInfo, error) {
	authInfo, err := ra.AuthStore().AuthInfoFromCtx(ctx)
	if authInfo != nil || err != nil {
		return authInfo, err
	}
	if !ra.ClientCertAuthEnabled {
		return nil, nil
	}
	authInfo = ra.AuthStore().AuthInfoFromTLS(ctx)
	return authInfo, nil
}

func (ra *Replica) Watchable() mvcc.IWatchableKV { return ra.KV() }

func isStopped(err error) bool {
	return errors.Is(err, dragonboat.ErrClosed) || errors.Is(err, ErrStopped)
}
