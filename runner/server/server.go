package server

import (
	"context"

	json "github.com/bytedance/sonic"

	pb "github.com/olive-io/olive/api/runnerpb"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
	"github.com/olive-io/olive/runner/gather"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/storage"
)

type GRPCRunnerServer struct {
	pb.UnimplementedRunnerRPCServer

	scheduler *scheduler.Scheduler
	gather    *gather.Gather
	bs        *storage.Storage
}

func NewGRPCRunnerServer(scheduler *scheduler.Scheduler, gather *gather.Gather, bs *storage.Storage) *GRPCRunnerServer {
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
	resp.Statistics, err = s.gather.GetStat()
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

func (s *GRPCRunnerServer) ListProcessInstances(ctx context.Context, req *pb.ListProcessInstancesRequest) (resp *pb.ListProcessInstancesResponse, err error) {
	resp = &pb.ListProcessInstancesResponse{}

	resp.Instances, err = s.scheduler.ListProcess(ctx, req.DefinitionId, req.DefinitionVersion)
	return
}

func (s *GRPCRunnerServer) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (resp *pb.GetProcessInstanceResponse, err error) {
	resp = &pb.GetProcessInstanceResponse{}

	resp.Instance, err = s.scheduler.GetProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	return
}

func (s *GRPCRunnerServer) RunProcessInstance(ctx context.Context, req *pb.RunProcessInstanceRequest) (resp *pb.RunProcessInstanceResponse, err error) {
	resp = &pb.RunProcessInstanceResponse{}

	headers := req.Headers
	properties := req.Properties
	dataObjects := req.DataObjects
	resp.Instance, err = s.scheduler.RunProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Content, req.Process, req.InstanceName, headers, properties, dataObjects)
	return
}

func (s *GRPCRunnerServer) List(ctx context.Context, req *pb.ListRequest) (resp *pb.ListResponse, err error) {
	resp = &pb.ListResponse{}
	options := req.Options

	outs, err := s.bs.List(ctx, options)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(outs)
	if err != nil {
		return nil, err
	}

	result := &metav1.Result{
		GVK:   options.GVK,
		List:  true,
		Total: int64(len(outs)),
		Data:  data,
	}
	resp.Result = result

	return
}

func (s *GRPCRunnerServer) Get(ctx context.Context, req *pb.GetRequest) (resp *pb.GetResponse, err error) {
	resp = &pb.GetResponse{}
	options := req.Options

	outs, err := s.bs.Get(ctx, options)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(outs)
	if err != nil {
		return nil, err
	}

	result := &metav1.Result{
		GVK:   options.GVK,
		List:  false,
		Total: 1,
		Data:  data,
	}
	resp.Result = result

	return
}

func (s *GRPCRunnerServer) Post(ctx context.Context, req *pb.PostRequest) (resp *pb.PostResponse, err error) {
	resp = &pb.PostResponse{}
	options := req.Options

	err = s.bs.Post(ctx, options)
	if err != nil {
		return nil, err
	}

	return
}

func (s *GRPCRunnerServer) Patch(ctx context.Context, req *pb.PatchRequest) (resp *pb.PatchResponse, err error) {
	resp = &pb.PatchResponse{}
	options := req.Options

	err = s.bs.Patch(ctx, options)
	if err != nil {
		return nil, err
	}

	return
}

func (s *GRPCRunnerServer) Delete(ctx context.Context, req *pb.DeleteRequest) (resp *pb.DeleteResponse, err error) {
	resp = &pb.DeleteResponse{}
	options := req.Options

	err = s.bs.Delete(ctx, options)
	if err != nil {
		return nil, err
	}

	return
}
