// Copyright 2021 The etcd Authors
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

package betesting

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oliveio/olive/pkg/mvcc/backend"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func NewTmpBackendFromCfg(t testing.TB, bcfg backend.BackendConfig) (backend.IBackend, string) {
	dir, err := os.MkdirTemp(t.TempDir(), "etcd_backend_test")
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
