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
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	sm "github.com/lni/dragonboat/v4/statemachine"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

var (
	appliedIndex  = []byte("applied_index")
	defaultEndKey = []byte("\xff")
)

func processInstanceStoreFn(process *pb.ProcessInstance) int64 {
	return 1
}

type Region struct {
	id       uint64
	memberId uint64

	lg *zap.Logger

	w        wait.Wait
	openWait wait.Wait

	cc *Client

	reqIDGen *idutil.Generator

	be backend.IBackend

	rimu       sync.RWMutex
	regionInfo *pb.Region
	metric     *regionMetrics

	processQ *queue.SyncPriorityQueue[*pb.ProcessInstance]

	applied   uint64
	committed uint64
	term      uint64
	leader    uint64

	// config
	heartbeatMs          int64
	warningApplyDuration time.Duration

	changeCMu sync.RWMutex
	changeC   chan struct{}

	readyCMu sync.RWMutex
	readyC   chan struct{}

	stopping <-chan struct{}
}

func (c *Controller) InitDiskStateMachine(shardId, nodeId uint64) sm.IOnDiskStateMachine {
	reqIDGen := idutil.NewGenerator(uint16(nodeId), time.Now())
	processQ := queue.NewSync[*pb.ProcessInstance](processInstanceStoreFn)
	region := &Region{
		id:          shardId,
		memberId:    nodeId,
		lg:          c.Logger,
		w:           wait.New(),
		openWait:    c.regionW,
		cc:          c.NewClient(),
		reqIDGen:    reqIDGen,
		be:          c.be,
		processQ:    processQ,
		heartbeatMs: c.HeartbeatMs,
		regionInfo:  &pb.Region{},
		changeC:     make(chan struct{}),
		readyC:      make(chan struct{}),
		stopping:    c.stopping,
	}
	c.setRegion(region)

	return region
}

func (r *Region) Open(stopc <-chan struct{}) (uint64, error) {
	applyIndex, err := r.readApplyIndex()
	if err != nil {
		r.openWait.Trigger(r.id, err)
		return 0, err
	}

	r.metric, err = newRegionMetrics(r.id)
	if err != nil {
		r.openWait.Trigger(r.id, err)
		return 0, err
	}

	r.setApplied(applyIndex)
	r.openWait.Trigger(r.id, r)

	dm := make(map[string]struct{})
	kvs, _ := r.getRange(definitionPrefix, defaultEndKey, 0)
	for _, kv := range kvs {
		definitionId := path.Dir(string(kv.Key))
		if _, ok := dm[definitionId]; !ok {
			dm[definitionId] = struct{}{}
		}
	}
	r.metric.definition.Add(float64(len(dm)))

	kvs, _ = r.getRange(processPrefix, defaultEndKey, 0)
	for _, kv := range kvs {
		proc := new(pb.ProcessInstance)
		err = proc.Unmarshal(kv.Value)
		if err != nil {
			continue
		}
		if proc.Status == pb.ProcessInstance_Unknown ||
			proc.Status == pb.ProcessInstance_Ok ||
			proc.Status == pb.ProcessInstance_Fail ||
			proc.DefinitionId == "" ||
			proc.DefinitionVersion == 0 {
			continue
		}
		if proc.Status == pb.ProcessInstance_Waiting {
			proc.Status = pb.ProcessInstance_Prepare
		}
		r.processQ.Push(proc)
	}

	go r.heartbeat()
	go r.run()
	return applyIndex, nil
}

func (r *Region) waitUtilLeader() bool {
	for {
		if r.isLeader() {
			return true
		}

		select {
		case <-r.stopping:
			return false
		case <-r.changeNotify():
		}
	}
}

