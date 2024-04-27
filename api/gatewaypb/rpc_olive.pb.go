//
//Copyright 2023 The olive Authors
//
//This program is offered under a commercial and under the AGPL license.
//For AGPL licensing, see below.
//
//AGPL licensing:
//This program is free software: you can redistribute it and/or modify
//it under the terms of the GNU Affero General Public License as published by
//the Free Software Foundation, either version 3 of the License, or
//(at your option) any later version.
//
//This program is distributed in the hope that it will be useful,
//but WITHOUT ANY WARRANTY; without even the implied warranty of
//MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//GNU Affero General Public License for more details.
//
//You should have received a copy of the GNU Affero General Public License
//along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Code generated by protoc-gen-go-olive. DO NOT EDIT.
// versions:
// - protoc-gen-go-olive v0.1.0
// - protoc             v3.12.4
// source: github.com/olive-io/olive/api/gatewaypb/rpc.proto

package gatewaypb

import (
	server "github.com/olive-io/olive/pkg/proxy/server"
	grpc "google.golang.org/grpc"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.62.0 or later.
const _ = grpc.SupportPackageIsVersion8

func RegisterGatewayServerHandler(s server.IServer, srv GatewayServer, opts ...server.HandlerOption) error {
	type GatewayEmbedXX struct {
		GatewayServer
	}
	opts = append(opts, server.WithServerDesc(&Gateway_ServiceDesc))
	handler := s.NewHandler(&GatewayEmbedXX{srv}, opts...)
	return s.Handle(handler)
}

func RegisterEndpointRouterServerHandler(s server.IServer, srv EndpointRouterServer, opts ...server.HandlerOption) error {
	type EndpointRouterEmbedXX struct {
		EndpointRouterServer
	}
	opts = append(opts, server.WithServerDesc(&EndpointRouter_ServiceDesc))
	handler := s.NewHandler(&EndpointRouterEmbedXX{srv}, opts...)
	return s.Handle(handler)
}
