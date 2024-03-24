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
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/pebble"
	humanize "github.com/dustin/go-humanize"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
)

const (
	defaultStorageDir       = "default"
	defaultStorageCacheSize = 1024 * 1024 * 10

	defaultBatchLimit    = 10000
	defaultBatchInterval = 100 * time.Millisecond

	// minSnapshotWarningTimeout is the minimum threshold to trigger a long running snapshot warning.
	minSnapshotWarningTimeout = 30 * time.Second
)

var defaultSplit pebble.Split = func(a []byte) int {
	return 1
}

type IBackend interface {
	// ReadTx returns a read transaction. It is replaced by ConcurrentReadTx in the main data path, see #10523.
	ReadTx() IReadTx
	BatchTx() IBatchTx
	// ConcurrentReadTx returns a non-blocking read transaction.
	ConcurrentReadTx() IReadTx

	Snapshot() ISnapshot
	Recover(reader io.Reader) error

	Hash(ignores func(bucketName, keyName []byte) bool) (uint32, error)
	ForceCommit()
	Close() error

	AppendHooks(hook IHooks)
	// SetTxPostLockInsideApplyHook sets a txPostLockInsideApplyHook.
	SetTxPostLockInsideApplyHook(func())
}

type BackendConfig struct {
	// Dir is the file path to the storage file.
	Dir string
	WAL string
	// CacheSize is the size to pebble cache
	CacheSize uint64
	// BatchInterval is the maximum time before flushing the BatchTx.
	BatchInterval time.Duration
	// BatchLimit is the maximum puts before flushing the BatchTx.
	BatchLimit int
	// Logger logs backend-side operations.
	Logger *zap.Logger
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

	mu  sync.RWMutex
	pwo *pebble.WriteOptions
	db  *pebble.DB

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

	hmu sync.RWMutex
	// hooks are getting executed during lifecycle of Backend's transactions.
	hooks []IHooks

	// txPostLockInsideApplyHook is called each time right after locking the tx.
	txPostLockInsideApplyHook func()

	lg *zap.Logger
}

func New(cfg BackendConfig) IBackend {
	return newBackend(cfg)
}

func NewDefaultBackend(dir string) IBackend {
	bcfg := DefaultBackendConfig()
	bcfg.Dir = dir
	return newBackend(bcfg)
}

func newBackend(cfg BackendConfig) *backend {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewExample()
	}

	comparer := pebble.DefaultComparer
	comparer.Split = defaultSplit
	plg := cfg.Logger.
		WithOptions(zap.Fields(zap.String("pkg", "pebble"))).
		Sugar()
	options := &pebble.Options{
		Comparer: comparer,
		Logger:   plg,
	}
	if cfg.CacheSize != 0 {
		options.Cache = pebble.NewCache(int64(cfg.CacheSize))
	}
	if cfg.WAL != "" {
		options.WALDir = cfg.WAL
	}
	db, err := pebble.Open(cfg.Dir, options)
	if err != nil {
		cfg.Logger.Panic("failed to open database", zap.String("dir", cfg.Dir), zap.Error(err))
	}

	pwo := &pebble.WriteOptions{Sync: false}

	// In future, may want to make buffering optional for low-concurrency systems
	// or dynamically swap between buffered/non-buffered depending on workload.
	b := &backend{
		pwo: pwo,
		db:  db,

		batchInterval: cfg.BatchInterval,
		batchLimit:    cfg.BatchLimit,

		readTx: &readTx{
			baseReadTx: baseReadTx{
				buf: txReadBuffer{
					txBuffer:   txBuffer{make(map[BucketID]*bucketBuffer)},
					bufVersion: 0,
				},
				buckets: make(map[BucketID]*localBucket),
				txWg:    new(sync.WaitGroup),
				txMu:    new(sync.RWMutex),
			},
		},
		txReadBufferCache: txReadBufferCache{
			mu:         sync.Mutex{},
			buf:        nil,
			bufVersion: 0,
		},

		stopc: make(chan struct{}),
		donec: make(chan struct{}),

		hooks: make([]IHooks, 0),

		lg: cfg.Logger,
	}

	b.batchTx = newBatchTxBuffered(b)

	go b.run()
	return b
}

