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

package server

import (
	"time"

	"github.com/olive-io/olive/server/config"
	"github.com/olive-io/olive/server/mvcc/backend"
	"go.uber.org/zap"
)

func newBackend(cfg config.ServerConfig, hooks backend.IHooks) backend.IBackend {
	bcfg := backend.DefaultBackendConfig()
	bcfg.Dir = cfg.BackendPath()
	if cfg.CacheSize != 0 {
		bcfg.CacheSize = cfg.CacheSize
		if cfg.Logger != nil {
			cfg.Logger.GetLogger().Info("setting backend cache size", zap.Int64("cache size", cfg.CacheSize))
		}
	}
	if cfg.BackendBatchLimit != 0 {
		bcfg.BatchLimit = cfg.BackendBatchLimit
		if cfg.Logger != nil {
			cfg.Logger.GetLogger().Info("setting backend batch limit", zap.Int("batch limit", cfg.BackendBatchLimit))
		}
	}
	if cfg.BackendBatchInterval != 0 {
		bcfg.BatchInterval = cfg.BackendBatchInterval
		if cfg.Logger != nil {
			cfg.Logger.GetLogger().Info("setting backend batch interval", zap.Duration("batch interval", cfg.BackendBatchInterval))
		}
	}
	bcfg.Logger = cfg.Logger.GetLogger()
	be := backend.New(bcfg)
	if hooks != nil {
		be.AppendHooks(hooks)
	}
	return be
}

// openBackend returns a backend using the current etcd db.
func openBackend(cfg config.ServerConfig, hooks backend.IHooks) backend.IBackend {
	fn := cfg.BackendPath()

	now, beOpened := time.Now(), make(chan backend.IBackend)
	go func() {
		beOpened <- newBackend(cfg, hooks)
	}()

	select {
	case be := <-beOpened:
		cfg.Logger.GetLogger().Info("opened backend db", zap.String("path", fn), zap.Duration("took", time.Since(now)))
		return be

	case <-time.After(10 * time.Second):
		cfg.Logger.GetLogger().Info(
			"db file is flocked by another process, or taking too long",
			zap.String("path", fn),
			zap.Duration("took", time.Since(now)),
		)
	}

	return <-beOpened
}
