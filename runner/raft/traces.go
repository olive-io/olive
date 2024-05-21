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

import (
	"context"

	"github.com/lni/dragonboat/v4/raftio"

	pb "github.com/olive-io/olive/apis/pb/olive"
)

type leaderTrace raftio.LeaderInfo

func (t leaderTrace) TraceInterface() {}

type RegionStatTrace struct {
	Stat *pb.RegionStat
}

func (t *RegionStatTrace) TraceInterface() {}

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
