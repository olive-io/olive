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

package proxy

import (
	"fmt"

	"go.uber.org/zap"

	dsy "github.com/olive-io/olive/pkg/discovery"
)

var (
	DefaultMaxRecvSize int64 = 1024 * 1024 * 100 // 100Mb
)

type Config struct {
	Logger      *zap.Logger
	Discovery   dsy.IDiscovery
	MaxRecvSize int64
}

func (cfg *Config) validate() error {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.Discovery == nil {
		return fmt.Errorf("missing discovery")
	}
	if cfg.MaxRecvSize == 0 {
		cfg.MaxRecvSize = DefaultMaxRecvSize
	}

	return nil
}
