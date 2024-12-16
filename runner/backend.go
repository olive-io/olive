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

	"github.com/olive-io/olive/runner/config"
	"github.com/olive-io/olive/runner/storage/backend"
)

func newBackend(cfg *config.Config) (backend.IBackend, error) {
	bcfg := &backend.Config{
		Dir:       cfg.DBDir(),
		CacheSize: int64(cfg.CacheSize),
		Logger:    cfg.GetLogger(),
	}

	if cfg.BackendGCInterval != 0 {
		bcfg.GCInterval = cfg.BackendGCInterval
		if cfg.GetLogger() != nil {
			cfg.GetLogger().Info("setting backend gc interval", zap.Duration("batch interval", cfg.BackendGCInterval))
		}
	}

	return backend.NewBackend(bcfg)
}
