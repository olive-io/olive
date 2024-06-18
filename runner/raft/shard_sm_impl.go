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
	"io"

	sm "github.com/lni/dragonboat/v4/statemachine"

	pb "github.com/olive-io/olive/apis/pb/olive"

	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
)

func (s *Shard) Open(stopc <-chan struct{}) (uint64, error) {
	return s.initial(stopc)
}

func (s *Shard) Update(entries []sm.Entry) ([]sm.Entry, error) {
	var committed uint64
	if length := len(entries); length > 0 {
		committed = entries[length-1].Index
		s.setCommitted(committed)
	}

	for i := range entries {
		if entries[i].Index <= s.getApplied() {
			continue
		}
		s.applyEntry(entries[i])
	}

	return entries, nil
}

func (s *Shard) Lookup(query interface{}) (interface{}, error) {
	raftReq, ok := query.(*pb.RaftInternalRequest)
	if !ok {
		return nil, ErrRequestQuery
	}

	ar := s.applyBase.Apply(context.TODO(), raftReq)
	if ar.err != nil {
		return nil, ar.err
	}
	return ar, nil
}

func (s *Shard) Sync() error {
	return nil
}

func (s *Shard) Close() error {
	return nil
}

func (s *Shard) PrepareSnapshot() (interface{}, error) {
	snap := s.be.Snapshot()
	return snap, nil
}

func (s *Shard) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	snap := ctx.(backend.ISnapshot)
	prefix := bytesutil.PathJoin(buckets.Key.Name(), s.putPrefix())
	_, err := snap.WriteTo(writer, prefix)
	if err != nil {
		return err
	}

	return nil
}

func (s *Shard) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	err := s.be.Recover(reader)
	if err != nil {
		return err
	}

	applyIndex, err := s.readApplyIndex()
	if err != nil {
		return err
	}
	s.setApplied(applyIndex)

	return nil
}
