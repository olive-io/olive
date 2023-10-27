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

package errors

import "errors"

var (
	ErrUnknownMethod                 = errors.New("olive: unknown method")
	ErrStopped                       = errors.New("olive: server stopped")
	ErrCanceled                      = errors.New("olive: request cancelled")
	ErrTimeout                       = errors.New("olive: request timed out")
	ErrTimeoutDueToLeaderFail        = errors.New("olive: request timed out, possibly due to previous leader failure")
	ErrTimeoutDueToConnectionLost    = errors.New("olive: request timed out, possibly due to connection lost")
	ErrTimeoutLeaderTransfer         = errors.New("olive: request timed out, leader transfer took too long")
	ErrTimeoutWaitAppliedIndex       = errors.New("olive: request timed out, waiting for the applied index took too long")
	ErrLeaderChanged                 = errors.New("olive: leader changed")
	ErrNotEnoughStartedMembers       = errors.New("olive: re-configuration failed due to not enough started members")
	ErrLearnerNotReady               = errors.New("olive: can only promote a learner member which is in sync with leader")
	ErrNoLeader                      = errors.New("olive: no leader")
	ErrNotLeader                     = errors.New("olive: not leader")
	ErrRequestTooLarge               = errors.New("olive: request is too large")
	ErrNoSpace                       = errors.New("olive: no space")
	ErrTooManyRequests               = errors.New("olive: too many requests")
	ErrUnhealthy                     = errors.New("olive: unhealthy cluster")
	ErrKeyNotFound                   = errors.New("olive: key not found")
	ErrCorrupt                       = errors.New("olive: corrupt cluster")
	ErrBadLeaderTransferee           = errors.New("olive: bad leader transferee")
	ErrClusterVersionUnavailable     = errors.New("olive: cluster version not found during downgrade")
	ErrWrongDowngradeVersionFormat   = errors.New("olive: wrong downgrade target version format")
	ErrInvalidDowngradeTargetVersion = errors.New("olive: invalid downgrade target version")
	ErrDowngradeInProcess            = errors.New("olive: cluster has a downgrade job in progress")
	ErrNoInflightDowngrade           = errors.New("olive: no inflight downgrade job")
)
