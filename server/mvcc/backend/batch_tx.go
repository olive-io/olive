package backend

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/olive-io/olive/pkg/bytesutil"
	"go.uber.org/zap"
)

type IBatchTx interface {
	IReadTx
	UnsafePut(bucket IBucket, key []byte, value []byte)
	UnsafeSeqPut(bucket IBucket, key []byte, value []byte)
	UnsafeDelete(bucket IBucket, key []byte)
	// Commit commits a previous tx and begins a new writable one.
	Commit()
	// CommitAndStop commits the previous tx and does not create a new one.
	CommitAndStop()
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

// UnsafePut must be called holding the lock on the tx.
func (t *batchTx) UnsafePut(bucket IBucket, key []byte, value []byte) {
	t.unsafePut(bucket, key, value, false)
}

// UnsafeSeqPut must be called holding the lock on the tx.
func (t *batchTx) UnsafeSeqPut(bucket IBucket, key []byte, value []byte) {
	t.unsafePut(bucket, key, value, true)
}

func (t *batchTx) unsafePut(bucketType IBucket, key []byte, value []byte, seq bool) {
	if seq {
		// it is useful to increase fill percent when the workloads are mostly append-only.
		// this can delay the page split and reduce space usage.
		// bucket.FillPercent = 0.9
	}

	key = bytesutil.PathJoin(bucketType.Name(), key)
	if err := t.tx.Set(key, value, t.pwo); err != nil {
		t.backend.lg.Fatal(
			"failed to write to a bucket",
			zap.Stringer("bucket-name", bucketType),
			zap.Error(err),
		)
	}
	t.pending++
}

// UnsafeRange must be called holding the lock on the tx.
func (t *batchTx) UnsafeRange(bucket IBucket, key, endKey []byte, limit int64) ([][]byte, [][]byte, error) {
	//bucket := t.tx.Bucket(bucketType.Name())
	//if bucket == nil {
	//	t.backend.lg.Fatal(
	//		"failed to find a bucket",
	//		zap.Stringer("bucket-name", bucketType),
	//		zap.Stack("stack"),
	//	)
	//}
	return unsafeRange(t.tx, bucket, key, endKey, limit)
}

// UnsafeDelete must be called holding the lock on the tx.
func (t *batchTx) UnsafeDelete(bucketType IBucket, key []byte) {
	key = bytesutil.PathJoin(bucketType.Name(), key)
	wo := &pebble.WriteOptions{}
	err := t.tx.Delete(key, wo)
	if err != nil {
		t.backend.lg.Fatal(
			"failed to delete a key",
			zap.Stringer("bucket-name", bucketType),
			zap.Error(err),
		)
	}
	t.pending++
}

// UnsafeForEach must be called holding the lock on the tx.
func (t *batchTx) UnsafeForEach(bucket IBucket, visitor func(k, v []byte) error) error {
	return unsafeForEach(t.tx, bucket, visitor)
}

// Commit commits a previous tx and begins a new writable one.
func (t *batchTx) Commit() {
	t.lock()
	t.commit(false)
	t.Unlock()
}

// CommitAndStop commits the previous tx and does not create a new one.
func (t *batchTx) CommitAndStop() {
	t.lock()
	t.commit(true)
	t.Unlock()
}

func (t *batchTx) safePending() int {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	return t.pending
}

func (t *batchTx) commit(stop bool) {
	// commit the last tx
	if t.tx != nil {
		if t.pending == 0 && !stop {
			return
		}

		start := time.Now()

		// gofail: var beforeCommit struct{}
		err := t.tx.Commit(t.pwo)
		// gofail: var afterCommit struct{}

		//writeSec.Observe(t.tx.CommitStats().TotalDuration.Seconds())
		commitSec.Observe(time.Since(start).Seconds())
		atomic.AddInt64(&t.backend.commits, 1)

		t.pending = 0
		if err != nil {
			t.backend.lg.Fatal("failed to commit tx", zap.Error(err))
		}

		if err = t.tx.Close(); err != nil {
			t.backend.lg.Fatal("failed to stop tx", zap.Error(err))
		}
	}

	if !stop {
		t.tx = t.backend.begin(true)
	}
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

func (t *batchTxBuffered) Commit() {
	t.lock()
	t.commit(false)
	t.Unlock()
}

func (t *batchTxBuffered) CommitAndStop() {
	t.lock()
	t.commit(true)
	t.Unlock()
}

func (t *batchTxBuffered) commit(stop bool) {
	// all read txs must be closed to acquire pebble commit rwlock
	t.backend.readTx.Lock()
	t.unsafeCommit(stop)
	t.backend.readTx.Unlock()
}

func (t *batchTxBuffered) unsafeCommit(stop bool) {
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

	t.batchTx.commit(stop)

	if !stop {
		t.backend.readTx.tx = t.backend.begin(false)
	}
}

func (t *batchTxBuffered) UnsafePut(bucket IBucket, key []byte, value []byte) {
	t.batchTx.UnsafePut(bucket, key, value)
	t.buf.put(bucket, key, value)
}

func (t *batchTxBuffered) UnsafeSeqPut(bucket IBucket, key []byte, value []byte) {
	t.batchTx.UnsafeSeqPut(bucket, key, value)
	t.buf.putSeq(bucket, key, value)
}
