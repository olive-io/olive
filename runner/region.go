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

package runner

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

type Region struct {
	shardId uint64
	id      uint64

	lg *zap.Logger

	w        wait.Wait
	openWait wait.Wait

	reqIDGen *idutil.Generator

	be backend.IBackend

	applied   uint64
	committed uint64
	term      uint64
	leader    uint64
}

func (mrg *MultiRaftGroup) InitDiskStateMachine(shardId, nodeId uint64) sm.IOnDiskStateMachine {
	reqIDGen := idutil.NewGenerator(uint16(nodeId), time.Now())
	region := &Region{
		shardId:  shardId,
		id:       nodeId,
		lg:       mrg.lg,
		w:        wait.New(),
		openWait: mrg.w,
		reqIDGen: reqIDGen,
		be:       mrg.be,
	}

	return region
}

func (r *Region) Open(stopc <-chan struct{}) (uint64, error) {
	applyIndex, err := r.readApplyIndex()
	if err != nil {
		r.openWait.Trigger(r.id, err)
		return 0, err
	}
	r.setApplied(applyIndex)
	r.openWait.Trigger(r.id, nil)

	return applyIndex, nil
}

func (r *Region) readApplyIndex() (uint64, error) {
	tx := r.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	applyKey := r.putPrefix()
	value, err := tx.UnsafeGet(buckets.Key, applyKey)
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return 0, err
	}

	applied := binary.LittleEndian.Uint64(value)
	return applied, nil
}

func (r *Region) Update(entries []sm.Entry) ([]sm.Entry, error) {
	var committed uint64
	if length := len(entries); length > 0 {
		committed = entries[length-1].Index
		r.setCommitted(committed)
	}

	for i := range entries {
		ent := entries[i]
		r.applyEntry(ent)
	}

	return entries, nil
}

func (r *Region) applyEntry(entry sm.Entry) {
	r.setApplied(entry.Index)

	if entry.Index == r.getCommitted() {
		r.w.Trigger(r.id, nil)
	}
}

func (r *Region) Lookup(query interface{}) (interface{}, error) {
	return query, nil
}

func (r *Region) Sync() error {
	return nil
}

func (r *Region) PrepareSnapshot() (interface{}, error) {
	snap := r.be.Snapshot()
	return snap, nil
}

func (r *Region) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	snap := ctx.(backend.ISnapshot)
	prefix := r.putPrefix()
	_, err := snap.WriteTo(writer, bytesutil.PathJoin(buckets.Key.Name(), prefix))
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

func (r *Region) Close() error {
	return nil
}

func (r *Region) putPrefix() []byte {
	sb := []byte(fmt.Sprintf("%d", r.shardId))
	return sb
}

func (r *Region) setApplied(applied uint64) {
	atomic.StoreUint64(&r.applied, applied)
}

func (r *Region) getApplied() uint64 {
	return atomic.LoadUint64(&r.applied)
}

func (r *Region) setCommitted(committed uint64) {
	atomic.StoreUint64(&r.committed, committed)
}

func (r *Region) getCommitted() uint64 {
	return atomic.LoadUint64(&r.committed)
}

func (r *Region) setTerm(term uint64) {
	atomic.StoreUint64(&r.term, term)
}

func (r *Region) getTerm() uint64 {
	return atomic.LoadUint64(&r.term)
}

func (r *Region) setLeader(leader uint64) {
	atomic.StoreUint64(&r.leader, leader)
}

func (r *Region) getLeader() uint64 {
	return atomic.LoadUint64(&r.leader)
}
