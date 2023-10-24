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
	"fmt"
	"io"
	"path"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"go.uber.org/zap"
)

const (
	lastIndexKey = "applied_index"
)

type KV struct {
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

	kv := &KV{
		lg:        lg,
		getDB:     es.NewSession,
		clusterID: clusterID,
		nodeID:    nodeID,
	}
	return kv
}

func (kv *KV) Open(stopc <-chan struct{}) (uint64, error) {
	db, cancel := kv.getDB()
	defer cancel()

	key := path.Join(kv.prefix(), lastIndexKey)

	value, closer, err := db.Get([]byte(key))
	if err != nil {
		return 0, err
	}
	defer closer.Close()

	applied := binary.LittleEndian.Uint64(value)
	atomic.StoreUint64(&kv.applied, applied)
	return applied, nil
}

func (kv *KV) Update(entries []sm.Entry) ([]sm.Entry, error) {
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

	appliedKey := path.Join(kv.prefix(), lastIndexKey)
	if err := batch.Set([]byte(appliedKey), kv.putIntByte(lastIdx), wo); err != nil {
		kv.lg.Error("update applied index", zap.Uint64("index", lastIdx), zap.Error(err))
	}

	if err := batch.Commit(wo); err != nil {
		return entries, err
	}

	atomic.StoreUint64(&kv.applied, lastIdx)

	return entries, nil
}

func (kv *KV) Lookup(query interface{}) (interface{}, error) {
	return query, nil
}

func (kv *KV) Sync() error {
	return nil
}

func (kv *KV) PrepareSnapshot() (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (kv *KV) SaveSnapshot(ctx interface{}, writer io.Writer, done <-chan struct{}) error {
	return nil
}

func (kv *KV) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	return nil
}

func (kv *KV) Close() error {
	return nil
}

func (kv *KV) prefix() string {
	return fmt.Sprintf("/%d/%d", kv.clusterID, kv.nodeID)
}

func (kv *KV) putIntByte(i uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, i)
	return b
}
