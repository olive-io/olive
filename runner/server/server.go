package server

import (
	"context"
	"path"

	json "github.com/bytedance/sonic"

	"github.com/olive-io/olive/apis"
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	pb "github.com/olive-io/olive/apis/rpc/runnerpb"

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

	resp.List, err = s.scheduler.ListDefinition(ctx, req.Id)
	return
}

func (s *GRPCRunnerServer) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	resp = &pb.GetDefinitionResponse{}

	resp.Definition, err = s.scheduler.GetDefinition(ctx, req.Id, req.Version)
	return
}

func (s *GRPCRunnerServer) ListProcessInstances(ctx context.Context, req *pb.ListProcessInstancesRequest) (resp *pb.ListProcessInstancesResponse, err error) {
	resp = &pb.ListProcessInstancesResponse{}

	resp.List, err = s.scheduler.ListProcess(ctx, req.DefinitionId, req.DefinitionVersion)
	return
}

func (s *GRPCRunnerServer) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (resp *pb.GetProcessInstanceResponse, err error) {
	resp = &pb.GetProcessInstanceResponse{}

	resp.Process, err = s.scheduler.GetProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	return
}

func (s *GRPCRunnerServer) RunProcessInstance(ctx context.Context, req *pb.RunProcessInstanceRequest) (resp *pb.RunProcessInstanceResponse, err error) {
	resp = &pb.RunProcessInstanceResponse{}

	headers := req.Headers
	properties := req.Properties
	dataObjects := req.DataObjects
	resp.Process, err = s.scheduler.RunProcess(ctx, req.DefinitionName, req.DefinitionVersion, req.Content, req.Process, req.InstanceName, headers, properties, dataObjects)
	return
}

func (s *GRPCRunnerServer) List(ctx context.Context, req *pb.ListRequest) (resp *pb.ListResponse, err error) {
	resp = &pb.ListResponse{}
	options := req.Options

	gvk := apis.FromGVK(options.GVK)
	key := gvk.String()

	outs, err := s.bs.NewList(gvk)
	if err != nil {
		return nil, err
	}

	err = s.bs.GetList(ctx, key, outs)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(outs)
	if err != nil {
		return nil, err
	}

	result := &apidiscoveryv1.APIResult{
		GVK:  options.GVK,
		List: true,
		Data: data,
	}
	resp.Result = result

	return
}

func (s *GRPCRunnerServer) Get(ctx context.Context, req *pb.GetRequest) (resp *pb.GetResponse, err error) {
	resp = &pb.GetResponse{}
	options := req.Options

	gvk := apis.FromGVK(options.GVK)
	key := gvk.String()
	if options.UID != "" {
		key = path.Join(key, options.UID)
	}

	out, err := s.bs.New(gvk)
	if err != nil {
		return nil, err
	}

	err = s.bs.Get(ctx, key, out)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	result := &apidiscoveryv1.APIResult{
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

	gvk := apis.FromGVK(options.GVK)
	key := gvk.String()
	if options.UID != "" {
		key = path.Join(key, options.UID)
	}

	out, err := s.bs.New(gvk)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(options.Data, out); err != nil {
		return nil, err
	}

	err = s.bs.Create(ctx, key, out, 0)
	if err != nil {
		return nil, err
	}

	return
}

func (s *GRPCRunnerServer) Patch(ctx context.Context, req *pb.PatchRequest) (resp *pb.PatchResponse, err error) {
	resp = &pb.PatchResponse{}
	options := req.Options

	gvk := apis.FromGVK(options.GVK)
	key := gvk.String()
	if options.UID != "" {
		key = path.Join(key, options.UID)
	}

	out, err := s.bs.New(gvk)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(options.Data, out); err != nil {
		return nil, err
	}

	err = s.bs.Create(ctx, key, out, 0)
	if err != nil {
		return nil, err
	}

	return
}

func (s *GRPCRunnerServer) Delete(ctx context.Context, req *pb.DeleteRequest) (resp *pb.DeleteResponse, err error) {
	resp = &pb.DeleteResponse{}
	options := req.Options

	gvk := apis.FromGVK(options.GVK)
	key := gvk.String()
	if options.UID != "" {
		key = path.Join(key, options.UID)
	}

	err = s.bs.Delete(ctx, key)
	if err != nil {
		return nil, err
	}

	return
}
