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

import (
	"context"

	"github.com/lni/dragonboat/v4/raftio"

	pb "github.com/olive-io/olive/api/olivepb"
)

type leaderTrace raftio.LeaderInfo

func (t leaderTrace) TraceInterface() {}

type RegionStatTrace pb.RegionStat

func (t RegionStatTrace) TraceInterface() {}

type proposeTrace struct {
	ctx    context.Context
	region uint64
	data   []byte

	ech chan error
}

func newProposeTrace(ctx context.Context, region uint64, data []byte, ech chan error) *proposeTrace {
	return &proposeTrace{
		ctx:    ctx,
		region: region,
		data:   data,
		ech:    ech,
	}
}

func (t *proposeTrace) Trigger(err error) {
	t.ech <- err
}

func (t *proposeTrace) TraceInterface() {}

type readTrace struct {
	ctx    context.Context
	region uint64
	query  *pb.RaftInternalRequest

	arch chan *applyResult
	ech  chan error
}

func newReadTrace(ctx context.Context, region uint64, query *pb.RaftInternalRequest) *readTrace {
	return &readTrace{
		ctx:    ctx,
		region: region,
		query:  query,
		arch:   make(chan *applyResult, 1),
		ech:    make(chan error, 1),
	}
}

func (t *readTrace) Write(ar *applyResult, err error) {
	if ar != nil {
		t.arch <- ar
	}
	if err != nil {
		t.ech <- err
	}
}

func (t *readTrace) Trigger() (<-chan *applyResult, <-chan error) {
	return t.arch, t.ech
}

func (t *readTrace) Close() {
	close(t.arch)
	close(t.ech)
}

func (t *readTrace) TraceInterface() {}
