/*
   Copyright 2024 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package routes

import (
	"path"

	"github.com/gin-gonic/gin"
	json "github.com/json-iterator/go"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/pkg/tonic"
	"github.com/olive-io/olive/pkg/tonic/fizz"
	"github.com/olive-io/olive/pkg/tonic/openapi"
)

type ServiceGroup struct {
	lg  *zap.Logger
	oct *client.Client
}

func (tree *RouteTree) registerServiceGroup() error {
	sg := &ServiceGroup{lg: tree.lg, oct: tree.oct}
	summary := sg.Summary()

	group := tree.root.Group("/discovery", summary.Name, summary.Description, sg.HandlerChains()...)
	group.GET("/service/list", []fizz.OperationOption{
		fizz.Summary("List all gateway service in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(sg.serviceList, 200))

	group.GET("/endpoint/list", []fizz.OperationOption{
		fizz.Summary("List all proxy endpoint in olive system."),
		fizz.Security(&openapi.SecurityRequirement{"Bearer": []string{}}),
	}, tonic.Handler(sg.endpointList, 200))

	return tree.Group(sg)
}

func (sg *ServiceGroup) Summary() RouteGroupSummary {
	return RouteGroupSummary{
		Name:        "olive.Service",
		Description: "the documents of olive proxy service",
	}
}

func (sg *ServiceGroup) HandlerChains() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}

type ServiceListRequest struct {
	Namespace string `query:"namespace" default:"default"`
}

type ServiceListResponse struct {
	Header   *pb.ResponseHeader
	Services []*dsypb.Service
}

func (sg *ServiceGroup) serviceList(ctx *gin.Context, in *ServiceListRequest) (*ServiceListResponse, error) {
	prefix := runtime.DefaultRunnerDiscoveryNode
	if in.Namespace != "" {
		prefix = path.Join(prefix, in.Namespace)
	}
	prefix = path.Join(prefix, "_svc_")

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	rsp, err := sg.oct.Get(ctx, prefix, options...)
	if err != nil {
		return nil, err
	}

	services := make([]*dsypb.Service, 0, len(rsp.Kvs))
	for _, kv := range rsp.Kvs {
		service := &dsypb.Service{}
		if err := json.Unmarshal(kv.Value, service); err == nil {
			services = append(services, service)
		}
	}

	resp := &ServiceListResponse{
		Services: services,
	}
	if rsp.Header != nil {
		resp.Header = &pb.ResponseHeader{
			ClusterId: rsp.Header.ClusterId,
			MemberId:  rsp.Header.MemberId,
			RaftTerm:  rsp.Header.RaftTerm,
		}
	}
	return resp, nil
}

type EndpointListRequest struct {
	Namespace string `query:"namespace" default:"default"`
}

type EndpointListResponse struct {
	Header    *pb.ResponseHeader
	Endpoints []*dsypb.Endpoint
}

func (sg *ServiceGroup) endpointList(ctx *gin.Context, in *EndpointListRequest) (*EndpointListResponse, error) {
	prefix := runtime.DefaultRunnerDiscoveryNode
	if in.Namespace != "" {
		prefix = path.Join(prefix, in.Namespace)
	}
	prefix = path.Join(prefix, "_ep_")

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	rsp, err := sg.oct.Get(ctx, prefix, options...)
	if err != nil {
		return nil, err
	}

	endpoints := make([]*dsypb.Endpoint, 0, len(rsp.Kvs))
	for _, kv := range rsp.Kvs {
		ep := &dsypb.Endpoint{}
		if err := json.Unmarshal(kv.Value, ep); err == nil {
			endpoints = append(endpoints, ep)
		}
	}

	resp := &EndpointListResponse{
		Endpoints: endpoints,
	}
	if rsp.Header != nil {
		resp.Header = &pb.ResponseHeader{
			ClusterId: rsp.Header.ClusterId,
			MemberId:  rsp.Header.MemberId,
			RaftTerm:  rsp.Header.RaftTerm,
		}
	}
	return resp, nil
}
