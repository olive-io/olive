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

	pb "github.com/olive-io/olive/apis/pb/olive"
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
	cc pb.MetaClusterRPCClient
}

// RetryClusterClient implements a ClusterClient.
func RetryClusterClient(conn *grpc.ClientConn) pb.MetaClusterRPCClient {
	return &retryClusterClient{
		cc: pb.NewMetaClusterRPCClient(conn),
	}
}

func (rc *retryClusterClient) MemberList(ctx context.Context, in *pb.MemberListRequest, opts ...grpc.CallOption) (resp *pb.MemberListResponse, err error) {
	return rc.cc.MemberList(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryClusterClient) MemberAdd(ctx context.Context, in *pb.MemberAddRequest, opts ...grpc.CallOption) (resp *pb.MemberAddResponse, err error) {
	return rc.cc.MemberAdd(ctx, in, opts...)
}

func (rc *retryClusterClient) MemberRemove(ctx context.Context, in *pb.MemberRemoveRequest, opts ...grpc.CallOption) (resp *pb.MemberRemoveResponse, err error) {
	return rc.cc.MemberRemove(ctx, in, opts...)
}

func (rc *retryClusterClient) MemberUpdate(ctx context.Context, in *pb.MemberUpdateRequest, opts ...grpc.CallOption) (resp *pb.MemberUpdateResponse, err error) {
	return rc.cc.MemberUpdate(ctx, in, opts...)
}

func (rc *retryClusterClient) MemberPromote(ctx context.Context, in *pb.MemberPromoteRequest, opts ...grpc.CallOption) (resp *pb.MemberPromoteResponse, err error) {
	return rc.cc.MemberPromote(ctx, in, opts...)
}

type retryMetaRunnerClient struct {
	mc pb.MetaRunnerRPCClient
	md map[string]string
}

// RetryMetaRunnerClient implements a MetaRunnerRPCClient.
func RetryMetaRunnerClient(conn *grpc.ClientConn) pb.MetaRunnerRPCClient {
	return &retryMetaRunnerClient{
		mc: pb.NewMetaRunnerRPCClient(conn),
	}
}

func (rc *retryMetaRunnerClient) ListRunner(ctx context.Context, in *pb.ListRunnerRequest, opts ...grpc.CallOption) (*pb.ListRunnerResponse, error) {
	return rc.mc.ListRunner(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryMetaRunnerClient) GetRunner(ctx context.Context, in *pb.GetRunnerRequest, opts ...grpc.CallOption) (*pb.GetRunnerResponse, error) {
	return rc.mc.GetRunner(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

type retryMetaRegionClient struct {
	mc pb.MetaRegionRPCClient
	md map[string]string
}

// RetryMetaRegionClient implements a MetaRegionRPCClient
func RetryMetaRegionClient(conn *grpc.ClientConn) pb.MetaRegionRPCClient {
	return &retryMetaRegionClient{
		mc: pb.NewMetaRegionRPCClient(conn),
	}
}

func (rc *retryMetaRegionClient) ListRegion(ctx context.Context, in *pb.ListRegionRequest, opts ...grpc.CallOption) (*pb.ListRegionResponse, error) {
	return rc.mc.ListRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryMetaRegionClient) GetRegion(ctx context.Context, in *pb.GetRegionRequest, opts ...grpc.CallOption) (*pb.GetRegionResponse, error) {
	return rc.mc.GetRegion(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

type retryAuthClient struct {
	ac pb.AuthRPCClient
	md map[string]string
}

// RetryAuthClient implements a AuthRPCClient.
func RetryAuthClient(conn *grpc.ClientConn) pb.AuthRPCClient {
	return &retryAuthClient{
		ac: pb.NewAuthRPCClient(conn),
	}
}

func (rc *retryAuthClient) ListRole(ctx context.Context, in *pb.ListRoleRequest, opts ...grpc.CallOption) (*pb.ListRoleResponse, error) {
	return rc.ac.ListRole(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryAuthClient) GetRole(ctx context.Context, in *pb.GetRoleRequest, opts ...grpc.CallOption) (*pb.GetRoleResponse, error) {
	return rc.ac.GetRole(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryAuthClient) CreateRole(ctx context.Context, in *pb.CreateRoleRequest, opts ...grpc.CallOption) (*pb.CreateRoleResponse, error) {
	return rc.ac.CreateRole(ctx, in, opts...)
}

func (rc *retryAuthClient) UpdateRole(ctx context.Context, in *pb.UpdateRoleRequest, opts ...grpc.CallOption) (*pb.UpdateRoleResponse, error) {
	return rc.ac.UpdateRole(ctx, in, opts...)
}

func (rc *retryAuthClient) RemoveRole(ctx context.Context, in *pb.RemoveRoleRequest, opts ...grpc.CallOption) (*pb.RemoveRoleResponse, error) {
	return rc.ac.RemoveRole(ctx, in, opts...)
}

func (rc *retryAuthClient) ListUser(ctx context.Context, in *pb.ListUserRequest, opts ...grpc.CallOption) (*pb.ListUserResponse, error) {
	return rc.ac.ListUser(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryAuthClient) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	return rc.ac.GetUser(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryAuthClient) CreateUser(ctx context.Context, in *pb.CreateUserRequest, opts ...grpc.CallOption) (*pb.CreateUserResponse, error) {
	return rc.ac.CreateUser(ctx, in, opts...)
}

func (rc *retryAuthClient) UpdateUser(ctx context.Context, in *pb.UpdateUserRequest, opts ...grpc.CallOption) (*pb.UpdateUserResponse, error) {
	return rc.ac.UpdateUser(ctx, in, opts...)
}

func (rc *retryAuthClient) RemoveUser(ctx context.Context, in *pb.RemoveUserRequest, opts ...grpc.CallOption) (*pb.RemoveUserResponse, error) {
	return rc.ac.RemoveUser(ctx, in, opts...)
}

func (rc *retryAuthClient) Authenticate(ctx context.Context, in *pb.AuthenticateRequest, opts ...grpc.CallOption) (*pb.AuthenticateResponse, error) {
	return rc.ac.Authenticate(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

type retryRbacClient struct {
	rc pb.RbacRPCClient
	md map[string]string
}

// RetryRbacClient implements a RbacRPCClient.
func RetryRbacClient(conn *grpc.ClientConn) pb.RbacRPCClient {
	return &retryRbacClient{
		rc: pb.NewRbacRPCClient(conn),
	}
}

func (rc *retryRbacClient) ListPolicy(ctx context.Context, in *pb.ListPolicyRequest, opts ...grpc.CallOption) (*pb.ListPolicyResponse, error) {
	return rc.rc.ListPolicy(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryRbacClient) AddPolicy(ctx context.Context, in *pb.AddPolicyRequest, opts ...grpc.CallOption) (*pb.AddPolicyResponse, error) {
	return rc.rc.AddPolicy(ctx, in, opts...)
}

func (rc *retryRbacClient) RemovePolicy(ctx context.Context, in *pb.RemovePolicyRequest, opts ...grpc.CallOption) (*pb.RemovePolicyResponse, error) {
	return rc.rc.RemovePolicy(ctx, in, opts...)
}

func (rc *retryRbacClient) AddGroupPolicy(ctx context.Context, in *pb.AddGroupPolicyRequest, opts ...grpc.CallOption) (*pb.AddGroupPolicyResponse, error) {
	return rc.rc.AddGroupPolicy(ctx, in, opts...)
}

func (rc *retryRbacClient) RemoveGroupPolicy(ctx context.Context, in *pb.RemoveGroupPolicyRequest, opts ...grpc.CallOption) (*pb.RemoveGroupPolicyResponse, error) {
	return rc.rc.RemoveGroupPolicy(ctx, in, opts...)
}

func (rc *retryRbacClient) Admit(ctx context.Context, in *pb.AdmitRequest, opts ...grpc.CallOption) (*pb.AdmitResponse, error) {
	return rc.rc.Admit(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

type retryBpmnClient struct {
	bc pb.BpmnRPCClient
	md map[string]string
}

// RetryBpmnClient implements a BpmnRPCClient.
func RetryBpmnClient(conn *grpc.ClientConn) pb.BpmnRPCClient {
	return &retryBpmnClient{
		bc: pb.NewBpmnRPCClient(conn),
	}
}

func (rc *retryBpmnClient) DeployDefinition(ctx context.Context, in *pb.DeployDefinitionRequest, opts ...grpc.CallOption) (resp *pb.DeployDefinitionResponse, err error) {
	return rc.bc.DeployDefinition(ctx, in, opts...)
}

func (rc *retryBpmnClient) ListDefinition(ctx context.Context, in *pb.ListDefinitionRequest, opts ...grpc.CallOption) (*pb.ListDefinitionResponse, error) {
	return rc.bc.ListDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryBpmnClient) GetDefinition(ctx context.Context, in *pb.GetDefinitionRequest, opts ...grpc.CallOption) (*pb.GetDefinitionResponse, error) {
	return rc.bc.GetDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryBpmnClient) RemoveDefinition(ctx context.Context, in *pb.RemoveDefinitionRequest, opts ...grpc.CallOption) (*pb.RemoveDefinitionResponse, error) {
	return rc.bc.RemoveDefinition(ctx, in, opts...)
}

func (rc *retryBpmnClient) ExecuteDefinition(ctx context.Context, in *pb.ExecuteDefinitionRequest, opts ...grpc.CallOption) (*pb.ExecuteDefinitionResponse, error) {
	return rc.bc.ExecuteDefinition(ctx, in, opts...)
}

func (rc *retryBpmnClient) ListProcessInstances(ctx context.Context, in *pb.ListProcessInstancesRequest, opts ...grpc.CallOption) (*pb.ListProcessInstancesResponse, error) {
	return rc.bc.ListProcessInstances(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryBpmnClient) GetProcessInstance(ctx context.Context, in *pb.GetProcessInstanceRequest, opts ...grpc.CallOption) (*pb.GetProcessInstanceResponse, error) {
	return rc.bc.GetProcessInstance(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}
