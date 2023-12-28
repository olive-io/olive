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

package backend

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"go.uber.org/zap"

	"github.com/olive-io/olive/pkg/bytesutil"
)

type IBatchTx interface {
	IReadTx
	UnsafeCreateBucket(bucket IBucket)
	UnsafeDeleteBucket(bucket IBucket)
	UnsafePut(bucket IBucket, key []byte, value []byte) error
	UnsafeDelete(bucket IBucket, key []byte) error
	// Commit commits a previous tx and begins a new writable one.
	Commit() error
	// CommitAndStop commits the previous tx and does not create a new one.
	CommitAndStop() error
	LockInsideApply()
	LockOutsideApply()
}

type batchTx struct {
	sync.Mutex

	pwo     *pebble.WriteOptions
	tx      *pebble.Batch
	backend *backend

	pending int
}

// Lock is supposed to be called only by the unit test.
func (t *batchTx) Lock() {
	ValidateCalledInsideUnittest(t.backend.lg)
	t.lock()
}

func (t *batchTx) lock() {
	t.Mutex.Lock()
}

func (t *batchTx) LockInsideApply() {
	t.lock()
	if t.backend.txPostLockInsideApplyHook != nil {
		// The callers of some methods (i.e., (*RaftCluster).AddMember)
		// can be coming from both InsideApply and OutsideApply, but the
		// callers from OutsideApply will have a nil txPostLockInsideApplyHook.
		// So we should check the txPostLockInsideApplyHook before validating
		// the callstack.
		ValidateCalledInsideApply(t.backend.lg)
		t.backend.txPostLockInsideApplyHook()
	}
}

func (t *batchTx) LockOutsideApply() {
	ValidateCalledOutSideApply(t.backend.lg)
	t.lock()
}

func (t *batchTx) Unlock() {
	if t.pending >= t.backend.batchLimit {
		t.commit(false)
	}
	t.Mutex.Unlock()
}

// BatchTx interface embeds ReadTx interface. But RLock() and RUnlock() do not
// have appropriate semantics in BatchTx interface. Therefore should not be called.
// TODO: might want to decouple ReadTx and BatchTx

func (t *batchTx) RLock() {
	panic("unexpected RLock")
}

func (t *batchTx) RUnlock() {
	panic("unexpected RUnlock")
}

func (t *batchTx) UnsafeCreateBucket(bucket IBucket) {
	err := createBucket(t.tx, bucket)
	if err != nil {
		t.backend.lg.Fatal(
			"failed to create a bucket",
			zap.Stringer("bucket-name", bucket),
			zap.Error(err),
		)
	}
	t.pending++
}

func (t *batchTx) UnsafeDeleteBucket(bucket IBucket) {
	err := deleteBucket(t.tx, bucket)
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		t.backend.lg.Fatal(
			"failed to delete a bucket",
			zap.Stringer("bucket-name", bucket),
			zap.Error(err),
		)
	}
	t.pending++
}

// UnsafePut must be called holding the lock on the tx.
func (t *batchTx) UnsafePut(bucket IBucket, key []byte, value []byte) error {
	return t.unsafePut(bucket, key, value)
}

func (t *batchTx) unsafePut(bucketType IBucket, key []byte, value []byte) error {
	key = bytesutil.PathJoin(bucketType.Name(), key)
	if err := t.tx.Set(key, value, t.pwo); err != nil {
		t.backend.lg.Error(
			"failed to write to a bucket",
			zap.Stringer("bucket-name", bucketType),
			zap.Error(err),
		)
		return err
	}
	t.pending++
	return nil
}

// UnsafeGet must be called holding the lock on the tx.
func (t *batchTx) UnsafeGet(bucket IBucket, key []byte) ([]byte, error) {
	_, vals, err := t.UnsafeRange(bucket, key, nil, 1)
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, pebble.ErrNotFound
	}
	return vals[0], nil
}

// UnsafeRange must be called holding the lock on the tx.
func (t *batchTx) UnsafeRange(bucketType IBucket, key, endKey []byte, limit int64) ([][]byte, [][]byte, error) {
	bucket, _ := readBucket(t.tx, bucketType.Name())
	if bucket == nil {
		t.backend.lg.Fatal(
			"failed to find a bucket",
			zap.Stringer("bucket-name", bucketType),
			zap.Stack("stack"),
		)
	}
	return unsafeRange(t.tx, bucket, key, endKey, limit)
}

