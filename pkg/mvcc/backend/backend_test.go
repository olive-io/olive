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

package backend_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/oliveio/olive/pkg/mvcc/backend"
	betesting "github.com/oliveio/olive/pkg/mvcc/backend/testing"
	"github.com/oliveio/olive/pkg/mvcc/buckets"
	"github.com/stretchr/testify/assert"
)

func TestBackendClose(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)

	// check close could work
	done := make(chan struct{})
	go func() {
		err := b.Close()
		if err != nil {
			t.Errorf("close error = %v, want nil", err)
		}
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Errorf("failed to close database in 10s")
	}
}

func TestBackendSnapshot(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Test)
	tx.UnsafePut(buckets.Test, []byte("foo"), []byte("bar"))
	tx.Unlock()
	b.ForceCommit()

	// write snapshot to a new file
	f, err := os.CreateTemp(t.TempDir(), "olive_backend_test")
	if err != nil {
		t.Fatal(err)
	}

	snap := b.Snapshot()
	if _, err := snap.WriteTo(f); err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, snap.Close())
	assert.NoError(t, f.Close())
	betesting.Close(t, b)

	reader, _ := os.Open(f.Name())

	// bootstrap new backend from the snapshot
	bcfg := backend.DefaultBackendConfig()
	dir, err := os.MkdirTemp(t.TempDir(), "etcd_backend_test1")
	if err != nil {
		panic(err)
	}
	tmpPath := filepath.Join(dir, "database1")
	bcfg.Dir, bcfg.BatchInterval, bcfg.BatchLimit = tmpPath, time.Hour, 10000
	nb := backend.New(bcfg)
	defer betesting.Close(t, nb)

	if reader != nil {
		if err := nb.Recover(reader); err != nil {
			t.Fatal(err)
		}
	}

	newTx := nb.BatchTx()
	newTx.Lock()
	ks, _, err := newTx.UnsafeRange(buckets.Test, []byte("foo"), []byte("goo"), 0)
	if err != nil {
		t.Error(err)
	}
	if len(ks) != 1 {
		t.Errorf("len(kvs) = %d, want 1", len(ks))
	}
	newTx.Unlock()
}

func TestBackendBatchIntervalCommit(t *testing.T) {
	// start backend with super short batch interval so
	// we do not need to wait long before commit to happen.
	b, _ := betesting.NewTmpBackend(t, time.Nanosecond, 10000)

	pc := backend.CommitsForTest(b)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Test)
	tx.UnsafePut(buckets.Test, []byte("foo"), []byte("bar"))
	tx.Unlock()

	for i := 0; i < 10; i++ {
		if backend.CommitsForTest(b) >= pc+1 {
			break
		}
		time.Sleep(time.Duration(i*100) * time.Millisecond)
	}

	// check whether put happens via db view
	batch := backend.DbFromBackendForTest(b).NewIndexedBatch()
	defer batch.Close()
	_, _, err := batch.Get([]byte("test/foo"))
	assert.NoError(t, err)
}

