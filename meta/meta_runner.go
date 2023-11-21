package meta

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
)

func (s *Server) RegistryRunner(ctx context.Context, req *pb.RegistryRunnerRequest) (resp *pb.RegistryRunnerResponse, err error) {
	resp = &pb.RegistryRunnerResponse{}
	return
}

func (s *Server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (resp *pb.HeartbeatResponse, err error) {
	resp = &pb.HeartbeatResponse{}
	return
}
