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

package raft

import "errors"

var (
	ErrNoRegion               = errors.New("region: not found")
	ErrRegionReplicaAdded     = errors.New("region: replica already added")
	ErrRaftAddress            = errors.New("invalid raft address")
	ErrStopped                = errors.New("region has stopped; skipping request")
	ErrCanceled               = errors.New("region: request cancelled")
	ErrTimeout                = errors.New("region: request timed out")
	ErrTimeoutDueToLeaderFail = errors.New("region: request timed out, possibly due to previous leader failure")
	ErrRequestQuery           = errors.New("region: request is invalid query")
	ErrTooManyRequests        = errors.New("region: too many requests")
	ErrProcessExecuted        = errors.New("region: process already executed")
)
