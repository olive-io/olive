// Copyright 2022 The etcd Authors
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
	"os"
	"testing"
	"time"

	"github.com/oliveio/olive/pkg/mvcc/backend"
	betesting "github.com/oliveio/olive/pkg/mvcc/backend/testing"
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
