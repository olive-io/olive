package betesting

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/olive-io/olive/runner/backend"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func NewTmpBackendFromCfg(t testing.TB, bcfg backend.BackendConfig) (backend.IBackend, string) {
	dir, err := os.MkdirTemp(t.TempDir(), "olive_backend_test")
	if err != nil {
		panic(err)
	}
	tmpPath := filepath.Join(dir, "database")
	bcfg.Dir = tmpPath
	bcfg.Logger = zaptest.NewLogger(t)
	return backend.New(bcfg), tmpPath
}

// NewTmpBackend creates a backend implementation for testing.
func NewTmpBackend(t testing.TB, batchInterval time.Duration, batchLimit int) (backend.IBackend, string) {
	bcfg := backend.DefaultBackendConfig()
	bcfg.BatchInterval, bcfg.BatchLimit = batchInterval, batchLimit
	return NewTmpBackendFromCfg(t, bcfg)
}

func NewDefaultTmpBackend(t testing.TB) (backend.IBackend, string) {
	return NewTmpBackendFromCfg(t, backend.DefaultBackendConfig())
}

func Close(t testing.TB, b backend.IBackend) {
	assert.NoError(t, b.Close())
}
