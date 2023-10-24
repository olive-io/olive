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

package storage

import (
	"github.com/cockroachdb/pebble"
	"github.com/oliveio/olive/pkg/config"
	"go.uber.org/zap"
)

type GetDB func() (*pebble.DB, func())

type EmbedStorage struct {
	*config.StorageConfig

	lg *zap.Logger

	db *pebble.DB
}

func NewEmbedStorage(lg *zap.Logger, cfg *config.StorageConfig) (*EmbedStorage, error) {
	options := &pebble.Options{
		Cache: pebble.NewCache(cfg.CacheSize),
	}
	db, err := pebble.Open(cfg.Dir, options)
	if err != nil {
		return nil, err
	}

	es := &EmbedStorage{
		StorageConfig: cfg,
		lg:            lg,
		db:            db,
	}

	return es, nil
}

func (es *EmbedStorage) NewSession() (*pebble.DB, func()) {
	cancel := func() {}
	return es.db, cancel
}

func (es *EmbedStorage) Close() error {
	return es.db.Close()
}
