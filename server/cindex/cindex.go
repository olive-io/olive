package cindex

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/server/mvcc/backend"
	"github.com/olive-io/olive/server/mvcc/buckets"
)

type IBackend interface {
	BatchTx() backend.IBatchTx
	ReadTx() backend.IReadTx
}

// IConsistentIndexer is an interface that wraps the Get/Set/Save method for consistentIndex.
type IConsistentIndexer interface {

	// ConsistentIndex returns the consistent index of current executing entry.
	ConsistentIndex() uint64

	// ConsistentApplyingIndex returns the consistent applying index of current executing entry.
	ConsistentApplyingIndex() (uint64, uint64)

	// UnsafeConsistentIndex is similar to ConsistentIndex, but it doesn't lock the transaction.
	UnsafeConsistentIndex() uint64

	// SetConsistentIndex set the consistent index of current executing entry.
	SetConsistentIndex(v, term uint64)

	// SetConsistentApplyingIndex set the consistent applying index of current executing entry.
	SetConsistentApplyingIndex(v, term uint64)

	// UnsafeSave must be called holding the lock on the tx.
	// It saves consistentIndex to the underlying stable storage.
	UnsafeSave(tx backend.IBatchTx)

	// SetBackend set the available backend.IBatchTx for ConsistentIndexer.
	SetBackend(be IBackend)
}

// consistentIndex implements the ConsistentIndexer interface.
type consistentIndex struct {
	// consistentIndex represents the offset of an entry in a consistent replica log.
	// It caches the "consistent_index" key's value.
	// Accessed through atomics so must be 64-bit aligned.
	consistentIndex uint64
	// term represents the RAFT term of committed entry in a consistent replica log.
	// Accessed through atomics so must be 64-bit aligned.
	// The value is being persisted in the backend since v3.5.
	term uint64

	// applyingIndex are just temporary cache of the sm.Entry.Index
	// and sm.Entry.Term, and they are not ready to be persisted yet. They will be
	// saved to consistentIndex and term above in the txPostLockInsideApplyHook.
	applyingIndex uint64
	applyingTerm  uint64

	// be is used for initial read consistentIndex
	be IBackend
	// mutex is protecting be.
	mutex sync.Mutex

	shardID, nodeID uint64
}

// NewConsistentIndex creates a new consistent index.
// If `be` is nil, it must be set (SetBackend) before first access using `ConsistentIndex()`.
func NewConsistentIndex(be IBackend, shardID, nodeID uint64) IConsistentIndexer {
	return &consistentIndex{be: be, shardID: shardID, nodeID: nodeID}
}

func (ci *consistentIndex) ConsistentIndex() uint64 {
	if index := atomic.LoadUint64(&ci.consistentIndex); index > 0 {
		return index
	}
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	v, term := ReadConsistentIndex(ci.be.ReadTx(), ci.shardID, ci.nodeID)
	ci.SetConsistentIndex(v, term)
	return v
}

func (ci *consistentIndex) UnsafeConsistentIndex() uint64 {
	if index := atomic.LoadUint64(&ci.consistentIndex); index > 0 {
		return index
	}

	v, term := unsafeReadConsistentIndex(ci.be.ReadTx(), ci.shardID, ci.nodeID)
	ci.SetConsistentIndex(v, term)
	return v
}

func (ci *consistentIndex) SetConsistentIndex(v uint64, term uint64) {
	atomic.StoreUint64(&ci.consistentIndex, v)
	atomic.StoreUint64(&ci.term, term)
}

func (ci *consistentIndex) UnsafeSave(tx backend.IBatchTx) {
	index := atomic.LoadUint64(&ci.consistentIndex)
	term := atomic.LoadUint64(&ci.term)
	UnsafeUpdateConsistentIndex(tx, index, term, ci.shardID, ci.nodeID)
}

func (ci *consistentIndex) SetBackend(be IBackend) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.be = be
	// After the backend is changed, the first access should re-read it.
	ci.SetConsistentIndex(0, 0)
}

func (ci *consistentIndex) ConsistentApplyingIndex() (uint64, uint64) {
	return atomic.LoadUint64(&ci.applyingIndex), atomic.LoadUint64(&ci.applyingTerm)
}

func (ci *consistentIndex) SetConsistentApplyingIndex(v uint64, term uint64) {
	atomic.StoreUint64(&ci.applyingIndex, v)
	atomic.StoreUint64(&ci.applyingTerm, term)
}

