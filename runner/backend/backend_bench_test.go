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
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	betesting "github.com/olive-io/olive/runner/backend/testing"
	"github.com/olive-io/olive/runner/buckets"
)

func BenchmarkBackendPut(b *testing.B) {
	backend, _ := betesting.NewTmpBackend(b, 100*time.Millisecond, 10000)
	defer betesting.Close(b, backend)

	// prepare keys
	keys := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = make([]byte, 64)
		_, err := rand.Read(keys[i])
		assert.NoError(b, err)
	}
	value := make([]byte, 128)
	_, err := rand.Read(value)
	assert.NoError(b, err)

	batchTx := backend.BatchTx()

	batchTx.Lock()
	batchTx.UnsafeCreateBucket(buckets.Test)
	batchTx.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batchTx.Lock()
		batchTx.UnsafePut(buckets.Test, keys[i], value)
		batchTx.Unlock()
	}
}