func (r *Region) readyLeader(ctx context.Context) (bool, error) {
	for {
		if r.isLeader() {
			return true, nil
		}
		if r.getLeader() != 0 {
			return false, nil
		}

		select {
		case <-ctx.Done():
			return false, context.Canceled
		case <-r.stopping:
			return false, errors.Wrap(ErrStopped, "ready region leader")
		case <-r.changeNotify():
		}
	}
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
	index := entry.Index

	r.writeApplyIndex(index)
	if index == r.getCommitted() {
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

func (r *Region) Close() error {
	return nil
}

func (r *Region) run() {
	for {
		select {
		case <-r.stopping:
			return
		}
	}
}

func (r *Region) readApplyIndex() (uint64, error) {
	kv, err := r.get(appliedIndex)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	value := kv.Value[:8]

	applied := binary.LittleEndian.Uint64(value)
	return applied, nil
}

func (r *Region) writeApplyIndex(index uint64) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, index)
	r.put(appliedIndex, data, true)
	r.setApplied(index)
}

func (r *Region) get(key []byte) (*pb.KeyValue, error) {
	tx := r.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	key = bytesutil.PathJoin(r.putPrefix(), key)
	value, err := tx.UnsafeGet(buckets.Key, key)
	if err != nil {
		return nil, err
	}
	kv := &pb.KeyValue{
		Key:   key,
		Value: value,
	}

	return kv, err
}

func (r *Region) getRange(startKey, endKey []byte, limit int64) ([]*pb.KeyValue, error) {
	tx := r.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	startKey = bytesutil.PathJoin(r.putPrefix(), startKey)
	if len(endKey) > 0 {
		endKey = bytesutil.PathJoin(r.putPrefix(), endKey)
	}
	keys, values, err := tx.UnsafeRange(buckets.Key, startKey, endKey, limit)
	if err != nil {
		return nil, err
	}
	kvs := make([]*pb.KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = &pb.KeyValue{
			Key:   key,
			Value: values[i],
		}
	}
	return kvs, nil
}

func (r *Region) put(key, value []byte, isSync bool) error {
	key = bytesutil.PathJoin(r.putPrefix(), key)
	tx := r.be.BatchTx()
	tx.Lock()
	if err := tx.UnsafePut(buckets.Key, key, value); err != nil {
		tx.Unlock()
		return err
	}
	tx.Unlock()
	if isSync {
		return tx.Commit()
	}
	return nil
}

func (r *Region) del(key []byte, isSync bool) error {
	key = bytesutil.PathJoin(r.putPrefix(), key)
	tx := r.be.BatchTx()
	tx.Lock()
	if err := tx.UnsafeDelete(buckets.Key, key); err != nil {
		tx.Unlock()
		return err
	}
	tx.Unlock()
	if isSync {
		return tx.Commit()
	}
	return nil
}

func (r *Region) notifyAboutChange() {
	r.changeCMu.Lock()
	changeClose := r.changeC
	r.changeC = make(chan struct{})
	r.changeCMu.Unlock()
	close(changeClose)
}

func (r *Region) changeNotify() <-chan struct{} {
	r.changeCMu.RLock()
	defer r.changeCMu.RUnlock()
	return r.changeC
}

func (r *Region) notifyAboutReady() {
	r.readyCMu.Lock()
	readyClose := r.readyC
	r.readyC = make(chan struct{})
	r.readyCMu.Unlock()
	close(readyClose)
}

func (r *Region) ReadyNotify() <-chan struct{} {
	r.readyCMu.RLock()
	defer r.readyCMu.RUnlock()
	return r.readyC
}

func (r *Region) putPrefix() []byte {
	sb := []byte(fmt.Sprintf("%d", r.id))
	return sb
}

func (r *Region) updateInfo(info *pb.Region) {
	r.rimu.Lock()
	defer r.rimu.Unlock()
	r.regionInfo = info
}

func (r *Region) getInfo() *pb.Region {
	r.rimu.RLock()
	defer r.rimu.RUnlock()
	return r.regionInfo
}

func (r *Region) getID() uint64 {
	return r.id
}

func (r *Region) getMember() uint64 {
	return r.memberId
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

func (r *Region) isLeader() bool {
	lead := r.getLeader()
	return lead != 0 && lead == r.getMember()
}
