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

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// isSafeRetryImmutableRPC returns "true" when an immutable request is safe for retry.
//
// immutable requests (e.g. Get) should be retried unless it's
// an obvious server-side error (e.g. rpctypes.ErrRequestTooLarge).
//
// Returning "false" means retry should stop, since client cannot
// handle itself even with retries.
func isSafeRetryImmutableRPC(err error) bool {
	eErr := rpctypes.Error(err)
	if serverErr, ok := eErr.(rpctypes.OliveError); ok && serverErr.Code() != codes.Unavailable {
		// interrupted by non-transient server-side or gRPC-side error
		// client cannot handle itself (e.g. rpctypes.ErrCompacted)
		return false
	}
	// only retry if unavailable
	ev, ok := status.FromError(err)
	if !ok {
		// all errors from RPC is typed "grpc/status.(*statusError)"
		// (ref. https://github.com/grpc/grpc-go/pull/1782)
		//
		// if the error type is not "grpc/status.(*statusError)",
		// it could be from "Dial"
		// TODO: do not retry for now
		// ref. https://github.com/grpc/grpc-go/issues/1581
		return false
	}
	return ev.Code() == codes.Unavailable
}

// isSafeRetryMutableRPC returns "true" when a mutable request is safe for retry.
//
// mutable requests (e.g. Put, Delete, Txn) should only be retried
// when the status code is codes.Unavailable when initial connection
// has not been established (no endpoint is up).
//
// Returning "false" means retry should stop, otherwise it violates
// write-at-most-once semantics.
func isSafeRetryMutableRPC(err error) bool {
	if ev, ok := status.FromError(err); ok && ev.Code() != codes.Unavailable {
		// not safe for mutable RPCs
		// e.g. interrupted by non-transient error that client cannot handle itself,
		// or transient error while the connection has already been established
		return false
	}
	desc := rpctypes.ErrorDesc(err)
	return desc == "there is no address available" || desc == "there is no connection available"
}

type retryDefinitionClient struct {
	kc api.DefinitionRPCClient
}

// RetryDefinitionClient implements a DefinitionClient.
func RetryDefinitionClient(c *Client) api.DefinitionRPCClient {
	return &retryDefinitionClient{
		kc: api.NewDefinitionRPCClient(c.conn),
	}
}

func (rc *retryDefinitionClient) DeployDefinition(ctx context.Context, in *api.DeployDefinitionRequest, opts ...grpc.CallOption) (resp *api.DeployDefinitionResponse, err error) {
	return rc.kc.DeployDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryDefinitionClient) ListDefinition(ctx context.Context, in *api.ListDefinitionRequest, opts ...grpc.CallOption) (resp *api.ListDefinitionResponse, err error) {
	return rc.kc.ListDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryDefinitionClient) GetDefinition(ctx context.Context, in *api.GetDefinitionRequest, opts ...grpc.CallOption) (resp *api.GetDefinitionResponse, err error) {
	return rc.kc.GetDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryDefinitionClient) RemoveDefinition(ctx context.Context, in *api.RemoveDefinitionRequest, opts ...grpc.CallOption) (resp *api.RemoveDefinitionResponse, err error) {
	return rc.kc.RemoveDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}

func (rc *retryDefinitionClient) ExecuteDefinition(ctx context.Context, in *api.ExecuteDefinitionRequest, opts ...grpc.CallOption) (resp *api.ExecuteDefinitionResponse, err error) {
	return rc.kc.ExecuteDefinition(ctx, in, append(opts, withRetryPolicy(repeatable))...)
}
