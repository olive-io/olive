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
	"os"
	"testing"
	"time"

	"github.com/olive-io/olive/runner/backend"
	betesting "github.com/olive-io/olive/runner/backend/testing"
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