func NewFakeConsistentIndex(index uint64) IConsistentIndexer {
	return &fakeConsistentIndex{index: index}
}

type fakeConsistentIndex struct {
	index uint64
	term  uint64
}

func (f *fakeConsistentIndex) ConsistentIndex() uint64 {
	return atomic.LoadUint64(&f.index)
}
func (f *fakeConsistentIndex) ConsistentApplyingIndex() (uint64, uint64) {
	return atomic.LoadUint64(&f.index), atomic.LoadUint64(&f.term)
}
func (f *fakeConsistentIndex) UnsafeConsistentIndex() uint64 {
	return atomic.LoadUint64(&f.index)
}

func (f *fakeConsistentIndex) SetConsistentIndex(index uint64, term uint64) {
	atomic.StoreUint64(&f.index, index)
	atomic.StoreUint64(&f.term, term)
}
func (f *fakeConsistentIndex) SetConsistentApplyingIndex(index uint64, term uint64) {
	atomic.StoreUint64(&f.index, index)
	atomic.StoreUint64(&f.term, term)
}

func (f *fakeConsistentIndex) UnsafeSave(_ backend.IBatchTx) {}
func (f *fakeConsistentIndex) SetBackend(_ IBackend)         {}

// UnsafeCreateMetaBucket creates the `meta` bucket (if it does not exists yet).
func UnsafeCreateMetaBucket(tx backend.IBatchTx) {}

// CreateMetaBucket creates the `meta` bucket (if it does not exists yet).
func CreateMetaBucket(tx backend.IBatchTx) {
	tx.LockOutsideApply()
	defer tx.Unlock()
}

// unsafeGetConsistentIndex loads consistent index & term from given transaction.
// returns 0,0 if the data are not found.
// Term is persisted since v3.5.
func unsafeReadConsistentIndex(tx backend.IReadTx, shardID, nodeID uint64) (uint64, uint64) {
	key := bytesutil.PathJoin(buckets.MetaConsistentIndexKeyName, []byte(fmt.Sprintf("%d/%d", shardID, nodeID)))
	_, vs, _ := tx.UnsafeRange(buckets.Meta, key, nil, 0)
	if len(vs) == 0 {
		return 0, 0
	}
	v := binary.BigEndian.Uint64(vs[0])

	key = bytesutil.PathJoin(buckets.MetaTermKeyName, []byte(fmt.Sprintf("%d/%d", shardID, nodeID)))
	_, ts, _ := tx.UnsafeRange(buckets.Meta, buckets.MetaTermKeyName, nil, 0)
	if len(ts) == 0 {
		return v, 0
	}
	t := binary.BigEndian.Uint64(ts[0])
	return v, t
}

// ReadConsistentIndex loads consistent index and term from given transaction.
// returns 0 if the data are not found.
func ReadConsistentIndex(tx backend.IReadTx, shardID, nodeID uint64) (uint64, uint64) {
	tx.Lock()
	defer tx.Unlock()
	return unsafeReadConsistentIndex(tx, shardID, nodeID)
}

func UnsafeUpdateConsistentIndex(tx backend.IBatchTx, index, term, shardID, nodeID uint64) {
	if index == 0 {
		// Never save 0 as it means that we didn't loaded the real index yet.
		return
	}

	bs1 := make([]byte, 8)
	binary.BigEndian.PutUint64(bs1, index)
	key := bytesutil.PathJoin(buckets.MetaConsistentIndexKeyName, []byte(fmt.Sprintf("%d/%d", shardID, nodeID)))
	// put the index into the underlying backend
	// tx has been locked in TxnBegin, so there is no need to lock it again
	tx.UnsafePut(buckets.Meta, key, bs1)
	if term > 0 {
		bs2 := make([]byte, 8)
		binary.BigEndian.PutUint64(bs2, term)
		key = bytesutil.PathJoin(buckets.MetaTermKeyName, []byte(fmt.Sprintf("%d/%d", shardID, nodeID)))
		tx.UnsafePut(buckets.Meta, key, bs2)
	}
}

func UpdateConsistentIndex(tx backend.IBatchTx, index, term, shardID, nodeID uint64) {
	tx.LockOutsideApply()
	defer tx.Unlock()
	UnsafeUpdateConsistentIndex(tx, index, term, shardID, nodeID)
}
