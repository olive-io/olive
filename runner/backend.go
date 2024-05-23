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

package runner

import (
	"go.uber.org/zap"

	"github.com/olive-io/olive/runner/backend"
)

func newBackend(cfg *Config) backend.IBackend {
	bcfg := backend.BackendConfig{
		Dir:       cfg.DBDir(),
		WAL:       cfg.WALDir(),
		CacheSize: cfg.CacheSize,
		Logger:    cfg.GetLogger(),
	}

	if cfg.BackendBatchInterval != 0 {
		bcfg.BatchInterval = cfg.BackendBatchInterval
		if cfg.GetLogger() != nil {
			cfg.GetLogger().Info("setting backend batch interval", zap.Duration("batch interval", cfg.BackendBatchInterval))
		}
	}
	if cfg.BackendBatchLimit != 0 {
		bcfg.BatchLimit = cfg.BackendBatchLimit
		if cfg.GetLogger() != nil {
			cfg.GetLogger().Info("setting backend batch limit", zap.Int("batch limit", cfg.BackendBatchLimit))
		}
	}

	return backend.New(bcfg)
}
