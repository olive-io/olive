package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	json "github.com/json-iterator/go"
	"github.com/olive-io/olive/api/raftpb"
	"github.com/olive-io/olive/server/membership"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.uber.org/zap"
)

func (ra *Replica) checkMembershipOperationPermission(ctx context.Context) error {
	if ra.authStore == nil {
		// In the context of ordinary olive process, s.authStore will never be nil.
		// This branch is for handling cases in server_test.go
		return nil
	}

	// Note that this permission check is done in the API layer,
	// so TOCTOU problem can be caused potentially in a schedule like this:
	// update membership with user A -> revoke root role of A -> apply membership change
	// in the state machine layer
	// However, both of membership change and role management requires the root privilege.
	// So careful operation by admins can prevent the problem.
	authInfo, err := ra.AuthInfoFromCtx(ctx)
	if err != nil {
		return err
	}

	return ra.AuthStore().IsAdminPermitted(authInfo)
}

func (ra *Replica) AddMember(ctx context.Context, memb membership.Member) ([]*membership.Member, error) {
	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
		return nil, err
	}

	// TODO: move Member to protobuf type
	b, err := json.Marshal(memb)
	if err != nil {
		return nil, err
	}

	// by default StrictReconfigCheck is enabled; reject new members if unhealthy.
	if err := ra.mayAddMember(memb); err != nil {
		return nil, err
	}

	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeType_ConfChangeAddNode,
		NodeID:  memb.ID,
		Context: b,
	}

	if memb.IsLearner {
		cc.Type = raftpb.ConfChangeType_ConfChangeAddLearnerNode
	}

	members, err := ra.configure(ctx, cc)
	if err != nil {
		return nil, err
	}

	ra.cluster.AddMember(&memb)

	return members, nil
}

func (ra *Replica) mayAddMember(memb membership.Member) error {
	lg := ra.Logger()
	if !ra.StrictReconfigCheck {
		return nil
	}

	// protect quorum when adding voting member
	if !memb.IsLearner && !ra.cluster.IsReadyToAddVotingMember() {
		lg.Warn(
			"rejecting member add request; not enough healthy members",
			zap.Uint64("local-member-id", ra.NodeID()),
			zap.String("requested-member-add", fmt.Sprintf("%+v", memb)),
			zap.Error(ErrNotEnoughStartedMembers),
		)
		return ErrNotEnoughStartedMembers
	}

	//if !isConnectedFullySince(ra.r.transport, time.Now().Add(-HealthInterval), ra.ID(), ra.cluster.VotingMembers()) {
	//	lg.Warn(
	//		"rejecting member add request; local member has not been connected to all peers, reconfigure breaks active quorum",
	//		zap.Uint64("local-member-id", ra.NodeID()),
	//		zap.String("requested-member-add", fmt.Sprintf("%+v", memb)),
	//		zap.Error(ErrUnhealthy),
	//	)
	//	return ErrUnhealthy
	//}

	return nil
}

func (ra *Replica) RemoveMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
		return nil, err
	}

	// by default StrictReconfigCheck is enabled; reject removal if leads to quorum loss
	if err := ra.mayRemoveMember(id); err != nil {
		return nil, err
	}

	cc := raftpb.ConfChange{
		Type:   raftpb.ConfChangeType_ConfChangeRemoveNode,
		NodeID: id,
	}

	members, err := ra.configure(ctx, cc)
	if err != nil {
		return nil, err
	}

	ra.cluster.RemoveMember(id)

	return members, nil
}