func (b *backend) ReadTx() IReadTx { return b.readTx }

// BatchTx returns the current batch tx in coalescer. The tx can be used for read and
// write operations. The write result can be retrieved within the same tx immediately.
// The write result is isolated with other txs until the current one get committed.
func (b *backend) BatchTx() IBatchTx { return b.batchTx }

// ConcurrentReadTx creates and returns a new ReadTx, which:
// A) creates and keeps a copy of backend.readTx.txReadBuffer,
// B) references the pebble read Tx (and its bucket cache) of current batch interval.
func (b *backend) ConcurrentReadTx() IReadTx {
	b.readTx.RLock()
	defer b.readTx.RUnlock()
	// prevent pebble read Tx from been rolled back until store read Tx is done. Needs to be called when holding readTx.RLock().
	b.readTx.txWg.Add(1)

	// TODO: might want to copy the read buffer lazily - create copy when A) end of a write transaction B) end of a batch interval.

	// inspect/update cache recency iff there's no ongoing update to the cache
	// this falls through if there's no cache update

	// by this line, "ConcurrentReadTx" code path is already protected against concurrent "writeback" operations
	// which requires write lock to update "readTx.baseReadTx.buf".
	// Which means setting "buf *txReadBuffer" with "readTx.buf.unsafeCopy()" is guaranteed to be up-to-date,
	// whereas "txReadBufferCache.buf" may be stale from concurrent "writeback" operations.
	// We only update "txReadBufferCache.buf" if we know "buf *txReadBuffer" is up-to-date.
	// The update to "txReadBufferCache.buf" will benefit the following "ConcurrentReadTx" creation
	// by avoiding copying "readTx.baseReadTx.buf".
	b.txReadBufferCache.mu.Lock()

	curCache := b.txReadBufferCache.buf
	curCacheVer := b.txReadBufferCache.bufVersion
	curBufVer := b.readTx.buf.bufVersion

	isEmptyCache := curCache == nil
	isStaleCache := curCacheVer != curBufVer

	var buf *txReadBuffer
	switch {
	case isEmptyCache:
		// perform safe copy of buffer while holding "b.txReadBufferCache.mu.Lock"
		// this is only supposed to run once so there won't be much overhead
		curBuf := b.readTx.buf.unsafeCopy()
		buf = &curBuf
	case isStaleCache:
		// to maximize the concurrency, try unsafe copy of buffer
		// release the lock while copying buffer -- cache may become stale again and
		// get overwritten by someone else.
		// therefore, we need to check the readTx buffer version again
		b.txReadBufferCache.mu.Unlock()
		curBuf := b.readTx.buf.unsafeCopy()
		b.txReadBufferCache.mu.Lock()
		buf = &curBuf
	default:
		// neither empty nor stale cache, just use the current buffer
		buf = curCache
	}
	// txReadBufferCache.bufVersion can be modified when we doing an unsafeCopy()
	// as a result, curCacheVer could be no longer the same as
	// txReadBufferCache.bufVersion
	// if !isEmptyCache && curCacheVer != b.txReadBufferCache.bufVersion
	// then the cache became stale while copying "readTx.baseReadTx.buf".
	// It is safe to not update "txReadBufferCache.buf", because the next following
	// "ConcurrentReadTx" creation will trigger a new "readTx.baseReadTx.buf" copy
	// and "buf" is still used for the current "concurrentReadTx.baseReadTx.buf".
	if isEmptyCache || curCacheVer == b.txReadBufferCache.bufVersion {
		// continue if the cache is never set or no one has modified the cache
		b.txReadBufferCache.buf = buf
		b.txReadBufferCache.bufVersion = curBufVer
	}

	b.txReadBufferCache.mu.Unlock()

	// concurrentReadTx is not supposed to write to its txReadBuffer
	return &concurrentReadTx{
		baseReadTx: baseReadTx{
			buf:     *buf,
			txMu:    b.readTx.txMu,
			tx:      b.readTx.tx,
			buckets: b.readTx.buckets,
			txWg:    b.readTx.txWg,
		},
	}
}

