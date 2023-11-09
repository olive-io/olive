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
	Range(ctx context.Context, shard uint64, r *pb.RangeRequest) (*pb.RangeResponse, error)
	Put(ctx context.Context, shard uint64, r *pb.PutRequest) (*pb.PutResponse, error)
	DeleteRange(ctx context.Context, shard uint64, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error)
	Txn(ctx context.Context, shard uint64, r *pb.TxnRequest) (*pb.TxnResponse, error)
	Compact(ctx context.Context, shard uint64, r *pb.CompactionRequest) (*pb.CompactionResponse, error)
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

func (s *KVServer) Range(ctx context.Context, shardID uint64, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	ra, exists := s.getReplica(shardID)
	if !exists {
		return nil, ErrShardNotFound
	}

	trace := traceutil.New("range",
		s.Logger(),
		traceutil.Field{Key: "range_begin", Value: string(r.Key)},
		traceutil.Field{Key: "range_end", Value: string(r.RangeEnd)},
		traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
	)
	ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

	var resp *pb.RangeResponse
	var err error
	defer func(start time.Time) {
		warnOfExpensiveReadOnlyRangeRequest(s.Logger(), s.WarningApplyDuration, start, r, resp, err)
		if resp != nil {
			trace.AddField(
				traceutil.Field{Key: "response_count", Value: len(resp.Kvs)},
				traceutil.Field{Key: "response_revision", Value: resp.Header.Revision},
				traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
			)
		}
		trace.LogIfLong(traceThreshold)
	}(time.Now())

	var get func()

	if !r.Serializable {
		get = func() {
			var result any
			var ok bool
			result, err = s.nh.StaleRead(shardID, r)
			if err != nil {
				return
			}
			resp, ok = result.(*pb.RangeResponse)
			if !ok {
				s.Logger().Panic("not match raft read", zap.Stringer("request", r))
			}
		}
		trace.Step("agreement among raft nodes before linearized reading")
	} else {
		get = func() {
			var result any
			var ok bool

			tctx, cancel := context.WithTimeout(ctx, queryTimeout)
			defer cancel()
			result, err = s.nh.SyncRead(tctx, shardID, r)
			if err != nil {
				return
			}
			resp, ok = result.(*pb.RangeResponse)
			if !ok {
				s.Logger().Panic("not match raft read", zap.Stringer("request", r))
			}
		}
	}
	chk := func(ai *auth.AuthInfo) error {
		return ra.AuthStore().IsRangePermitted(ai, r.Key, r.RangeEnd)
	}

	if serr := doSerialize(ctx, ra, chk, get); serr != nil {
		err = serr
		return nil, err
	}
	return resp, err
}

func (s *KVServer) Put(ctx context.Context, shardID uint64, r *pb.PutRequest) (*pb.PutResponse, error) {
	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{Put: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.PutResponse), nil
}

func (s *KVServer) DeleteRange(ctx context.Context, shardID uint64, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{DeleteRange: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.DeleteRangeResponse), nil
}

func (s *KVServer) Txn(ctx context.Context, shardID uint64, r *pb.TxnRequest) (*pb.TxnResponse, error) {
	ra, exists := s.getReplica(shardID)
	if !exists {
		return nil, ErrShardNotFound
	}

	if isTxnReadonly(r) {
		trace := traceutil.New("transaction",
			s.Logger(),
			traceutil.Field{Key: "read_only", Value: true},
			traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)},
		)
		ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

		var get func()
		var resp *pb.TxnResponse
		var err error

		if !isTxnSerializable(r) {
			get = func() {
				var result any
				var ok bool
				result, err = s.nh.StaleRead(shardID, r)
				if err != nil {
					return
				}
				resp, ok = result.(*pb.TxnResponse)
				if !ok {
					s.Logger().Panic("not match raft read", zap.Stringer("request", r))
				}
			}
			trace.Step("agreement among raft nodes before linearized reading")
		} else {
			get = func() {
				var result any
				var ok bool

				tctx, cancel := context.WithTimeout(ctx, queryTimeout)
				defer cancel()
				result, err = s.nh.SyncRead(tctx, shardID, r)
				if err != nil {
					return
				}
				resp, ok = result.(*pb.TxnResponse)
				if !ok {
					s.Logger().Panic("not match raft read", zap.Stringer("request", r))
				}
			}
		}
		chk := func(ai *auth.AuthInfo) error {
			return checkTxnAuth(ra.authStore, ai, r)
		}

		defer func(start time.Time) {
			warnOfExpensiveReadOnlyTxnRequest(s.Logger(), s.WarningApplyDuration, start, r, resp, err)
			trace.LogIfLong(traceThreshold)
		}(time.Now())

		if serr := doSerialize(ctx, ra, chk, get); serr != nil {
			return nil, serr
		}
		return resp, err
	}

	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{Txn: r})
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

