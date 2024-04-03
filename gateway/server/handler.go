// Copyright 2024 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"path"
	"reflect"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/proxy/server"
)

type rpcHandler struct {
	options server.HandlerOptions
	eps     []*dsypb.Endpoint
}

func (r *rpcHandler) Name() string {
	return r.options.ServiceDesc.ServiceName
}

func (r *rpcHandler) Handler() interface{} {
	return r
}

func (r *rpcHandler) Endpoints() []*dsypb.Endpoint {
	return r.eps
}

func (r *rpcHandler) Options() server.HandlerOptions {
	return r.options
}

func newRPCHandler(handler any, opts ...server.HandlerOption) *rpcHandler {
	var options server.HandlerOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.ServiceDesc == nil {
		panic("missing ServiceDesc")
	}

	typ := reflect.TypeOf(handler)
	name := options.ServiceDesc.ServiceName

	endpoints := make([]*dsypb.Endpoint, 0)

	for m := 0; m < typ.NumMethod(); m++ {
		if e := extractGRPCEndpoint(typ.Method(m)); e != nil {
			paths := path.Join("/", name, e.Name)
			e.Name = name + "." + e.Name

			e.Metadata[api.EndpointKey] = e.Name
			e.Metadata[api.HandlerKey] = api.RPCHandler
			e.Metadata[api.ContentTypeKey] = "application/grpc+json"
			e.Metadata[api.URLKey] = paths
			e.Metadata[api.MethodKey] = paths
			e.Metadata[api.ProtocolKey] = "grpc"

			for k, v := range options.Metadata[e.Name] {
				e.Metadata[k] = v
			}

			endpoints = append(endpoints, e)
		}
	}

	return &rpcHandler{
		options: options,
		eps:     endpoints,
	}
}
