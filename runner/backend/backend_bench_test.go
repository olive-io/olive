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

package backend_test

import (
	"crypto/rand"
	"testing"
	"time"

	betesting "github.com/olive-io/olive/runner/backend/testing"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/stretchr/testify/assert"
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
