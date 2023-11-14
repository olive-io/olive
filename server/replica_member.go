package server

//
//func (ra *Replica) checkMembershipOperationPermission(ctx context.Context) error {
//	if ra.authStore == nil {
//		// In the context of ordinary etcd process, s.authStore will never be nil.
//		// This branch is for handling cases in server_test.go
//		return nil
//	}
//
//	// Note that this permission check is done in the API layer,
//	// so TOCTOU problem can be caused potentially in a schedule like this:
//	// update membership with user A -> revoke root role of A -> apply membership change
//	// in the state machine layer
//	// However, both of membership change and role management requires the root privilege.
//	// So careful operation by admins can prevent the problem.
//	authInfo, err := ra.AuthInfoFromCtx(ctx)
//	if err != nil {
//		return err
//	}
//
//	return ra.AuthStore().IsAdminPermitted(authInfo)
//}
//
//func (ra *Replica) AddMember(ctx context.Context, memb membership.Member) ([]*membership.Member, error) {
//	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
//		return nil, err
//	}
//
//	// TODO: move Member to protobuf type
//	b, err := json.Marshal(memb)
//	if err != nil {
//		return nil, err
//	}
//
//	// by default StrictReconfigCheck is enabled; reject new members if unhealthy.
//	if err := ra.mayAddMember(memb); err != nil {
//		return nil, err
//	}
//
//	cc := raftpb.ConfChange{
//		Type:    raftpb.ConfChangeAddNode,
//		NodeID:  uint64(memb.ID),
//		Context: b,
//	}
//
//	if memb.IsLearner {
//		cc.Type = raftpb.ConfChangeAddLearnerNode
//	}
//
//	return ra.configure(ctx, cc)
//}
//
//func (ra *Replica) mayAddMember(memb membership.Member) error {
//	lg := ra.Logger()
//	if !ra.Cfg.StrictReconfigCheck {
//		return nil
//	}
//
//	// protect quorum when adding voting member
//	if !memb.IsLearner && !ra.cluster.IsReadyToAddVotingMember() {
//		lg.Warn(
//			"rejecting member add request; not enough healthy members",
//			zap.String("local-member-id", ra.ID().String()),
//			zap.String("requested-member-add", fmt.Sprintf("%+v", memb)),
//			zap.Error(ErrNotEnoughStartedMembers),
//		)
//		return ErrNotEnoughStartedMembers
//	}
//
//	if !isConnectedFullySince(ra.r.transport, time.Now().Add(-HealthInterval), ra.ID(), ra.cluster.VotingMembers()) {
//		lg.Warn(
//			"rejecting member add request; local member has not been connected to all peers, reconfigure breaks active quorum",
//			zap.String("local-member-id", ra.ID().String()),
//			zap.String("requested-member-add", fmt.Sprintf("%+v", memb)),
//			zap.Error(ErrUnhealthy),
//		)
//		return ErrUnhealthy
//	}
//
//	return nil
//}
//
//func (ra *Replica) RemoveMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
//	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
//		return nil, err
//	}
//
//	// by default StrictReconfigCheck is enabled; reject removal if leads to quorum loss
//	if err := ra.mayRemoveMember(types.ID(id)); err != nil {
//		return nil, err
//	}
//
//	cc := raftpb.ConfChange{
//		Type:   raftpb.ConfChangeRemoveNode,
//		NodeID: id,
//	}
//	return ra.configure(ctx, cc)
//}
//
//// PromoteMember promotes a learner node to a voting node.
//func (ra *Replica) PromoteMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
//	// only raft leader has information on whether the to-be-promoted learner node is ready. If promoteMember call
//	// fails with ErrNotLeader, forward the request to leader node via HTTP. If promoteMember call fails with error
//	// other than ErrNotLeader, return the error.
//	resp, err := ra.promoteMember(ctx, id)
//	if err == nil {
//		learnerPromoteSucceed.Inc()
//		return resp, nil
//	}
//	if err != ErrNotLeader {
//		learnerPromoteFailed.WithLabelValues(err.Error()).Inc()
//		return resp, err
//	}
//
//	cctx, cancel := context.WithTimeout(ctx, ra.ReqTimeout())
//	defer cancel()
//	// forward to leader
//	for cctx.Err() == nil {
//		leader, err := ra.waitLeader(cctx)
//		if err != nil {
//			return nil, err
//		}
//		for _, url := range leader.PeerURLs {
//			resp, err := promoteMemberHTTP(cctx, url, id, ra.peerRt)
//			if err == nil {
//				return resp, nil
//			}
//			// If member promotion failed, return early. Otherwise keep retry.
//			if err == ErrLearnerNotReady || err == membership.ErrIDNotFound || err == membership.ErrMemberNotLearner {
//				return nil, err
//			}
//		}
//	}
//
//	if errors.Is(cctx.Err(), context.DeadlineExceeded) {
//		return nil, ErrTimeout
//	}
//	return nil, ErrCanceled
//}
//
//// promoteMember checks whether the to-be-promoted learner node is ready before sending the promote
//// request to raft.
//// The function returns ErrNotLeader if the local node is not raft leader (therefore does not have
//// enough information to determine if the learner node is ready), returns ErrLearnerNotReady if the
//// local node is leader (therefore has enough information) but decided the learner node is not ready
//// to be promoted.
//func (ra *Replica) promoteMember(ctx context.Context, id uint64) ([]*membership.Member, error) {
//	if err := ra.checkMembershipOperationPermission(ctx); err != nil {
//		return nil, err
//	}
//
//	// check if we can promote this learner.
//	if err := ra.mayPromoteMember(types.ID(id)); err != nil {
//		return nil, err
//	}
//
//	// build the context for the promote confChange. mark IsLearner to false and IsPromote to true.
//	promoteChangeContext := membership.ConfigChangeContext{
//		Member: membership.Member{
//			ID: types.ID(id),
//		},
//		IsPromote: true,
//	}
//
//	b, err := json.Marshal(promoteChangeContext)
//	if err != nil {
//		return nil, err
//	}
//
//	cc := raftpb.ConfChange{
//		Type:    raftpb.ConfChangeAddNode,
//		NodeID:  id,
//		Context: b,
//	}
//
//	return ra.configure(ctx, cc)
//}
//
//func (ra *Replica) mayPromoteMember(id types.ID) error {
//	lg := ra.Logger()
//	err := ra.isLearnerReady(uint64(id))
//	if err != nil {
//		return err
//	}
//
//	if !ra.Cfg.StrictReconfigCheck {
//		return nil
//	}
//	if !ra.cluster.IsReadyToPromoteMember(uint64(id)) {
//		lg.Warn(
//			"rejecting member promote request; not enough healthy members",
//			zap.String("local-member-id", ra.ID().String()),
//			zap.String("requested-member-remove-id", id.String()),
//			zap.Error(ErrNotEnoughStartedMembers),
//		)
//		return ErrNotEnoughStartedMembers
//	}
//
//	return nil
//}
