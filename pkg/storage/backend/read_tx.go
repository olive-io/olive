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

package backend

import (
	"bytes"
	"fmt"
	"math"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/oliveio/olive/pkg/bytesutil"
)

type IReadTx interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()

	UnsafeRange(bucket IBucket, key, endKey []byte, limit int64) (keys [][]byte, vals [][]byte, err error)
	UnsafeForEach(bucket IBucket, visitor func(k, v []byte) error) error
}

func unsafeRange(iter *pebble.Iterator, bucket IBucket, key, endKey []byte, limit int64) (keys [][]byte, vs [][]byte, err error) {
	if limit <= 0 {
		limit = math.MaxInt64
	}

	key = bytesutil.PathJoin(bucket.Name(), key)
	endKey = bytesutil.PathJoin(bucket.Name(), endKey)

	var isMatch func(b []byte) bool
	if len(endKey) > 0 {
		isMatch = func(b []byte) bool { return bytes.Compare(b, endKey) < 0 }
	} else {
		isMatch = func(b []byte) bool { return bytes.Equal(b, key) }
		limit = 1
	}

	for iter.SeekGE(key); iteratorIsValid(iter) && isMatch(iter.Key()); iter.Next() {
		var value []byte
		value, err = iter.ValueAndErr()
		if err != nil {
			return
		}

		keys = append(keys, iter.Key())
		vs = append(vs, value)

		if limit == int64(len(keys)) {
			break
		}
	}

	return keys, vs, nil
}

func unsafeForEach(tx *pebble.Batch, bucket IBucket, visitor func(k, v []byte) error) error {
	options := &pebble.IterOptions{}
	iter, err := tx.NewIter(options)
	if err != nil {
		return err
	}

	prefix := bytesutil.PathJoin(bucket.Name())
	for iter.SeekPrefixGE(prefix); iteratorIsValid(iter); iter.Next() {
		key := iter.Key()
		var value []byte
		if value, err = iter.ValueAndErr(); err != nil {
			return err
		}

		if err = visitor(key, value); err != nil {
			return err
		}
	}

	return nil
}

func iteratorIsValid(iter *pebble.Iterator) bool {
	v := iter.Valid()
	if err := iter.Error(); err != nil {
		panic(fmt.Sprintf("do not use iterator: %+v", err))
	}
	return v
}

// Base type for readTx and concurrentReadTx to eliminate duplicate functions between these
type baseReadTx struct {
	// mu protects accesses to the txReadBuffer
	mu  sync.RWMutex
	buf txReadBuffer

	// TODO: group and encapsulate {txMu, tx, buckets, txWg}, as they share the same lifecycle.
	// txMu protects accesses to buckets and tx on Range requests.
	txMu    *sync.RWMutex
	tx      *pebble.Batch
	buckets map[BucketID]*localBucket
	// txWg protects tx from being rolled back at the end of a batch interval until all reads using this tx are done.
	txWg *sync.WaitGroup
}

func (tx *baseReadTx) UnsafeForEach(bucket IBucket, visitor func(k, v []byte) error) error {
	dups := make(map[string]struct{})
	getDups := func(k, v []byte) error {
		dups[string(k)] = struct{}{}
		return nil
	}
	visitNoDup := func(k, v []byte) error {
		if _, ok := dups[string(k)]; ok {
			return nil
		}
		return visitor(k, v)
	}
	if err := tx.buf.ForEach(bucket, getDups); err != nil {
		return err
	}
	tx.txMu.Lock()
	err := unsafeForEach(tx.tx, bucket, visitNoDup)
	tx.txMu.Unlock()
	if err != nil {
		return err
	}
	return tx.buf.ForEach(bucket, visitor)
}

func (tx *baseReadTx) UnsafeRange(bucketType IBucket, key, endKey []byte, limit int64) ([][]byte, [][]byte, error) {
	if endKey == nil {
		// forbid duplicates for single keys
		limit = 1
	}
	if limit <= 0 {
		limit = math.MaxInt64
	}
	if limit > 1 && !bucketType.IsSafeRangeBucket() {
		panic("do not use unsafeRange on non-keys bucket")
	}
	keys, vals, err := tx.buf.Range(bucketType, key, endKey, limit)
	if err != nil {
		return nil, nil, err
	}
	if int64(len(keys)) == limit {
		return keys, vals, nil
	}

	// find/cache bucket
	bn := bucketType.ID()
	tx.txMu.RLock()
	bucket, ok := tx.buckets[bn]
	tx.txMu.RUnlock()
	lockHeld := false
	if !ok {
		tx.txMu.Lock()
		lockHeld = true
		bucket, _ = readBucket(tx.tx, bucketType.Name())
		tx.buckets[bn] = bucket
	}

	// ignore missing bucket since may have been created in this batch
	if bucket == nil {
		if lockHeld {
			tx.txMu.Unlock()
		}
		return keys, vals, nil
	}
	if !lockHeld {
		tx.txMu.Lock()
	}

	options := &pebble.IterOptions{}
	c, err := tx.tx.NewIter(options)
	if err != nil {
		tx.txMu.Unlock()
		return nil, nil, err
	}
	tx.txMu.Unlock()

	k2, v2, err := unsafeRange(c, bucket, key, endKey, limit-int64(len(keys)))
	if err != nil {
		return nil, nil, err
	}
	return append(k2, keys...), append(v2, vals...), nil
}

type readTx struct {
	baseReadTx
}

func (rt *readTx) Lock()    { rt.mu.Lock() }
func (rt *readTx) Unlock()  { rt.mu.Unlock() }
func (rt *readTx) RLock()   { rt.mu.RLock() }
func (rt *readTx) RUnlock() { rt.mu.RUnlock() }

func (rt *readTx) reset() {
	rt.buf.reset()
	rt.tx = nil
	rt.txWg = new(sync.WaitGroup)
}

type concurrentReadTx struct {
	baseReadTx
}

func (rt *concurrentReadTx) Lock()   {}
func (rt *concurrentReadTx) Unlock() {}

// RLock is no-op. concurrentReadTx does not need to be locked after it is created.
func (rt *concurrentReadTx) RLock() {}

// RUnlock signals the end of concurrentReadTx.
func (rt *concurrentReadTx) RUnlock() { rt.txWg.Done() }
