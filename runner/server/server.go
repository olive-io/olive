package server

import (
	"context"

	"github.com/olive-io/olive/api/runnerpb"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
)

type GRPCRunnerServer struct {
	runnerpb.UnimplementedRunnerRPCServer

	scheduler *scheduler.Scheduler
	gather    *gather.Gather
}

func NewGRPCRunnerServer(scheduler *scheduler.Scheduler, gather *gather.Gather) *GRPCRunnerServer {
	s := &GRPCRunnerServer{
		scheduler: scheduler,
		gather:    gather,
	}

	return s
}

func (s *GRPCRunnerServer) GetRunner(ctx context.Context, req *runnerpb.GetRunnerRequest) (resp *runnerpb.GetRunnerResponse, err error) {
	resp = &runnerpb.GetRunnerResponse{}
	resp.Runner = s.gather.GetRunner()
	resp.Statistics, err = s.gather.GetStat()
	return resp, err
}

func (s *GRPCRunnerServer) ListDefinitions(ctx context.Context, req *runnerpb.ListDefinitionsRequest) (resp *runnerpb.ListDefinitionsResponse, err error) {
	resp = &runnerpb.ListDefinitionsResponse{}

	resp.Definitions, err = s.scheduler.ListDefinition(ctx, req.Id)
	return
}

func (s *GRPCRunnerServer) GetDefinition(ctx context.Context, req *runnerpb.GetDefinitionRequest) (resp *runnerpb.GetDefinitionResponse, err error) {
	resp = &runnerpb.GetDefinitionResponse{}

	resp.Definition, err = s.scheduler.GetDefinition(ctx, req.Id, req.Version)
	return
}

func (s *GRPCRunnerServer) ListProcessInstances(ctx context.Context, req *runnerpb.ListProcessInstancesRequest) (resp *runnerpb.ListProcessInstancesResponse, err error) {
	resp = &runnerpb.ListProcessInstancesResponse{}

	resp.Instances, err = s.scheduler.ListProcess(ctx, req.DefinitionId, req.DefinitionVersion)
	return
}

func (s *GRPCRunnerServer) GetProcessInstance(ctx context.Context, req *runnerpb.GetProcessInstanceRequest) (resp *runnerpb.GetProcessInstanceResponse, err error) {
	resp = &runnerpb.GetProcessInstanceResponse{}

	resp.Instance, err = s.scheduler.GetProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	return
}

func (s *GRPCRunnerServer) RunProcessInstance(ctx context.Context, req *runnerpb.RunProcessInstanceRequest) (resp *runnerpb.RunProcessInstanceResponse, err error) {
	resp = &runnerpb.RunProcessInstanceResponse{}

	headers := req.Headers
	properties := req.Properties
	dataObjects := req.DataObjects
	resp.Instance, err = s.scheduler.RunProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Content, req.Process, req.InstanceName, headers, properties, dataObjects)
	return
}
