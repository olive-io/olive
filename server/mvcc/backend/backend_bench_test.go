package backend_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/olive-io/olive/server/mvcc/backend/testing"
	"github.com/olive-io/olive/server/mvcc/buckets"
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

	batchTx := backend.IBatchTx()

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