func (s *KVServer) Compact(ctx context.Context, shardID uint64, r *pb.CompactionRequest) (*pb.CompactionResponse, error) {
	ra, exists := s.getReplica(shardID)
	if !exists {
		return nil, ErrShardNotFound
	}

	startTime := time.Now()
	result, err := s.processInternalRaftRequestOnce(ctx, shardID, pb.InternalRaftRequest{Compaction: r})
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
	trace.AddField(traceutil.Field{Key: "shard", Value: fmt.Sprintf("%d", shardID)})
	return resp, nil
}

func (s *KVServer) LeaseGrant(ctx context.Context, r *pb.LeaseGrantRequest) (*pb.LeaseGrantResponse, error) {
	// no id given? choose one
	//for r.ID == int64(lease.NoLease) {
	//	// only use positive int64 id's
	//	r.ID = int64(s.reqIDGen.Next() & ((1 << 63) - 1))
	//}

	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}

	resp, err := s.raftRequestOnce(ctx, shardID, pb.InternalRaftRequest{LeaseGrant: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.LeaseGrantResponse), nil
}

func (s *KVServer) waitAppliedIndex() error {
	select {
	//case <-s.ApplyWait():
	case <-s.stopping:
		return ErrStopped
	case <-time.After(applyTimeout):
		return ErrTimeoutWaitAppliedIndex
	}

	return nil
}

