package meta

import (
	"context"

	"github.com/olive-io/olive/api/olivepb"
)

func (s *Server) RegistryRunner(ctx context.Context, req *olivepb.RegistryRunnerRequest) (resp *olivepb.RegistryRunnerResponse, err error) {
	resp = &olivepb.RegistryRunnerResponse{}
	return
}

func (s *Server) Heartbeat(ctx context.Context, req *olivepb.HeartbeatRequest) (resp *olivepb.HeartbeatResponse, err error) {
	resp = &olivepb.HeartbeatResponse{}
	return
}
