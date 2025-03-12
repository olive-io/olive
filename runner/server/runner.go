package server

import (
	"context"

	pb "github.com/olive-io/olive/api/rpc/runnerpb"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/storage"
)

type GRPCRunnerServer struct {
	pb.UnimplementedRunnerRPCServer

	scheduler *scheduler.Scheduler
	gather    *gather.Gather
	bs        storage.Storage
}

func NewGRPCRunnerServer(scheduler *scheduler.Scheduler, gather *gather.Gather, bs storage.Storage) *GRPCRunnerServer {
	s := &GRPCRunnerServer{
		scheduler: scheduler,
		gather:    gather,
		bs:        bs,
	}

	return s
}

func (s *GRPCRunnerServer) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (resp *pb.GetRunnerResponse, err error) {
	resp = &pb.GetRunnerResponse{}
	resp.Runner = s.gather.GetRunner()
	resp.Statistics = s.gather.GetStat()
	return resp, err
}

func (s *GRPCRunnerServer) ListDefinitions(ctx context.Context, req *pb.ListDefinitionsRequest) (resp *pb.ListDefinitionsResponse, err error) {
	resp = &pb.ListDefinitionsResponse{}

	resp.Definitions, err = s.scheduler.ListDefinition(ctx, req.Id)
	return
}

func (s *GRPCRunnerServer) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	resp = &pb.GetDefinitionResponse{}

	resp.Definition, err = s.scheduler.GetDefinition(ctx, req.Id, req.Version)
	return
}

func (s *GRPCRunnerServer) ListProcess(ctx context.Context, req *pb.ListProcessRequest) (resp *pb.ListProcessResponse, err error) {
	resp = &pb.ListProcessResponse{}

	resp.Processes, err = s.scheduler.ListProcess(ctx, req.DefinitionId, req.DefinitionVersion)
	return
}

func (s *GRPCRunnerServer) GetProcess(ctx context.Context, req *pb.GetProcessRequest) (resp *pb.GetProcessResponse, err error) {
	resp = &pb.GetProcessResponse{}

	resp.Process, err = s.scheduler.GetProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	return
}

func (s *GRPCRunnerServer) RunProcess(ctx context.Context, req *pb.RunProcessRequest) (resp *pb.RunProcessResponse, err error) {

	err = s.scheduler.RunProcess(ctx, req.Process)
	if err != nil {
		return
	}
	resp = &pb.RunProcessResponse{
		Process: req.Process,
	}
	return
}
