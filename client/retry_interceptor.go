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