func (b *backend) Snapshot() ISnapshot {
	b.batchTx.Commit()

	b.mu.RLock()
	defer b.mu.RUnlock()

	dbBytes := b.db.Metrics().Total().Size
	sn := b.db.NewSnapshot()
	stopc, donec := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(donec)
		// sendRateBytes is based on transferring snapshot data over a 1 gigabit/s connection
		// assuming a min tcp throughput of 100MB/s.
		var sendRateBytes int64 = 100 * 1024 * 1024
		warningTimeout := time.Duration(int64((float64(dbBytes) / float64(sendRateBytes)) * float64(time.Second)))
		if warningTimeout < minSnapshotWarningTimeout {
			warningTimeout = minSnapshotWarningTimeout
		}
		start := time.Now()
		ticker := time.NewTicker(warningTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.lg.Warn(
					"snapshotting taking too long to transfer",
					zap.Duration("taking", time.Since(start)),
					zap.Int64("bytes", dbBytes),
					zap.String("size", humanize.Bytes(uint64(dbBytes))),
				)

			case <-stopc:
				snapshotTransferSec.Observe(time.Since(start).Seconds())
				return
			}
		}
	}()

	return &snapshot{sn, stopc, donec}
}

func (b *backend) Recover(reader io.Reader) error {
	b.ForceCommit()

	sz := make([]byte, 8)
	if _, err := io.ReadFull(reader, sz); err != nil {
		return err
	}

	total := binary.LittleEndian.Uint64(sz)
	wb := b.db.NewBatch()
	defer wb.Close()
	for i := uint64(0); i < total; i++ {
		if _, err := io.ReadFull(reader, sz); err != nil {
			return err
		}
		toRead := binary.LittleEndian.Uint64(sz)
		data := make([]byte, toRead)
		if _, err := io.ReadFull(reader, data); err != nil {
			return err
		}
		rkv := &pb.InternalKV{}
		if err := proto.Unmarshal(data, rkv); err != nil {
			panic(err)
		}
		wb.Set(rkv.Key, rkv.Value, b.pwo)
	}
	err := wb.Commit(b.pwo)
	if err != nil {
		return err
	}

	return nil
}

func (b *backend) Hash(ignores func(bucketName []byte, keyName []byte) bool) (uint32, error) {
	h := crc32.New(crc32.MakeTable(crc32.Castagnoli))

	b.mu.RLock()
	defer b.mu.RUnlock()
	iter := b.db.NewIter(&pebble.IterOptions{})
	for iter.First(); iteratorIsValid(iter); iter.Next() {
		k := iter.Key()
		v := iter.Value()

		bn, _, _ := bytes.Cut(k, []byte("/"))
		if ignores != nil && !ignores(bn, k) {
			h.Write(k)
			h.Write(v)
		}
	}

	return h.Sum32(), nil
}

func (b *backend) ForceCommit() {
	_ = b.db.Flush()
}

func (b *backend) run() {
	defer close(b.donec)
	t := time.NewTimer(b.batchInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-b.stopc:
			_ = b.batchTx.CommitAndStop()
			return
		}
		if b.batchTx.safePending() != 0 {
			_ = b.batchTx.Commit()
		}
		t.Reset(b.batchInterval)
	}
}

// Commits returns total number of commits since start
func (b *backend) Commits() int64 {
	return atomic.LoadInt64(&b.commits)
}

func (b *backend) Close() error {
	close(b.stopc)
	<-b.donec
	return b.db.Close()
}

func (b *backend) AppendHooks(hook IHooks) {
	b.hmu.Lock()
	defer b.hmu.Unlock()
	b.hooks = append(b.hooks, hook)
}

func (b *backend) SetTxPostLockInsideApplyHook(hook func()) {
	// It needs to lock the batchTx, because the periodic commit
	// may be accessing the txPostLockInsideApplyHook at the moment.
	b.batchTx.lock()
	defer b.batchTx.Unlock()
	b.txPostLockInsideApplyHook = hook
}

func (b *backend) unsafeBegin(writeOnly bool) *pebble.Batch {
	var tx *pebble.Batch
	if writeOnly {
		tx = b.db.NewBatch()
	} else {
		tx = b.db.NewIndexedBatch()
	}

	return tx
}

func (b *backend) begin(writeOnly bool) *pebble.Batch {
	b.mu.RLock()
	tx := b.unsafeBegin(writeOnly)
	b.mu.RUnlock()

	return tx
}
