package rpc

import (
	"context"
	"time"

	"github.com/olive-io/olive/api/rpctypes"
	pb "github.com/olive-io/olive/api/serverpb"
	"github.com/olive-io/olive/meta/api"
	"github.com/olive-io/olive/server"
	"github.com/olive-io/olive/server/membership"
	"go.etcd.io/etcd/client/pkg/v3/types"
)

type ClusterServer struct {
	cluster api.Cluster
	ra      *server.Replica
}

func NewClusterServer(ra *server.Replica) *ClusterServer {
	return &ClusterServer{
		cluster: ra.Cluster(),
		ra:      ra,
	}
}

func (cs *ClusterServer) MemberAdd(ctx context.Context, r *pb.MemberAddRequest) (*pb.MemberAddResponse, error) {
	urls, err := types.NewURLs(r.PeerURLs)
	if err != nil {
		return nil, rpctypes.ErrGRPCMemberBadURLs
	}

	now := time.Now()
	var m *membership.Member
	if r.IsLearner {
		m = membership.NewMemberAsLearner("", urls, "", &now)
	} else {
		m = membership.NewMember("", urls, "", &now)
	}
	membs, merr := cs.ra.AddMember(ctx, *m)
	if merr != nil {
		return nil, togRPCError(merr)
	}

	return &pb.MemberAddResponse{
		Header: cs.header(),
		Member: &pb.Member{
			ID:        m.ID,
			PeerURLs:  m.PeerURLs,
			IsLearner: m.IsLearner,
		},
		Members: membersToProtoMembers(membs),
	}, nil
}

func (cs *ClusterServer) MemberRemove(ctx context.Context, r *pb.MemberRemoveRequest) (*pb.MemberRemoveResponse, error) {
	membs, err := cs.ra.RemoveMember(ctx, r.ID)
	if err != nil {
		return nil, togRPCError(err)
	}
	return &pb.MemberRemoveResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
}

func (cs *ClusterServer) MemberUpdate(ctx context.Context, r *pb.MemberUpdateRequest) (*pb.MemberUpdateResponse, error) {
	m := membership.Member{
		ID:             r.ID,
		RaftAttributes: membership.RaftAttributes{PeerURLs: r.PeerURLs},
	}
	membs, err := cs.ra.UpdateMember(ctx, m)
	if err != nil {
		return nil, togRPCError(err)
	}
	return &pb.MemberUpdateResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
}

func (cs *ClusterServer) MemberList(ctx context.Context, r *pb.MemberListRequest) (*pb.MemberListResponse, error) {
	//if r.Linearizable {
	//	if err := cs.ra.LinearizableReadNotify(ctx); err != nil {
	//		return nil, togRPCError(err)
	//	}
	//}
	membs := membersToProtoMembers(cs.cluster.Members())
	return &pb.MemberListResponse{Header: cs.header(), Members: membs}, nil
}

func (cs *ClusterServer) MemberPromote(ctx context.Context, r *pb.MemberPromoteRequest) (*pb.MemberPromoteResponse, error) {
	membs, err := cs.ra.PromoteMember(ctx, r.ID)
	if err != nil {
		return nil, togRPCError(err)
	}
	return &pb.MemberPromoteResponse{Header: cs.header(), Members: membersToProtoMembers(membs)}, nil
}

func (cs *ClusterServer) header() *pb.ResponseHeader {
	return &pb.ResponseHeader{ShardId: cs.cluster.ID(), NodeId: cs.ra.NodeID(), RaftTerm: cs.ra.Term()}
}

func membersToProtoMembers(membs []*membership.Member) []*pb.Member {
	protoMembs := make([]*pb.Member, len(membs))
	for i := range membs {
		protoMembs[i] = &pb.Member{
			Name:       membs[i].Name,
			ID:         membs[i].ID,
			PeerURLs:   membs[i].PeerURLs,
			ClientURLs: membs[i].ClientURLs,
			IsLearner:  membs[i].IsLearner,
		}
	}
	return protoMembs
}
