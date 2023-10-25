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
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
	"go.uber.org/zap"
)

const (
	defaultStorageDir       = "default"
	defaultStorageCacheSize = 1024 * 1024 * 10

	defaultBatchLimit    = 10000
	defaultBatchInterval = 100 * time.Millisecond
)

type IBackend interface {
	// ReadTx returns a read transaction. It is replaced by ConcurrentReadTx in the main data path, see #10523.
	ReadTx() IReadTx
	BatchTx() IBatchTx
	// ConcurrentReadTx returns a non-blocking read transaction.
	ConcurrentReadTx() IReadTx

	Snapshot() ISnapshot
	Hash(ignores func(bucketName, keyName []byte) bool) (uint32, error)
	ForceCommit()
	Close() error

	// SetTxPostLockInsideApplyHook sets a txPostLockInsideApplyHook.
	SetTxPostLockInsideApplyHook(func())
}

type BackendConfig struct {
	// Dir is the file path to the storage file.
	Dir string
	// CacheSize is the size to pebble cache
	CacheSize int64
	// BatchInterval is the maximum time before flushing the BatchTx.
	BatchInterval time.Duration
	// BatchLimit is the maximum puts before flushing the BatchTx.
	BatchLimit int
	// Logger logs backend-side operations.
	Logger *zap.Logger
	// UnsafeNoFsync disables all uses of fsync.
	UnsafeNoFsync bool `json:"unsafe-no-fsync"`
	// Mlock prevents backend database file to be swapped
	Mlock bool

	// Hooks are getting executed during lifecycle of Backend's transactions.
	Hooks IHooks
}

func DefaultBackendConfig() BackendConfig {
	cfg := BackendConfig{
		Dir:           defaultStorageDir,
		CacheSize:     defaultStorageCacheSize,
		BatchInterval: defaultBatchInterval,
		BatchLimit:    defaultBatchLimit,
	}

	return cfg
}

type txReadBufferCache struct {
	mu         sync.Mutex
	buf        *txReadBuffer
	bufVersion uint64
}

type backend struct {
	// size and commits are used with atomic operations so they must be
	// 64-bit aligned, otherwise 32-bit tests will crash

	// size is the number of bytes allocated in the backend
	size int64
	// sizeInUse is the number of bytes actually used in the backend
	sizeInUse int64
	// commits counts number of commits since start
	commits int64
	// openReadTxN is the number of currently open read transactions in the backend
	openReadTxN int64
	// mlock prevents backend database file to be swapped
	mlock bool

	mu    sync.RWMutex
	bopts *pebble.Options
	db    *pebble.DB

	batchInterval time.Duration
	batchLimit    int
	batchTx       *batchTxBuffered

	readTx *readTx
	// txReadBufferCache mirrors "txReadBuffer" within "readTx" -- readTx.baseReadTx.buf.
	// When creating "concurrentReadTx":
	// - if the cache is up-to-date, "readTx.baseReadTx.buf" copy can be skipped
	// - if the cache is empty or outdated, "readTx.baseReadTx.buf" copy is required
	txReadBufferCache txReadBufferCache

	stopc chan struct{}
	donec chan struct{}

	hooks IHooks

	// txPostLockInsideApplyHook is called each time right after locking the tx.
	txPostLockInsideApplyHook func()

	lg *zap.Logger
}

func New(cfg BackendConfig) IBackend {
	return newBackend(cfg)
}

func newBackend(cfg BackendConfig) *backend {
	b := &backend{}

	return b
}

func (b *backend) ReadTx() IReadTx {
	//TODO implement me
	panic("implement me")
}

func (b *backend) BatchTx() IBatchTx {
	//TODO implement me
	panic("implement me")
}

func (b *backend) ConcurrentReadTx() IReadTx {
	//TODO implement me
	panic("implement me")
}

func (b *backend) Snapshot() ISnapshot {
	//TODO implement me
	panic("implement me")
}

func (b *backend) Hash(ignores func(bucketName []byte, keyName []byte) bool) (uint32, error) {
	//TODO implement me
	panic("implement me")
}

func (b *backend) ForceCommit() {
	//TODO implement me
	panic("implement me")
}

func (b *backend) Close() error {
	//TODO implement me
	panic("implement me")
}

func (b *backend) SetTxPostLockInsideApplyHook(f func()) {
	if f != nil {
		f()
	}
}

func (b *backend) unsafeBegin(write bool) *pebble.Batch {
	tx := b.db.NewBatch()

	return tx
}

func (b *backend) begin(write bool) *pebble.Batch {
	b.mu.RLock()
	tx := b.unsafeBegin(write)
	b.mu.RUnlock()

	//size := tx.Size()
	//db := tx.DB()
	//stats := db.Stats()
	//atomic.StoreInt64(&b.size, size)
	//atomic.StoreInt64(&b.sizeInUse, size-(int64(stats.FreePageN)*int64(db.Info().PageSize)))
	//atomic.StoreInt64(&b.openReadTxN, int64(stats.OpenTxN))

	return tx
}