// PromoteMember promotes a learner node to a voting node.
func (ra *Replica) PromoteMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
	// only raft leader has information on whether the to-be-promoted learner node is ready. If promoteMember call
	// fails with ErrNotLeader, forward the request to leader node via HTTP. If promoteMember call fails with error
	// other than ErrNotLeader, return the error.
	resp, err := ra.promoteMember(ctx, id)
	if err == nil {
		learnerPromoteSucceed.Inc()
		return resp, nil
	}
	if !errors.Is(err, ErrNotLeader) {
		learnerPromoteFailed.WithLabelValues(err.Error()).Inc()
		return resp, err
	}

	cctx, cancel := context.WithTimeout(ctx, ra.ReqTimeout())
	defer cancel()
	// forward to leader
	for cctx.Err() == nil {
		leader, err := ra.waitLeader(cctx)
		if err != nil {
			return nil, err
		}
		for _, url := range leader.PeerURLs {
			_ = url
			//resp, err := promoteMemberHTTP(cctx, url, id, ra.peerRt)
			//if err == nil {
			//	return resp, nil
			//}
			//// If member promotion failed, return early. Otherwise keep retry.
			//if err == ErrLearnerNotReady || err == membership.ErrIDNotFound || err == membership.ErrMemberNotLearner {
			//	return nil, err
			//}
		}
	}

	if errors.Is(cctx.Err(), context.DeadlineExceeded) {
		return nil, ErrTimeout
	}
	return nil, ErrCanceled
}

// promoteMember checks whether the to-be-promoted learner node is ready before sending the promote
// request to raft.
// The function returns ErrNotLeader if the local node is not raft leader (therefore does not have
// enough information to determine if the learner node is ready), returns ErrLearnerNotReady if the
// local node is leader (therefore has enough information) but decided the learner node is not ready
// to be promoted.
func (ra *Replica) promoteMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
		return nil, err
	}

	// check if we can promote this learner.
	if err := ra.mayPromoteMember(id); err != nil {
		return nil, err
	}

	// build the context for the promote confChange. mark IsLearner to false and IsPromote to true.
	promoteChangeContext := membership.ConfigChangeContext{
		Member: membership.Member{
			ID: id,
		},
		IsPromote: true,
	}

	b, err := json.Marshal(promoteChangeContext)
	if err != nil {
		return nil, err
	}

	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeType_ConfChangeAddNode,
		NodeID:  id,
		Context: b,
	}

	return ra.configure(ctx, cc)
}

func (ra *Replica) mayPromoteMember(id uint64) error {
	lg := ra.Logger()
	err := ra.isLearnerReady(uint64(id))
	if err != nil {
		return err
	}

	if !ra.StrictReconfigCheck {
		return nil
	}
	if !ra.cluster.IsReadyToPromoteMember(id) {
		lg.Warn(
			"rejecting member promote request; not enough healthy members",
			zap.Uint64("local-member-id", ra.NodeID()),
			zap.Uint64("requested-member-remove-id", id),
			zap.Error(ErrNotEnoughStartedMembers),
		)
		return ErrNotEnoughStartedMembers
	}

	return nil
}

// check whether the learner catches up with leader or not.
// Note: it will return nil if member is not found in cluster or if member is not learner.
// These two conditions will be checked before apply phase later.
func (ra *Replica) isLearnerReady(id uint64) error {
	if err := ra.waitAppliedIndex(); err != nil {
		return err
	}

	//rs := ra.raftStatus()
	//
	//// leader's raftStatus.Progress is not nil
	//if rs.Progress == nil {
	//	return ErrNotLeader
	//}
	//
	//var learnerMatch uint64
	//isFound := false
	//leaderID := rs.ID
	//for memberID, progress := range rs.Progress {
	//	if id == memberID {
	//		// check its status
	//		learnerMatch = progress.Match
	//		isFound = true
	//		break
	//	}
	//}
	//
	//// We should return an error in API directly, to avoid the request
	//// being unnecessarily delivered to raft.
	//if !isFound {
	//	return membership.ErrIDNotFound
	//}
	//
	//leaderMatch := rs.Progress[leaderID].Match
	//// the learner's Match not caught up with leader yet
	//if float64(learnerMatch) < float64(leaderMatch)*readyPercent {
	//	return ErrLearnerNotReady
	//}

	return nil
}

