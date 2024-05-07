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

	ErrNotFound = errors.New("key: not found")
)
