package meta

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
)

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	//TODO implement me
	client := v3client.New(s.etcd.Server)

	client.Get(ctx, req.Id)
	return
}

func (s *Server) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (resp *pb.ListDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (resp *pb.RemoveDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (resp *pb.ExecuteDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}