// TestBackendWriteback ensures writes are stored to the read txn on write txn unlock.
func TestBackendWriteback(t *testing.T) {
	b, _ := betesting.NewDefaultTmpBackend(t)
	defer betesting.Close(t, b)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Key)
	tx.UnsafePut(buckets.Key, []byte("abc"), []byte("bar"))
	tx.UnsafePut(buckets.Key, []byte("def"), []byte("baz"))
	tx.UnsafePut(buckets.Key, []byte("overwrite"), []byte("1"))
	tx.Unlock()

	// overwrites should be propagated too
	tx.Lock()
	tx.UnsafePut(buckets.Key, []byte("overwrite"), []byte("2"))
	tx.Unlock()

	keys := []struct {
		key   []byte
		end   []byte
		limit int64

		wkey [][]byte
		wval [][]byte
	}{
		{
			key: []byte("abc"),
			end: nil,

			wkey: [][]byte{[]byte("abc")},
			wval: [][]byte{[]byte("bar")},
		},
		{
			key: []byte("abc"),
			end: []byte("def"),

			wkey: [][]byte{[]byte("abc")},
			wval: [][]byte{[]byte("bar")},
		},
		{
			key: []byte("abc"),
			end: []byte("deg"),

			wkey: [][]byte{[]byte("abc"), []byte("def")},
			wval: [][]byte{[]byte("bar"), []byte("baz")},
		},
		{
			key:   []byte("abc"),
			end:   []byte("\xff"),
			limit: 1,

			wkey: [][]byte{[]byte("abc")},
			wval: [][]byte{[]byte("bar")},
		},
		{
			key: []byte("abc"),
			end: []byte("\xff"),

			wkey: [][]byte{[]byte("abc"), []byte("def"), []byte("overwrite")},
			wval: [][]byte{[]byte("bar"), []byte("baz"), []byte("2")},
		},
	}
	rtx := b.ReadTx()
	for i, tt := range keys {
		func() {
			rtx.RLock()
			defer rtx.RUnlock()
			k, v, _ := rtx.UnsafeRange(buckets.Key, tt.key, tt.end, tt.limit)
			if !reflect.DeepEqual(tt.wkey, k) || !reflect.DeepEqual(tt.wval, v) {
				t.Errorf("#%d: want k=%+v, v=%+v; got k=%+v, v=%+v", i, tt.wkey, tt.wval, k, v)
			}
		}()
	}
}

// TestConcurrentReadTx ensures that current read transaction can see all prior writes stored in read buffer
func TestConcurrentReadTx(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	wtx1 := b.BatchTx()
	wtx1.Lock()
	wtx1.UnsafeCreateBucket(buckets.Key)
	wtx1.UnsafePut(buckets.Key, []byte("abc"), []byte("ABC"))
	wtx1.UnsafePut(buckets.Key, []byte("overwrite"), []byte("1"))
	wtx1.Unlock()

	wtx2 := b.BatchTx()
	wtx2.Lock()
	wtx2.UnsafePut(buckets.Key, []byte("def"), []byte("DEF"))
	wtx2.UnsafePut(buckets.Key, []byte("overwrite"), []byte("2"))
	wtx2.Unlock()

	rtx := b.ConcurrentReadTx()
	rtx.RLock() // no-op
	k, v, err := rtx.UnsafeRange(buckets.Key, []byte("abc"), []byte("\xff"), 0)
	if err != nil {
		rtx.RUnlock()
		t.Fatal(err)
	}
	rtx.RUnlock()
	wKey := [][]byte{[]byte("abc"), []byte("def"), []byte("overwrite")}
	wVal := [][]byte{[]byte("ABC"), []byte("DEF"), []byte("2")}
	if !reflect.DeepEqual(wKey, k) || !reflect.DeepEqual(wVal, v) {
		t.Errorf("want k=%+v, v=%+v; got k=%+v, v=%+v", wKey, wVal, k, v)
	}
}

// TestBackendWritebackForEach checks that partially written / buffered
// data is visited in the same order as fully committed data.
func TestBackendWritebackForEach(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Key)
	for i := 0; i < 5; i++ {
		k := []byte(fmt.Sprintf("%04d", i))
		tx.UnsafePut(buckets.Key, k, []byte("bar"))
	}
	tx.Unlock()

	// writeback
	b.ForceCommit()

	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Key)
	for i := 5; i < 20; i++ {
		k := []byte(fmt.Sprintf("%04d", i))
		tx.UnsafePut(buckets.Key, k, []byte("bar"))
	}
	tx.Unlock()

	seq := ""
	getSeq := func(k, v []byte) error {
		seq += string(k)
		return nil
	}
	rtx := b.ReadTx()
	rtx.RLock()
	assert.NoError(t, rtx.UnsafeForEach(buckets.Key, getSeq))
	rtx.RUnlock()

	partialSeq := seq

	seq = ""
	b.ForceCommit()

	tx.Lock()
	assert.NoError(t, tx.UnsafeForEach(buckets.Key, getSeq))
	tx.Unlock()

	if seq != partialSeq {
		t.Fatalf("expected %q, got %q", seq, partialSeq)
	}
}
