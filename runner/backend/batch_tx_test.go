/*
   Copyright 2023 The olive Authors

   This program is offered under a commercial and under the AGPL license.
   For AGPL licensing, see below.

   AGPL licensing:
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package backend_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/runner/backend"
	betesting "github.com/olive-io/olive/runner/backend/testing"
	"github.com/olive-io/olive/runner/buckets"
)

func TestBatchTxPut(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	tx := b.BatchTx()

	tx.Lock()

	// put
	v := []byte("bar")
	tx.UnsafeCreateBucket(buckets.Test)
	tx.UnsafePut(buckets.Test, []byte("foo"), v)

	tx.Unlock()

	// check put result before and after tx is committed
	for k := 0; k < 2; k++ {
		tx.Lock()
		_, gv, _ := tx.UnsafeRange(buckets.Test, []byte("foo"), nil, 0)
		tx.Unlock()
		if !reflect.DeepEqual(gv[0], v) {
			t.Errorf("v = %s, want %s", string(gv[0]), string(v))
		}
		tx.Commit()
	}
}

func TestBatchTxGet(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	tx := b.BatchTx()

	tx.Lock()

	// put
	v := []byte("bar")
	tx.UnsafeCreateBucket(buckets.Meta)
	tx.UnsafePut(buckets.Meta, []byte("runner"), v)

	tx.Unlock()

	// check put result before and after tx is committed
	val, _ := b.ReadTx().UnsafeGet(buckets.Meta, []byte("runner"))
	if !assert.Equal(t, v, val) {
		return
	}
}

func TestBatchTxRange(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Test)
	defer tx.Unlock()

	// put keys
	allKeys := [][]byte{[]byte("foo"), []byte("foo1"), []byte("foo2")}
	allVals := [][]byte{[]byte("bar"), []byte("bar1"), []byte("bar2")}
	for i := range allKeys {
		tx.UnsafePut(buckets.Test, allKeys[i], allVals[i])
	}

	tests := []struct {
		key    []byte
		endKey []byte
		limit  int64

		wkeys [][]byte
		wvals [][]byte
	}{
		// single key
		{
			[]byte("foo"), nil, 0,
			allKeys[:1], allVals[:1],
		},
		// single key, bad
		{
			[]byte("doo"), nil, 0,
			nil, nil,
		},
		// key range
		{
			[]byte("foo"), []byte("foo1"), 0,
			allKeys[:1], allVals[:1],
		},
		// key range, get all keys
		{
			[]byte("foo"), []byte("foo3"), 0,
			allKeys, allVals,
		},
		// key range, bad
		{
			[]byte("goo"), []byte("goo3"), 0,
			nil, nil,
		},
		// key range with effective limit
		{
			[]byte("foo"), []byte("foo3"), 1,
			allKeys[:1], allVals[:1],
		},
		// key range with limit
		{
			[]byte("foo"), []byte("foo3"), 4,
			allKeys, allVals,
		},
	}
	for i, tt := range tests {
		keys, vals, _ := tx.UnsafeRange(buckets.Test, tt.key, tt.endKey, tt.limit)
		if !reflect.DeepEqual(keys, tt.wkeys) {
			t.Errorf("#%d: keys = %+v, want %+v", i, keys, tt.wkeys)
		}
		if !reflect.DeepEqual(vals, tt.wvals) {
			t.Errorf("#%d: vals = %+v, want %+v", i, vals, tt.wvals)
		}
	}
}

func TestBatchTxDelete(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
	defer betesting.Close(t, b)

	tx := b.BatchTx()
	tx.Lock()

	tx.UnsafeCreateBucket(buckets.Test)
	tx.UnsafePut(buckets.Test, []byte("foo"), []byte("bar"))

	tx.UnsafeDelete(buckets.Test, []byte("foo"))

	tx.Unlock()

	// check put result before and after tx is committed
	for k := 0; k < 2; k++ {
		tx.Lock()
		ks, _, _ := tx.UnsafeRange(buckets.Test, []byte("foo"), nil, 0)
		tx.Unlock()
		if len(ks) != 0 {
			t.Errorf("keys on foo = %v, want nil", ks)
		}
		tx.Commit()
	}
}

func TestBatchTxCommit(t *testing.T) {
	b, _ := betesting.NewTmpBackend(t, time.Hour, 10000)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafePut(buckets.Test, []byte("foo"), []byte("bar"))
	tx.Unlock()

	tx.Commit()

	// check whether put happens via db view
	batch := backend.DbFromBackendForTest(b).NewIndexedBatch()
	defer batch.Close()

	v, _, _ := batch.Get([]byte("test/foo"))
	if v == nil {
		t.Errorf("foo key failed to written in backend")
	}
}

func TestBatchTxBatchLimitCommit(t *testing.T) {
	// start backend with batch limit 1 so one write can
	// trigger a commit
	b, _ := betesting.NewTmpBackend(t, time.Hour, 1)

	tx := b.BatchTx()
	tx.Lock()
	tx.UnsafePut(buckets.Test, []byte("foo"), []byte("bar"))
	tx.Unlock()

	// batch limit commit should have been triggered
	// check whether put happens via db view
	batch := backend.DbFromBackendForTest(b).NewIndexedBatch()
	defer batch.Close()

	v, _, _ := batch.Get([]byte("test/foo"))
	if v == nil {
		t.Errorf("foo key failed to written in backend")
	}
}
