/*
Copyright 2023 The olive Authors

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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/rpc/monpb"
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

type retrySystemClient struct {
	rpc pb.SystemRPCClient
}

// RetryPlaneClient implements a PlaneRPCClient.
func RetryPlaneClient(conn *grpc.ClientConn) pb.SystemRPCClient {
	return &retrySystemClient{
		rpc: pb.NewSystemRPCClient(conn),
	}
}

func (rmc *retrySystemClient) GetCluster(ctx context.Context, in *pb.GetClusterRequest, opts ...grpc.CallOption) (resp *pb.GetClusterResponse, err error) {
	return rmc.rpc.GetCluster(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retrySystemClient) ListRunners(ctx context.Context, in *pb.ListRunnersRequest, opts ...grpc.CallOption) (*pb.ListRunnersResponse, error) {
	return rmc.rpc.ListRunners(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rmc *retrySystemClient) GetRunner(ctx context.Context, in *pb.GetRunnerRequest, opts ...grpc.CallOption) (resp *pb.GetRunnerResponse, err error) {
	return rmc.rpc.GetRunner(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

//func (rmc *retryPlaneClient) ListRegion(ctx context.Context, in *pb.ListRegionRequest, opts ...grpc.CallOption) (*pb.ListRegionResponse, error) {
//	return rmc.mc.ListRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
//}
//
//func (rmc *retryPlaneClient) GetRegion(ctx context.Context, in *pb.GetRegionRequest, opts ...grpc.CallOption) (*pb.GetRegionResponse, error) {
//	return rmc.mc.GetRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
//}

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

//func (rbc *retryBpmnClient) GetProcessInstance(ctx context.Context, in *pb.GetProcessInstanceRequest, opts ...grpc.CallOption) (resp *pb.GetProcessInstanceResponse, err error) {
//	//return rbc.bc.GetProcessInstance(ctx, in, append(opts, withRetryPolicy(repeatable))...)
//	return
//}
