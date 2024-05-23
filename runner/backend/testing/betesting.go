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

package betesting

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	"github.com/olive-io/olive/runner/backend"
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
