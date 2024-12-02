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

package grpc

import (
	"path"
	"reflect"
	"strings"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/x/proxy/api"
	"github.com/olive-io/olive/x/proxy/server"
)

type rpcHandler struct {
	options server.HandlerOptions
	handler any
	eps     []*dsypb.Endpoint
}

func (r *rpcHandler) Name() string {
	return r.options.ServiceDesc.ServiceName
}

func (r *rpcHandler) Handler() any {
	return r.handler
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
	names := strings.Split(name, ".")

	endpoints := make([]*dsypb.Endpoint, 0)

	for m := 0; m < typ.NumMethod(); m++ {
		if e := extractGRPCEndpoint(typ.Method(m)); e != nil {
			paths := path.Join("/", name, e.Name)
			e.Name = names[len(names)-1] + "." + e.Name

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
		handler: handler,
		eps:     endpoints,
	}
}
