// Copyright 2023 Lack (xingyys@gmail.com).
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

	pb "github.com/olive-io/olive/api/olivepb"
	"google.golang.org/grpc"
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

type retryBpmnClient struct {
	bc pb.BpmnRPCClient
}

// RetryBpmnClient implements a BpmnRPCClient.
func RetryBpmnClient(c *Client) pb.BpmnRPCClient {
	return &retryBpmnClient{
		bc: pb.NewBpmnRPCClient(c.ActiveConnection()),
	}
}

func (rbc *retryBpmnClient) DeployDefinition(ctx context.Context, in *pb.DeployDefinitionRequest, opts ...grpc.CallOption) (resp *pb.DeployDefinitionResponse, err error) {
	return rbc.bc.DeployDefinition(ctx, in, withRetryPolicy(repeatable))
}

func (rbc *retryBpmnClient) ListDefinition(ctx context.Context, in *pb.ListDefinitionRequest, opts ...grpc.CallOption) (*pb.ListDefinitionResponse, error) {
	return rbc.bc.ListDefinition(ctx, in, withRetryPolicy(repeatable))
}

func (rbc *retryBpmnClient) GetDefinition(ctx context.Context, in *pb.GetDefinitionRequest, opts ...grpc.CallOption) (*pb.GetDefinitionResponse, error) {
	return rbc.bc.GetDefinition(ctx, in, withRetryPolicy(repeatable))
}

func (rbc *retryBpmnClient) RemoveDefinition(ctx context.Context, in *pb.RemoveDefinitionRequest, opts ...grpc.CallOption) (*pb.RemoveDefinitionResponse, error) {
	return rbc.bc.RemoveDefinition(ctx, in, withRetryPolicy(repeatable))
}

func (rbc *retryBpmnClient) ExecuteDefinition(ctx context.Context, in *pb.ExecuteDefinitionRequest, opts ...grpc.CallOption) (*pb.ExecuteDefinitionResponse, error) {
	return rbc.bc.ExecuteDefinition(ctx, in, withRetryPolicy(repeatable))
}

type retryRunnerClient struct {
	rc pb.RunnerRPCClient
}

// RetryRunnerClient implements a RunnerRPCClient.
func RetryRunnerClient(c *Client) pb.RunnerRPCClient {
	return &retryRunnerClient{
		rc: pb.NewRunnerRPCClient(c.ActiveConnection()),
	}
}

func (rrc *retryRunnerClient) RegistryRunner(ctx context.Context, in *pb.RegistryRunnerRequest, opts ...grpc.CallOption) (*pb.RegistryRunnerResponse, error) {
	return rrc.rc.RegistryRunner(ctx, in, withRetryPolicy(repeatable))
}

func (rrc *retryRunnerClient) ReportRunner(ctx context.Context, in *pb.ReportRunnerRequest, opts ...grpc.CallOption) (*pb.ReportRunnerResponse, error) {
	return rrc.rc.ReportRunner(ctx, in, withRetryPolicy(repeatable))
}
