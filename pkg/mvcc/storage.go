// Copyright 2023 Lack (xingyys@gmail.com).
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

package mvcc

import (
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/oliveio/olive/pkg/mvcc/backend"
	"go.uber.org/zap"
)

const (
	defaultStorageDir       = "default"
	defaultStorageCacheSize = 1024 * 1024 * 10

	defaultBatchLimit    = 10000
	defaultBatchInterval = 100 * time.Millisecond
)

type IStorage interface {
}

type StorageConfig struct {
	// Dir is the file path to the storage file.
	Dir string
	// CacheSize is the size to pebble cache
	CacheSize int64
	// BatchInterval is the maximum time before flushing the BatchTx.
	BatchInterval time.Duration
	// BatchLimit is the maximum puts before flushing the BatchTx.
	BatchLimit int
	// Logger logs backend-side operations.
	Logger *zap.Logger
	// UnsafeNoFsync disables all uses of fsync.
	UnsafeNoFsync bool `json:"unsafe-no-fsync"`
	// Mlock prevents backend database file to be swapped
	Mlock bool

	// Hooks are getting executed during lifecycle of Backend's transactions.
	Hooks backend.IHooks
}

func NewStorageConfig() *StorageConfig {
	cfg := &StorageConfig{
		Dir:           defaultStorageDir,
		CacheSize:     defaultStorageCacheSize,
		BatchInterval: defaultBatchInterval,
		BatchLimit:    defaultBatchLimit,
	}

	return cfg
}

type GetDB func() (*pebble.DB, func())

type EmbedStorage struct {
	lg *zap.Logger

	db *pebble.DB

	closeCh chan struct{}
}

func NewEmbedStorage(cfg StorageConfig) (*EmbedStorage, error) {
	options := &pebble.Options{
		Cache: pebble.NewCache(cfg.CacheSize),
	}
	db, err := pebble.Open(cfg.Dir, options)
	if err != nil {
		return nil, err
	}

	es := &EmbedStorage{
		lg:      cfg.Logger,
		db:      db,
		closeCh: make(chan struct{}, 1),
	}

	return es, nil
}

func (es *EmbedStorage) NewSession() (*pebble.DB, func()) {
	cancel := func() {}
	return es.db, cancel
}

func (es *EmbedStorage) Close() error {
	select {
	case <-es.closeCh:
		return nil
	default:
		close(es.closeCh)
		return es.db.Close()
	}
}
