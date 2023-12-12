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
	"io"

	sm "github.com/lni/dragonboat/v4/statemachine"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
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
	prefix := bytesutil.PathJoin(buckets.Key.Name(), r.putPrefix())
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