func (s *KVServer) LeaseRevoke(ctx context.Context, r *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequestOnce(ctx, shardID, pb.InternalRaftRequest{LeaseRevoke: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.LeaseRevokeResponse), nil
}

func (s *KVServer) LeaseRenew(ctx context.Context, id lease.LeaseID) (int64, error) {
	//if s.isLeader() {
	//	if err := s.waitAppliedIndex(); err != nil {
	//		return 0, err
	//	}
	//
	//	ttl, err := s.lessor.Renew(id)
	//	if err == nil { // already requested to primary lessor(leader)
	//		return ttl, nil
	//	}
	//	if err != lease.ErrNotPrimary {
	//		return -1, err
	//	}
	//}
	//
	//cctx, cancel := context.WithTimeout(ctx, s.Cfg.ReqTimeout())
	//defer cancel()
	//
	//// renewals don't go through raft; forward to leader manually
	//for cctx.Err() == nil {
	//	leader, lerr := s.waitLeader(cctx)
	//	if lerr != nil {
	//		return -1, lerr
	//	}
	//	for _, url := range leader.PeerURLs {
	//		lurl := url + leasehttp.LeasePrefix
	//		ttl, err := leasehttp.RenewHTTP(cctx, id, lurl, s.peerRt)
	//		if err == nil || err == lease.ErrLeaseNotFound {
	//			return ttl, err
	//		}
	//	}
	//	// Throttle in case of e.g. connection problems.
	//	time.Sleep(50 * time.Millisecond)
	//}
	//
	//if cctx.Err() == context.DeadlineExceeded {
	//	return -1, ErrTimeout
	//}
	return -1, ErrCanceled
}

func (s *KVServer) checkLeaseTimeToLive(ctx context.Context, leaseID lease.LeaseID) (uint64, error) {
	rev := uint64(0)
	//rev := s.AuthStore().Revision()
	//if !s.AuthStore().IsAuthEnabled() {
	//	return rev, nil
	//}
	//authInfo, err := s.AuthInfoFromCtx(ctx)
	//if err != nil {
	//	return rev, err
	//}
	//if authInfo == nil {
	//	return rev, auth.ErrUserEmpty
	//}
	//
	//l := s.lessor.Lookup(leaseID)
	//if l != nil {
	//	for _, key := range l.Keys() {
	//		if err := s.AuthStore().IsRangePermitted(authInfo, []byte(key), []byte{}); err != nil {
	//			return 0, err
	//		}
	//	}
	//}

	return rev, nil
}

func (s *KVServer) leaseTimeToLive(ctx context.Context, r *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error) {
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

func (s *KVServer) LeaseTimeToLive(ctx context.Context, r *pb.LeaseTimeToLiveRequest) (*pb.LeaseTimeToLiveResponse, error) {
	var rev uint64
	var err error
	if r.Keys {
		// check RBAC permission only if Keys is true
		rev, err = s.checkLeaseTimeToLive(ctx, lease.LeaseID(r.ID))
		if err != nil {
			return nil, err
		}
	}
	_ = rev

	resp, err := s.leaseTimeToLive(ctx, r)
	if err != nil {
		return nil, err
	}

	if r.Keys {
		//if s.AuthStore().IsAuthEnabled() && rev != s.AuthStore().Revision() {
		//	return nil, auth.ErrAuthOldRevision
		//}
	}
	return resp, nil
}

// LeaseLeases is really ListLeases !???
func (s *KVServer) LeaseLeases(ctx context.Context, r *pb.LeaseLeasesRequest) (*pb.LeaseLeasesResponse, error) {
	var ra *Replica
	ls := ra.lessor.Leases()
	lss := make([]*pb.LeaseStatus, len(ls))
	for i := range ls {
		lss[i] = &pb.LeaseStatus{ID: int64(ls[i].ID)}
	}
	return &pb.LeaseLeasesResponse{Header: newHeader(ra), Leases: lss}, nil
}

//func (s *KVServer) waitLeader(ctx context.Context) (*membership.Member, error) {
//	leader := s.cluster.Member(s.Leader())
//	for leader == nil {
//		// wait an election
//		dur := time.Duration(s.Cfg.ElectionTicks) * time.Duration(s.Cfg.TickMs) * time.Millisecond
//		select {
//		case <-time.After(dur):
//			leader = s.cluster.Member(s.Leader())
//		case <-s.stopping:
//			return nil, ErrStopped
//		case <-ctx.Done():
//			return nil, ErrNoLeader
//		}
//	}
//	if leader == nil || len(leader.PeerURLs) == 0 {
//		return nil, ErrNoLeader
//	}
//	return leader, nil
//}

func (s *KVServer) AuthEnable(ctx context.Context, r *pb.AuthEnableRequest) (*pb.AuthEnableResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequestOnce(ctx, shardID, pb.InternalRaftRequest{AuthEnable: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthEnableResponse), nil
}

func (s *KVServer) AuthDisable(ctx context.Context, r *pb.AuthDisableRequest) (*pb.AuthDisableResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthDisable: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthDisableResponse), nil
}

func (s *KVServer) AuthStatus(ctx context.Context, r *pb.AuthStatusRequest) (*pb.AuthStatusResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthStatus: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthStatusResponse), nil
}

func (s *KVServer) Authenticate(ctx context.Context, r *pb.AuthenticateRequest) (*pb.AuthenticateResponse, error) {
	//if err := s.linearizableReadNotify(ctx); err != nil {
	//	return nil, err
	//}
	var ra *Replica

	lg := s.Logger()

	// fix https://nvd.nist.gov/vuln/detail/CVE-2021-28235
	defer func() {
		if r != nil {
			r.Password = ""
		}
	}()

	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}

	var resp proto.Message
	for {
		checkedRevision, err := ra.AuthStore().CheckPassword(r.Name, r.Password)
		if err != nil {
			if err != auth.ErrAuthNotEnabled {
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

		resp, err = s.raftRequestOnce(ctx, shardID, pb.InternalRaftRequest{Authenticate: internalReq})
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

func (s *KVServer) UserAdd(ctx context.Context, r *pb.AuthUserAddRequest) (*pb.AuthUserAddResponse, error) {
	var ra *Replica
	if r.Options == nil || !r.Options.NoPassword {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), ra.authStore.BcryptCost())
		if err != nil {
			return nil, err
		}
		r.HashedPassword = base64.StdEncoding.EncodeToString(hashedPassword)
		r.Password = ""
	}

	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}

	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserAdd: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserAddResponse), nil
}

func (s *KVServer) UserDelete(ctx context.Context, r *pb.AuthUserDeleteRequest) (*pb.AuthUserDeleteResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserDelete: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserDeleteResponse), nil
}

func (s *KVServer) UserChangePassword(ctx context.Context, r *pb.AuthUserChangePasswordRequest) (*pb.AuthUserChangePasswordResponse, error) {
	var ra *Replica
	if r.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), ra.authStore.BcryptCost())
		if err != nil {
			return nil, err
		}
		r.HashedPassword = base64.StdEncoding.EncodeToString(hashedPassword)
		r.Password = ""
	}

	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}

	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserChangePassword: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserChangePasswordResponse), nil
}

func (s *KVServer) UserGrantRole(ctx context.Context, r *pb.AuthUserGrantRoleRequest) (*pb.AuthUserGrantRoleResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserGrantRole: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserGrantRoleResponse), nil
}

func (s *KVServer) UserGet(ctx context.Context, r *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserGet: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserGetResponse), nil
}