func (ra *Replica) mayRemoveMember(id uint64) error {
	if !ra.StrictReconfigCheck {
		return nil
	}

	lg := ra.lg
	isLearner_ := ra.cluster.IsMemberExist(id) && ra.cluster.Member(id).IsLearner
	// no need to check quorum when removing non-voting member
	if isLearner_ {
		return nil
	}

	if !ra.cluster.IsReadyToRemoveVotingMember(id) {
		lg.Warn(
			"rejecting member remove request; not enough healthy members",
			zap.Uint64("local-member-id", ra.NodeID()),
			zap.Uint64("requested-member-remove-id", id),
			zap.Error(ErrNotEnoughStartedMembers),
		)
		return ErrNotEnoughStartedMembers
	}

	// downed member is safe to remove since it's not part of the active quorum
	//if t := s.r.transport.ActiveSince(id); id != s.ID() && t.IsZero() {
	//	return nil
	//}
	//
	//// protect quorum if some members are down
	//m := ra.cluster.VotingMembers()
	//active := numConnectedSince(s.r.transport, time.Now().Add(-HealthInterval), s.ID(), m)
	//if (active - 1) < 1+((len(m)-1)/2) {
	//	lg.Warn(
	//		"rejecting member remove request; local member has not been connected to all peers, reconfigure breaks active quorum",
	//		zap.Uint64("local-member-id", ra.NodeID()),
	//		zap.Uint64("requested-member-remove", id),
	//		zap.Int("active-peers", active),
	//		zap.Error(ErrUnhealthy),
	//	)
	//	return ErrUnhealthy
	//}

	return nil
}

func (ra *Replica) UpdateMember(ctx context.Context, memb membership.Member) ([]*membership.Member, error) {
	b, merr := json.Marshal(memb)
	if merr != nil {
		return nil, merr
	}

	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
		return nil, err
	}
	cc := raftpb.ConfChange{
		Type:    raftpb.ConfChangeType_ConfChangeUpdateNode,
		NodeID:  memb.ID,
		Context: b,
	}

	ra.cluster.UpdateAttributes(memb.ID, memb.Attributes)
	members, err := ra.configure(ctx, cc)
	if err != nil {
		return nil, err
	}

	return members, nil
}

type confChangeResponse struct {
	membs []*membership.Member
	err   error
}

// configure sends a configuration change through consensus and
// then waits for it to be applied to the server. It
// will block until the change is performed or there is an error.
func (ra *Replica) configure(ctx context.Context, cc raftpb.ConfChange) ([]*membership.Member, error) {
	if err := ra.cluster.ValidateConfigurationChange(cc); err != nil {
		return nil, err
	}

	lg := ra.lg
	cc.ID = ra.reqIDGen.Next()

	rc := make(chan any, 1)
	req := &replicaRequest{
		shardID: ra.ShardID(),
		nodeID:  ra.NodeID(),
		syncConfChange: &replicaSyncConfChange{
			ctx: ctx,
			cc:  cc,
			rc:  rc,
		},
	}

	start := time.Now()
	ra.ReplicaRequest(req)

	select {
	case x := <-rc:
		if x == nil {
			lg.Panic("failed to configure")
		}
		resp := x.(*confChangeResponse)
		lg.Info(
			"applied a configuration change through raft",
			zap.Uint64("local-member-id", ra.NodeID()),
			zap.String("raft-conf-change", cc.Type.String()),
			zap.String("raft-conf-change-node-id", types.ID(cc.NodeID).String()),
		)
		return resp.membs, resp.err

	case <-ctx.Done():
		return nil, ra.parseProposeCtxErr(ctx.Err(), start)

	case <-ra.stopping:
		return nil, ErrStopped
	}
}

func (ra *Replica) waitLeader(ctx context.Context) (*membership.Member, error) {
	leader := ra.cluster.Member(ra.Leader())
	for leader == nil {
		// wait an election
		dur := time.Duration(ra.RTTMillisecond) * time.Duration(ra.ElectionTTL) * time.Millisecond
		select {
		case <-time.After(dur):
			leader = ra.cluster.Member(ra.Leader())
		case <-ra.stopping:
			return nil, ErrStopped
		case <-ctx.Done():
			return nil, ErrNoLeader
		}
	}
	if leader == nil || len(leader.PeerURLs) == 0 {
		return nil, ErrNoLeader
	}
	return leader, nil
}
