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
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/runner/backend"
	betesting "github.com/olive-io/olive/runner/backend/testing"
	"github.com/olive-io/olive/runner/buckets"
)

var (
	bucket = buckets.Test
	key    = []byte("key")
)

func TestBackendPreCommitHook(t *testing.T) {
	be := newTestHooksBackend(t, backend.DefaultBackendConfig())

	tx := be.BatchTx()
	prepareBuckenAndKey(tx)
	tx.Commit()

	// Empty commit.
	tx.Commit()

	assert.Equal(t, ">cc", getCommitsKey(t, be), "expected 2 explict commits")
	tx.Commit()
	assert.Equal(t, ">ccc", getCommitsKey(t, be), "expected 3 explict commits")
}

func TestBackendAutoCommitLimitHook(t *testing.T) {
	cfg := backend.DefaultBackendConfig()
	cfg.BatchLimit = 3
	be := newTestHooksBackend(t, cfg)

	tx := be.BatchTx()
	prepareBuckenAndKey(tx) // writes 2 entries.

	for i := 3; i <= 9; i++ {
		write(tx, []byte("i"), []byte{byte(i)})
	}

	assert.Equal(t, ">ccc", getCommitsKey(t, be))
}

func write(tx backend.IBatchTx, k, v []byte) {
	tx.Lock()
	defer tx.Unlock()
	tx.UnsafePut(bucket, k, v)
}

func TestBackendAutoCommitBatchIntervalHook(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	cfg := backend.DefaultBackendConfig()
	cfg.BatchInterval = 10 * time.Millisecond
	be := newTestHooksBackend(t, cfg)
	tx := be.BatchTx()
	prepareBuckenAndKey(tx)

	// Edits trigger an auto-commit
	waitUntil(ctx, t, func() bool { return getCommitsKey(t, be) == ">c" })

	time.Sleep(time.Second)
	// No additional auto-commits, as there were no more edits
	assert.Equal(t, ">c", getCommitsKey(t, be))

	write(tx, []byte("foo"), []byte("bar1"))

	waitUntil(ctx, t, func() bool { return getCommitsKey(t, be) == ">cc" })

	write(tx, []byte("foo"), []byte("bar1"))

	waitUntil(ctx, t, func() bool { return getCommitsKey(t, be) == ">ccc" })
}

func waitUntil(ctx context.Context, t testing.TB, f func() bool) {
	for !f() {
		select {
		case <-ctx.Done():
			t.Fatalf("Context cancelled/timedout without condition met: %v", ctx.Err())
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func prepareBuckenAndKey(tx backend.IBatchTx) {
	tx.Lock()
	defer tx.Unlock()
	tx.UnsafeCreateBucket(bucket)
	tx.UnsafePut(bucket, key, []byte(">"))
}

func newTestHooksBackend(t testing.TB, baseConfig backend.BackendConfig) backend.IBackend {
	cfg := baseConfig
	hook := backend.NewHooks(func(tx backend.IBatchTx) {
		k, v, _ := tx.UnsafeRange(bucket, key, nil, 1)
		t.Logf("OnPreCommit executed: %v %v", string(k[0]), string(v[0]))
		assert.Len(t, k, 1)
		assert.Len(t, v, 1)
		tx.UnsafePut(bucket, key, append(v[0], byte('c')))
	})

	be, _ := betesting.NewTmpBackendFromCfg(t, cfg)
	be.AppendHooks(hook)
	t.Cleanup(func() {
		betesting.Close(t, be)
	})
	return be
}

func getCommitsKey(t testing.TB, be backend.IBackend) string {
	rtx := be.BatchTx()
	rtx.Lock()
	defer rtx.Unlock()
	_, v, _ := rtx.UnsafeRange(bucket, key, nil, 1)
	assert.Len(t, v, 1)
	return string(v[0])
}