func (s *KVServer) UserList(ctx context.Context, r *pb.AuthUserListRequest) (*pb.AuthUserListResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserList: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserListResponse), nil
}

func (s *KVServer) UserRevokeRole(ctx context.Context, r *pb.AuthUserRevokeRoleRequest) (*pb.AuthUserRevokeRoleResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthUserRevokeRole: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthUserRevokeRoleResponse), nil
}

func (s *KVServer) RoleAdd(ctx context.Context, r *pb.AuthRoleAddRequest) (*pb.AuthRoleAddResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleAdd: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleAddResponse), nil
}

func (s *KVServer) RoleGrantPermission(ctx context.Context, r *pb.AuthRoleGrantPermissionRequest) (*pb.AuthRoleGrantPermissionResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleGrantPermission: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleGrantPermissionResponse), nil
}

func (s *KVServer) RoleGet(ctx context.Context, r *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleGet: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleGetResponse), nil
}

func (s *KVServer) RoleList(ctx context.Context, r *pb.AuthRoleListRequest) (*pb.AuthRoleListResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleList: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleListResponse), nil
}

func (s *KVServer) RoleRevokePermission(ctx context.Context, r *pb.AuthRoleRevokePermissionRequest) (*pb.AuthRoleRevokePermissionResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleRevokePermission: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleRevokePermissionResponse), nil
}

func (s *KVServer) RoleDelete(ctx context.Context, r *pb.AuthRoleDeleteRequest) (*pb.AuthRoleDeleteResponse, error) {
	shardID, ok := s.isReadyInternalReplica()
	if !ok {
		return nil, ErrNoInternalReplica
	}
	resp, err := s.raftRequest(ctx, shardID, pb.InternalRaftRequest{AuthRoleDelete: r})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.AuthRoleDeleteResponse), nil
}

func (s *KVServer) raftRequestOnce(ctx context.Context, shardID uint64, r pb.InternalRaftRequest) (proto.Message, error) {
	result, err := s.processInternalRaftRequestOnce(ctx, shardID, r)
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

func (s *KVServer) raftRequest(ctx context.Context, shardID uint64, r pb.InternalRaftRequest) (proto.Message, error) {
	return s.raftRequestOnce(ctx, shardID, r)
}

// doSerialize handles the auth logic, with permissions checked by "chk", for a serialized request "get". Returns a non-nil error on authentication failure.
func doSerialize(ctx context.Context, s *Replica, chk func(*auth.AuthInfo) error, get func()) error {
	trace := traceutil.Get(ctx)
	ai, err := s.AuthInfoFromCtx(ctx)
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
	if ai.Revision != 0 && ai.Revision != s.authStore.Revision() {
		return auth.ErrAuthOldRevision
	}
	return nil
}

func (s *KVServer) processInternalRaftRequestOnce(ctx context.Context, shardID uint64, r pb.InternalRaftRequest) (*applyResult, error) {
	sra, exists := s.getReplica(shardID)
	if !exists {
		return nil, ErrShardNotFound
	}

	ai := sra.getAppliedIndex()
	ci := sra.getCommittedIndex()
	if ci > ai+maxGapBetweenApplyAndCommitIndex {
		return nil, ErrTooManyRequests
	}

	r.Header = &pb.RequestHeader{
		ID: sra.reqIDGen.Next(),
	}

	// check authinfo if it is not InternalAuthenticateRequest
	if r.Authenticate == nil {
		authInfo, err := sra.AuthInfoFromCtx(ctx)
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

	if len(data) > int(s.MaxRequestBytes) {
		return nil, ErrRequestTooLarge
	}

	id := r.Header.ID
	ch := sra.w.Register(id)

	cctx, cancel := context.WithTimeout(ctx, s.ReqTimeout())
	defer cancel()

	start := time.Now()

	session := s.nh.GetNoOPSession(shardID)
	_, err = s.nh.SyncPropose(cctx, session, data)
	if err != nil {
		proposalsFailed.Inc()
		sra.w.Trigger(id, nil) // GC wait
		return nil, err
	}
	proposalsPending.Inc()
	defer proposalsPending.Dec()

	select {
	case x := <-ch:
		return x.(*applyResult), nil
	case <-cctx.Done():
		proposalsFailed.Inc()
		sra.w.Trigger(id, nil) // GC wait
		return nil, s.parseProposeCtxErr(cctx.Err(), start)
	case <-s.done:
		return nil, ErrStopped
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

func isStopped(err error) bool {
	return errors.Is(err, dragonboat.ErrClosed) || errors.Is(err, ErrStopped)
}

func (ra *Replica) Watchable() mvcc.IWatchableKV { return ra.KV() }
