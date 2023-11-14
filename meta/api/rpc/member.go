package rpc

//
//type ClusterServer struct {
//	cluster api.Cluster
//	ra      *server.Replica
//}
//
//func NewClusterServer(ra *server.Replica) *ClusterServer {
//	return &ClusterServer{
//		cluster: ra.Cluster(),
//		ra:      ra,
//	}
//}
//
//func (cs *ClusterServer) MemberAdd(ctx context.Context, r *pb.MemberAddRequest) (*pb.MemberAddResponse, error) {
//	urls, err := types.NewURLs(r.PeerURLs)
//	if err != nil {
//		return nil, rpctypes.ErrGRPCMemberBadURLs
//	}
//
//	now := time.Now()
//	var m *membership.Member
//	if r.IsLearner {
//		m = membership.NewMemberAsLearner("", urls, "", &now)
//	} else {
//		m = membership.NewMember("", urls, "", &now)
//	}
//	membs, merr := cs.server.AddMember(ctx, *m)
//	if merr != nil {
//		return nil, togRPCError(merr)
//	}
//
//	return &pb.MemberAddResponse{
//		Header: cs.header(),
//		Member: &pb.Member{
//			ID:        uint64(m.ID),
//			PeerURLs:  m.PeerURLs,
//			IsLearner: m.IsLearner,
//		},
//		Members: membersToProtoMembers(membs),
//	}, nil
//}
//
//func (cs *ClusterServer) MemberRemove(ctx context.Context, r *pb.MemberRemoveRequest) (*pb.MemberRemoveResponse, error) {
//	membs, err := cs.server.RemoveMember(ctx, r.ID)
//	if err != nil {
//		return nil, togRPCError(err)
//	}
//	return &pb.MemberRemoveResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
//}
//
//func (cs *ClusterServer) MemberUpdate(ctx context.Context, r *pb.MemberUpdateRequest) (*pb.MemberUpdateResponse, error) {
//	m := membership.Member{
//		ID:             types.ID(r.ID),
//		RaftAttributes: membership.RaftAttributes{PeerURLs: r.PeerURLs},
//	}
//	membs, err := cs.server.UpdateMember(ctx, m)
//	if err != nil {
//		return nil, togRPCError(err)
//	}
//	return &pb.MemberUpdateResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
//}
//
//func (cs *ClusterServer) MemberList(ctx context.Context, r *pb.MemberListRequest) (*pb.MemberListResponse, error) {
//	if r.Linearizable {
//		if err := cs.server.LinearizableReadNotify(ctx); err != nil {
//			return nil, togRPCError(err)
//		}
//	}
//	membs := membersToProtoMembers(cs.cluster.Members())
//	return &pb.MemberListResponse{Header: cs.header(), Members: membs}, nil
//}
//
//func (cs *ClusterServer) MemberPromote(ctx context.Context, r *pb.MemberPromoteRequest) (*pb.MemberPromoteResponse, error) {
//	membs, err := cs.server.PromoteMember(ctx, r.ID)
//	if err != nil {
//		return nil, togRPCError(err)
//	}
//	return &pb.MemberPromoteResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
//}
//
//func (cs *ClusterServer) header() *pb.ResponseHeader {
//	return &pb.ResponseHeader{ShardId: uint64(cs.cluster.ID()), NodeId: uint64(cs.ra.NodeID()), RaftTerm: cs.ra.Term()}
//}
//
//func membersToProtoMembers(membs []*membership.Member) []*pb.Member {
//	protoMembs := make([]*pb.Member, len(membs))
//	for i := range membs {
//		protoMembs[i] = &pb.Member{
//			Name:       membs[i].Name,
//			ID:         uint64(membs[i].ID),
//			PeerURLs:   membs[i].PeerURLs,
//			ClientURLs: membs[i].ClientURLs,
//			IsLearner:  membs[i].IsLearner,
//		}
//	}
//	return protoMembs
//}
