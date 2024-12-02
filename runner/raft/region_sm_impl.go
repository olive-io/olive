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

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	xpath "github.com/olive-io/olive/x/path"
)

func (r *Region) Open(stopc <-chan struct{}) (uint64, error) {
	return r.initial(stopc)
}

func (r *Region) Update(entries []sm.Entry) ([]sm.Entry, error) {
	var committed uint64
	if length := len(entries); length > 0 {
		committed = entries[length-1].Index
		r.setCommitted(committed)
	}

	for i := range entries {
		if entries[i].Index <= r.getApplied() {
			continue
		}
		r.applyEntry(entries[i])
	}

	return entries, nil
}

func (r *Region) Lookup(query interface{}) (interface{}, error) {
	raftReq, ok := query.(*pb.RaftInternalRequest)
	if !ok {
		return nil, ErrRequestQuery
	}

	ar := r.applyBase.Apply(context.TODO(), raftReq)
	if ar.err != nil {
		return nil, ar.err
	}
	return ar, nil
}

func (r *Region) Sync() error {
	return nil
}

func (r *Region) Close() error {
	return nil
}

func (r *Region) PrepareSnapshot() (interface{}, error) {
	snap := r.be.Snapshot()
	return snap, nil
}

func (r *Region) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	snap := ctx.(backend.ISnapshot)
	prefix := xpath.Join(buckets.Key.Name(), r.putPrefix())
	_, err := snap.WriteTo(writer, prefix)
	if err != nil {
		return err
	}

	return nil
}

func (r *Region) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	err := r.be.Recover(reader)
	if err != nil {
		return err
	}

	applyIndex, err := r.readApplyIndex()
	if err != nil {
		return err
	}
	r.setApplied(applyIndex)

	return nil
}
