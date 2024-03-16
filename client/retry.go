// Copyright 2023 The olive Authors
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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
)

type retryPolicy uint8

const (
	repeatable retryPolicy = iota
	nonRepeatable
)

func (rp retryPolicy) String() string {
	switch rp {
	case repeatable:
		return "repeatable"
	case nonRepeatable:
		return "nonRepeatable"
	default:
		return "UNKNOWN"
	}
}

type retryClusterClient struct {
	cc pb.ClusterClient
}

// RetryClusterClient implements a ClusterClient.
func RetryClusterClient(c *Client) pb.ClusterClient {
	return &retryClusterClient{
		cc: pb.NewClusterClient(c.conn),
	}
}

func (rcc *retryClusterClient) MemberList(ctx context.Context, in *pb.MemberListRequest, opts ...grpc.CallOption) (resp *pb.MemberListResponse, err error) {
	return rcc.cc.MemberList(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rcc *retryClusterClient) MemberAdd(ctx context.Context, in *pb.MemberAddRequest, opts ...grpc.CallOption) (resp *pb.MemberAddResponse, err error) {
	return rcc.cc.MemberAdd(ctx, in, opts...)
}

func (rcc *retryClusterClient) MemberRemove(ctx context.Context, in *pb.MemberRemoveRequest, opts ...grpc.CallOption) (resp *pb.MemberRemoveResponse, err error) {
	return rcc.cc.MemberRemove(ctx, in, opts...)
}

func (rcc *retryClusterClient) MemberUpdate(ctx context.Context, in *pb.MemberUpdateRequest, opts ...grpc.CallOption) (resp *pb.MemberUpdateResponse, err error) {
	return rcc.cc.MemberUpdate(ctx, in, opts...)
}

func (rcc *retryClusterClient) MemberPromote(ctx context.Context, in *pb.MemberPromoteRequest, opts ...grpc.CallOption) (resp *pb.MemberPromoteResponse, err error) {
	return rcc.cc.MemberPromote(ctx, in, opts...)
}

type retryMetaClient struct {
	mc pb.MetaRPCClient
}

// RetryMetaClient implements a MetaRPCClient.
func RetryMetaClient(conn *grpc.ClientConn) pb.MetaRPCClient {
	return &retryMetaClient{
		mc: pb.NewMetaRPCClient(conn),
	}
}

func (rmc *retryMetaClient) GetMeta(ctx context.Context, in *pb.GetMetaRequest, opts ...grpc.CallOption) (resp *pb.GetMetaResponse, err error) {
	return rmc.mc.GetMeta(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retryMetaClient) ListRunner(ctx context.Context, in *pb.ListRunnerRequest, opts ...grpc.CallOption) (*pb.ListRunnerResponse, error) {
	return rmc.mc.ListRunner(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retryMetaClient) GetRunner(ctx context.Context, in *pb.GetRunnerRequest, opts ...grpc.CallOption) (*pb.GetRunnerResponse, error) {
	return rmc.mc.GetRunner(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retryMetaClient) ListRegion(ctx context.Context, in *pb.ListRegionRequest, opts ...grpc.CallOption) (*pb.ListRegionResponse, error) {
	return rmc.mc.ListRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retryMetaClient) GetRegion(ctx context.Context, in *pb.GetRegionRequest, opts ...grpc.CallOption) (*pb.GetRegionResponse, error) {
	return rmc.mc.GetRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

type retryBpmnClient struct {
	bc pb.BpmnRPCClient
}

// RetryBpmnClient implements a BpmnRPCClient.
func RetryBpmnClient(conn *grpc.ClientConn) pb.BpmnRPCClient {
	return &retryBpmnClient{
		bc: pb.NewBpmnRPCClient(conn),
	}
}

func (rbc *retryBpmnClient) DeployDefinition(ctx context.Context, in *pb.DeployDefinitionRequest, opts ...grpc.CallOption) (resp *pb.DeployDefinitionResponse, err error) {
	return rbc.bc.DeployDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rbc *retryBpmnClient) ListDefinition(ctx context.Context, in *pb.ListDefinitionRequest, opts ...grpc.CallOption) (*pb.ListDefinitionResponse, error) {
	return rbc.bc.ListDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rbc *retryBpmnClient) GetDefinition(ctx context.Context, in *pb.GetDefinitionRequest, opts ...grpc.CallOption) (*pb.GetDefinitionResponse, error) {
	return rbc.bc.GetDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rbc *retryBpmnClient) RemoveDefinition(ctx context.Context, in *pb.RemoveDefinitionRequest, opts ...grpc.CallOption) (*pb.RemoveDefinitionResponse, error) {
	return rbc.bc.RemoveDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rbc *retryBpmnClient) ExecuteDefinition(ctx context.Context, in *pb.ExecuteDefinitionRequest, opts ...grpc.CallOption) (*pb.ExecuteDefinitionResponse, error) {
	return rbc.bc.ExecuteDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rbc *retryBpmnClient) GetProcessInstance(ctx context.Context, in *pb.GetProcessInstanceRequest, opts ...grpc.CallOption) (*pb.GetProcessInstanceResponse, error) {
	return rbc.bc.GetProcessInstance(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}