// UnsafeDelete must be called holding the lock on the tx.
func (t *batchTx) UnsafeDelete(bucketType IBucket, key []byte) error {
	key = bytesutil.PathJoin(bucketType.Name(), key)
	wo := &pebble.WriteOptions{}
	err := t.tx.Delete(key, wo)
	if err != nil {
		t.backend.lg.Error(
			"failed to delete a key",
			zap.Stringer("bucket-name", bucketType),
			zap.Error(err),
		)
		return err
	}
	t.pending++
	return nil
}

// UnsafeForEach must be called holding the lock on the tx.
func (t *batchTx) UnsafeForEach(bucket IBucket, visitor func(k, v []byte) error) error {
	return unsafeForEach(t.tx, bucket, visitor)
}

// Commit commits a previous tx and begins a new writable one.
func (t *batchTx) Commit() error {
	t.lock()
	defer t.Unlock()
	return t.commit(false)
}

// CommitAndStop commits the previous tx and does not create a new one.
func (t *batchTx) CommitAndStop() error {
	t.lock()
	defer t.Unlock()
	return t.commit(true)
}

func (t *batchTx) safePending() int {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.pending
}

func (t *batchTx) commit(stop bool) error {
	// commit the last tx
	if t.tx != nil {
		if t.pending == 0 && !stop {
			return nil
		}

		start := time.Now()

		// gofail: var beforeCommit struct{}
		err := t.tx.Commit(&pebble.WriteOptions{Sync: true})
		// gofail: var afterCommit struct{}

		//writeSec.Observe(t.tx.CommitStats().TotalDuration.Seconds())
		commitSec.Observe(time.Since(start).Seconds())
		atomic.AddInt64(&t.backend.commits, 1)

		t.pending = 0
		if err != nil {
			t.backend.lg.Error("failed to commit tx", zap.Error(err))
			return err
		}

		if err = t.tx.Close(); err != nil {
			t.backend.lg.Error("failed to stop tx", zap.Error(err))
			return err
		}
	}

	if !stop {
		t.tx = t.backend.begin(false)
	}
	return nil
}

type batchTxBuffered struct {
	batchTx
	buf txWriteBuffer
}

func newBatchTxBuffered(backend *backend) *batchTxBuffered {
	tx := &batchTxBuffered{
		batchTx: batchTx{backend: backend},
		buf: txWriteBuffer{
			txBuffer:   txBuffer{make(map[BucketID]*bucketBuffer)},
			bucket2seq: make(map[BucketID]bool),
		},
	}
	tx.Commit()
	return tx
}

func (t *batchTxBuffered) Unlock() {
	if t.pending != 0 {
		t.backend.readTx.Lock() // blocks txReadBuffer for writing.
		t.buf.writeback(&t.backend.readTx.buf)
		t.backend.readTx.Unlock()
		if t.pending >= t.backend.batchLimit {
			t.commit(false)
		}
	}
	t.batchTx.Unlock()
}

func (t *batchTxBuffered) Commit() error {
	t.lock()
	defer t.Unlock()
	return t.commit(false)
}

func (t *batchTxBuffered) CommitAndStop() error {
	t.lock()
	defer t.Unlock()
	return t.commit(true)
}

func (t *batchTxBuffered) commit(stop bool) error {
	// all read txs must be closed to acquire pebble commit rwlock
	t.backend.readTx.Lock()
	defer t.backend.readTx.Unlock()
	return t.unsafeCommit(stop)
}

func (t *batchTxBuffered) unsafeCommit(stop bool) error {
	t.backend.hmu.RLock()
	for i := range t.backend.hooks {
		hook := t.backend.hooks[i]
		hook.OnPreCommitUnsafe(t)
	}
	t.backend.hmu.RUnlock()
	if t.backend.readTx.tx != nil {
		// wait all store read transactions using the current pebble tx to finish,
		// then close the pebble tx
		go func(tx *pebble.Batch, wg *sync.WaitGroup) {
			wg.Wait()
		}(t.backend.readTx.tx, t.backend.readTx.txWg)
		t.backend.readTx.reset()
	}

	if err := t.batchTx.commit(stop); err != nil {
		return err
	}

	if !stop {
		t.backend.readTx.tx = t.backend.begin(false)
	}

	return nil
}

func (t *batchTxBuffered) UnsafePut(bucket IBucket, key []byte, value []byte) error {
	if err := t.batchTx.UnsafePut(bucket, key, value); err != nil {
		return err
	}
	t.buf.put(bucket, key, value)
	return nil
}
