package server

import (
	"context"
	"sync"

	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/server/auth"
	"github.com/olive-io/olive/server/lease"
	"github.com/olive-io/olive/server/mvcc"
	"go.etcd.io/etcd/pkg/v3/traceutil"
)

type authApplier struct {
	applier
	as     auth.AuthStore
	lessor lease.ILessor

	// mu serializes Apply so that user isn't corrupted and so that
	// serialized requests don't leak data from TOCTOU errors
	mu sync.Mutex

	authInfo auth.AuthInfo
}

func newAuthApplier(as auth.AuthStore, base applier, lessor lease.ILessor) *authApplier {
	return &authApplier{applier: base, as: as, lessor: lessor}
}

func (aa *authApplier) Apply(r *pb.InternalRaftRequest) *applyResult {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	if r.Header != nil {
		aa.authInfo.Username = r.Header.Username
		aa.authInfo.Revision = r.Header.AuthRevision
	}
	if needAdminPermission(r) {
		if err := aa.as.IsAdminPermitted(&aa.authInfo); err != nil {
			aa.authInfo.Username = ""
			aa.authInfo.Revision = 0
			return &applyResult{err: err}
		}
	}
	ret := aa.applier.Apply(r)
	aa.authInfo.Username = ""
	aa.authInfo.Revision = 0
	return ret
}

func (aa *authApplier) Put(ctx context.Context, txn mvcc.ITxnWrite, r *pb.PutRequest) (*pb.PutResponse, *traceutil.Trace, error) {
	if err := aa.as.IsPutPermitted(&aa.authInfo, r.Key); err != nil {
		return nil, nil, err
	}

	if err := aa.checkLeasePuts(lease.LeaseID(r.Lease)); err != nil {
		// The specified lease is already attached with a key that cannot
		// be written by this user. It means the user cannot revoke the
		// lease so attaching the lease to the newly written key should
		// be forbidden.
		return nil, nil, err
	}

	if r.PrevKv {
		err := aa.as.IsRangePermitted(&aa.authInfo, r.Key, nil)
		if err != nil {
			return nil, nil, err
		}
	}
	return aa.applier.Put(ctx, txn, r)
}

func (aa *authApplier) Range(ctx context.Context, txn mvcc.ITxnRead, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	if err := aa.as.IsRangePermitted(&aa.authInfo, r.Key, r.RangeEnd); err != nil {
		return nil, err
	}
	return aa.applier.Range(ctx, txn, r)
}

func (aa *authApplier) DeleteRange(txn mvcc.ITxnWrite, r *pb.DeleteRangeRequest) (*pb.DeleteRangeResponse, error) {
	if err := aa.as.IsDeleteRangePermitted(&aa.authInfo, r.Key, r.RangeEnd); err != nil {
		return nil, err
	}
	if r.PrevKv {
		err := aa.as.IsRangePermitted(&aa.authInfo, r.Key, r.RangeEnd)
		if err != nil {
			return nil, err
		}
	}

	return aa.applier.DeleteRange(txn, r)
}

func checkTxnReqsPermission(as auth.AuthStore, ai *auth.AuthInfo, reqs []*pb.RequestOp) error {
	for _, requ := range reqs {
		switch tv := requ.Request.(type) {
		case *pb.RequestOp_RequestRange:
			if tv.RequestRange == nil {
				continue
			}

			if err := as.IsRangePermitted(ai, tv.RequestRange.Key, tv.RequestRange.RangeEnd); err != nil {
				return err
			}

		case *pb.RequestOp_RequestPut:
			if tv.RequestPut == nil {
				continue
			}

			if err := as.IsPutPermitted(ai, tv.RequestPut.Key); err != nil {
				return err
			}

		case *pb.RequestOp_RequestDeleteRange:
			if tv.RequestDeleteRange == nil {
				continue
			}

			if tv.RequestDeleteRange.PrevKv {
				err := as.IsRangePermitted(ai, tv.RequestDeleteRange.Key, tv.RequestDeleteRange.RangeEnd)
				if err != nil {
					return err
				}
			}

			err := as.IsDeleteRangePermitted(ai, tv.RequestDeleteRange.Key, tv.RequestDeleteRange.RangeEnd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkTxnAuth(as auth.AuthStore, ai *auth.AuthInfo, rt *pb.TxnRequest) error {
	for _, c := range rt.Compare {
		if err := as.IsRangePermitted(ai, c.Key, c.RangeEnd); err != nil {
			return err
		}
	}
	if err := checkTxnReqsPermission(as, ai, rt.Success); err != nil {
		return err
	}
	return checkTxnReqsPermission(as, ai, rt.Failure)
}

func (aa *authApplier) Txn(ctx context.Context, rt *pb.TxnRequest) (*pb.TxnResponse, *traceutil.Trace, error) {
	if err := checkTxnAuth(aa.as, &aa.authInfo, rt); err != nil {
		return nil, nil, err
	}
	return aa.applier.Txn(ctx, rt)
}

//func (aa *authApplier) LeaseRevoke(lc *pb.LeaseRevokeRequest) (*pb.LeaseRevokeResponse, error) {
//	if err := aa.checkLeasePuts(lease.LeaseID(lc.ID)); err != nil {
//		return nil, err
//	}
//	return aa.applier.LeaseRevoke(lc)
//}

func (aa *authApplier) checkLeasePuts(leaseID lease.LeaseID) error {
	le := aa.lessor.Lookup(leaseID)
	if le != nil {
		for _, key := range le.Keys() {
			if err := aa.as.IsPutPermitted(&aa.authInfo, []byte(key)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (aa *authApplier) UserGet(r *pb.AuthUserGetRequest) (*pb.AuthUserGetResponse, error) {
	err := aa.as.IsAdminPermitted(&aa.authInfo)
	if err != nil && r.Name != aa.authInfo.Username {
		aa.authInfo.Username = ""
		aa.authInfo.Revision = 0
		return &pb.AuthUserGetResponse{}, err
	}

	return aa.applier.UserGet(r)
}

func (aa *authApplier) RoleGet(r *pb.AuthRoleGetRequest) (*pb.AuthRoleGetResponse, error) {
	err := aa.as.IsAdminPermitted(&aa.authInfo)
	if err != nil && !aa.as.HasRole(aa.authInfo.Username, r.Role) {
		aa.authInfo.Username = ""
		aa.authInfo.Revision = 0
		return &pb.AuthRoleGetResponse{}, err
	}

	return aa.applier.RoleGet(r)
}

func needAdminPermission(r *pb.InternalRaftRequest) bool {
	switch {
	case r.AuthEnable != nil:
		return true
	case r.AuthDisable != nil:
		return true
	case r.AuthStatus != nil:
		return true
	case r.AuthUserAdd != nil:
		return true
	case r.AuthUserDelete != nil:
		return true
	case r.AuthUserChangePassword != nil:
		return true
	case r.AuthUserGrantRole != nil:
		return true
	case r.AuthUserRevokeRole != nil:
		return true
	case r.AuthRoleAdd != nil:
		return true
	case r.AuthRoleGrantPermission != nil:
		return true
	case r.AuthRoleRevokePermission != nil:
		return true
	case r.AuthRoleDelete != nil:
		return true
	case r.AuthUserList != nil:
		return true
	case r.AuthRoleList != nil:
		return true
	default:
		return false
	}
}
