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
	"context"

	"github.com/oliveio/olive/api"
	"github.com/oliveio/olive/pkg/storage/backend"
	"github.com/oliveio/olive/pkg/traceutil"
)

type RangeOptions struct {
	Limit int64
	Rev   int64
	Count bool
}

type RangeResult struct {
	KVs   []api.KeyValue
	Rev   int64
	Count int
}

type IReadView interface {
	// FirstRev returns the first KV revision at the time of opening the txn.
	// After a compaction, the first revision increases to the compaction
	// revision.
	FirstRev() int64

	// Rev returns the revision of the KV at the time of opening the txn.
	Rev() int64

	// Range gets the keys in the range at rangeRev.
	// The returned rev is the current revision of the KV when the operation is executed.
	// If rangeRev <=0, range gets the keys at currentRev.
	// If `end` is nil, the request returns the key.
	// If `end` is not nil and not empty, it gets the keys in range [key, range_end).
	// If `end` is not nil and empty, it gets the keys greater than or equal to key.
	// Limit limits the number of keys returned.
	// If the required rev is compacted, ErrCompacted will be returned.
	Range(ctx context.Context, key, end []byte, ro RangeOptions) (r *RangeResult, err error)
}

// ITxnRead represents a read-only transaction with operations that will not
// block other read transactions.
type ITxnRead interface {
	IReadView
	// End marks the transaction is complete and ready to commit.
	End()
}

type IWriteView interface {
	// DeleteRange deletes the given range from the store.
	// A deleteRange increases the rev of the store if any key in the range exists.
	// The number of key deleted will be returned.
	// The returned rev is the current revision of the KV when the operation is executed.
	// It also generates one event for each key delete in the event history.
	// if the `end` is nil, deleteRange deletes the key.
	// if the `end` is not nil, deleteRange deletes the keys in range [key, range_end).
	DeleteRange(key, end []byte) (n, rev int64)

	// Put puts the given key, value into the store.
	// A put also increases the rev of the store, and generates one event in the event history.
	// The returned rev is the current revision of the KV when the operation is executed.
	Put(key, value []byte) (rev int64)
}

// ITxnWrite represents a transaction that can modify the store.
type ITxnWrite interface {
	ITxnRead
	IWriteView
	// Changes gets the changes made since opening the write txn.
	Changes() []api.KeyValue
}

// txnReadWrite coerces a read txn to a write, panicking on any write operation.
type txnReadWrite struct{ ITxnRead }

func (trw *txnReadWrite) DeleteRange(key, end []byte) (n, rev int64) { panic("unexpected DeleteRange") }
func (trw *txnReadWrite) Put(key, value []byte) (rev int64) {
	panic("unexpected Put")
}
func (trw *txnReadWrite) Changes() []api.KeyValue { return nil }

func NewReadOnlyTxnWrite(txn ITxnRead) ITxnWrite { return &txnReadWrite{txn} }

type ReadTxMode uint32

const (
	// ConcurrentReadTxMode use ConcurrentReadTx and the txReadBuffer is copied
	ConcurrentReadTxMode = ReadTxMode(1)
	// SharedBufReadTxMode use backend ReadTx and txReadBuffer is not copied
	SharedBufReadTxMode = ReadTxMode(2)
)

type IKV interface {
	IReadView
	IWriteView

	// Read creates a read transaction.
	Read(mode ReadTxMode, trace *traceutil.Trace) ITxnRead

	// Write creates a write transaction.
	Write(trace *traceutil.Trace) ITxnWrite

	// HashStorage returns IHashStorage interface for IKV storage.
	HashStorage() IHashStorage

	// Compact frees all superseded keys with revisions less than rev.
	Compact(trace *traceutil.Trace, rev int64) (<-chan struct{}, error)

	// Commit commits outstanding txns into the underlying backend.
	Commit()

	// Restore restores the KV store from a backend.
	Restore(b backend.IBackend) error
	Close() error
}
