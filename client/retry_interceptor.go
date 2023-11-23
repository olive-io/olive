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
	"time"

	"google.golang.org/grpc"
)

type options struct {
	retryPolicy retryPolicy
	max         uint
	backoffFunc backoffFunc
	retryAuth   bool
}

// retryOption is a grpc.CallOption that is local to clientv3's retry interceptor.
type retryOption struct {
	grpc.EmptyCallOption // make sure we implement private after() and before() fields so we don't panic.
	applyFunc            func(opt *options)
}

// backoffFunc denotes a family of functions that control the backoff duration between call retries.
//
// They are called with an identifier of the attempt, and should return a time the system client should
// hold off for. If the time returned is longer than the `context.Context.Deadline` of the request
// the deadline of the request takes precedence and the wait will be interrupted before proceeding
// with the next iteration.
type backoffFunc func(attempt uint) time.Duration

// withRetryPolicy sets the retry policy of this call.
func withRetryPolicy(rp retryPolicy) retryOption {
	return retryOption{applyFunc: func(o *options) {
		o.retryPolicy = rp
	}}
}
