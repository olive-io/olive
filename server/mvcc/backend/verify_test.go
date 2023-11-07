package backend_test

import (
	"os"
	"testing"
	"time"

	"github.com/olive-io/olive/server/mvcc/backend"
	"github.com/olive-io/olive/server/mvcc/backend/testing"
)

func TestLockVerify(t *testing.T) {
	tcs := []struct {
		name                      string
		insideApply               bool
		lock                      func(tx backend.IBatchTx)
		txPostLockInsideApplyHook func()
		expectPanic               bool
	}{
		{
			name:        "call lockInsideApply from inside apply",
			insideApply: true,
			lock:        lockInsideApply,
			expectPanic: false,
		},
		{
			name:        "call lockInsideApply from outside apply (without txPostLockInsideApplyHook)",
			insideApply: false,
			lock:        lockInsideApply,
			expectPanic: false,
		},
		{
			name:                      "call lockInsideApply from outside apply (with txPostLockInsideApplyHook)",
			insideApply:               false,
			lock:                      lockInsideApply,
			txPostLockInsideApplyHook: func() {},
			expectPanic:               true,
		},
		{
			name:        "call lockOutsideApply from outside apply",
			insideApply: false,
			lock:        lockOutsideApply,
			expectPanic: false,
		},
		{
			name:        "call lockOutsideApply from inside apply",
			insideApply: true,
			lock:        lockOutsideApply,
			expectPanic: true,
		},
		{
			name:        "call Lock from unit test",
			insideApply: false,
			lock:        lockFromUT,
			expectPanic: false,
		},
	}
	env := os.Getenv("OLIVE_VERIFY")
	os.Setenv("OLIVE_VERIFY", "lock")
	defer func() {
		os.Setenv("OLIVE_VERIFY", env)
	}()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			be, _ := betesting.NewTmpBackend(t, time.Hour, 10000)
			be.SetTxPostLockInsideApplyHook(tc.txPostLockInsideApplyHook)

			hasPaniced := handlePanic(func() {
				if tc.insideApply {
					applyEntries(be, tc.lock)
				} else {
					tc.lock(be.BatchTx())
				}
			}) != nil
			if hasPaniced != tc.expectPanic {
				t.Errorf("%v != %v", hasPaniced, tc.expectPanic)
			}
		})
	}
}

func handlePanic(f func()) (result interface{}) {
	defer func() {
		result = recover()
	}()
	f()
	return result
}

func applyEntries(be backend.IBackend, f func(tx backend.IBatchTx)) {
	f(be.BatchTx())
}

func lockInsideApply(tx backend.IBatchTx)  { tx.LockInsideApply() }
func lockOutsideApply(tx backend.IBatchTx) { tx.LockOutsideApply() }
func lockFromUT(tx backend.IBatchTx)       { tx.Lock() }
