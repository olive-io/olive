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

package storage

import (
	"encoding/binary"
	"io"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"github.com/oliveio/olive/pkg/bytesutil"
	"go.uber.org/zap"
)

const (
	lastIndexKey = "applied_index"
)

type diskKV struct {
	lg *zap.Logger

	getDB GetDB

	clusterID uint64
	nodeID    uint64

	applied uint64
}

func (es *EmbedStorage) NewDiskKV(clusterID uint64, nodeID uint64) sm.IOnDiskStateMachine {
	lg := es.lg
	if lg == nil {
		lg = zap.NewExample()
	}

	kv := &diskKV{
		lg:        lg,
		getDB:     es.NewSession,
		clusterID: clusterID,
		nodeID:    nodeID,
	}
	return kv
}

func (kv *diskKV) Open(stopc <-chan struct{}) (uint64, error) {
	db, cancel := kv.getDB()
	defer cancel()

	key := bytesutil.PathJoin(kv.prefix(), []byte(lastIndexKey))

	value, closer, err := db.Get(key)
	if err != nil {
		return 0, err
	}
	defer closer.Close()

	applied := binary.LittleEndian.Uint64(value)
	atomic.StoreUint64(&kv.applied, applied)
	return applied, nil
}

func (kv *diskKV) Update(entries []sm.Entry) ([]sm.Entry, error) {
	db, cancel := kv.getDB()
	defer cancel()

	wo := &pebble.WriteOptions{Sync: true}

	batch := db.NewBatch()
	lastIdx := uint64(0)
	for i := range entries {
		entry := entries[i]
		entry.Result = sm.Result{Value: uint64(len(entry.Cmd))}
		lastIdx = entry.Index
	}

	if kv.applied >= lastIdx {
		panic("last applied not moving forward")
	}

	appliedKey := bytesutil.PathJoin(kv.prefix(), []byte(lastIndexKey))
	if err := batch.Set(appliedKey, kv.putIntByte(lastIdx), wo); err != nil {
		kv.lg.Error("update applied index", zap.Uint64("index", lastIdx), zap.Error(err))
	}

	if err := batch.Commit(wo); err != nil {
		return entries, err
	}

	atomic.StoreUint64(&kv.applied, lastIdx)

	return entries, nil
}

func (kv *diskKV) Lookup(query interface{}) (interface{}, error) {
	return query, nil
}

func (kv *diskKV) Sync() error {
	return nil
}

func (kv *diskKV) PrepareSnapshot() (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (kv *diskKV) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	return nil
}

func (kv *diskKV) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	return nil
}

func (kv *diskKV) Close() error {
	return nil
}

func (kv *diskKV) prefix() []byte {
	return bytesutil.PathJoin([]byte("/"), kv.putIntByte(kv.clusterID), kv.putIntByte(kv.nodeID))
}

func (kv *diskKV) putIntByte(i uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, i)
	return b
}
