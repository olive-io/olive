package meta

import (
	"context"

	"github.com/olive-io/olive/api/olivepb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
)

func (s *Server) DeployDefinition(ctx context.Context, req *olivepb.DeployDefinitionRequest) (resp *olivepb.DeployDefinitionResponse, err error) {
	//TODO implement me
	client := v3client.New(s.etcd.Server)

	client.Get(ctx, req.Id)
	return
}

func (s *Server) ListDefinition(ctx context.Context, req *olivepb.ListDefinitionRequest) (resp *olivepb.ListDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) GetDefinition(ctx context.Context, req *olivepb.GetDefinitionRequest) (resp *olivepb.GetDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) RemoveDefinition(ctx context.Context, req *olivepb.RemoveDefinitionRequest) (resp *olivepb.RemoveDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *olivepb.ExecuteDefinitionRequest) (resp *olivepb.ExecuteDefinitionResponse, err error) {
	//TODO implement me
	panic("implement me")
}
