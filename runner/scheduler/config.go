/*
Copyright 2024 The olive Authors

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

package scheduler

import (
	"context"

	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.uber.org/zap"

	"github.com/olive-io/olive/runner/storage"
)

const (
	DefaultPoolSize = 100
)

type Config struct {
	Context  context.Context
	Logger   *zap.Logger
	idGen    *idutil.Generator
	Storage  storage.Storage
	PoolSize int
}

func NewConfig(ctx context.Context, logger *zap.Logger, idGen *idutil.Generator, bs storage.Storage) *Config {
	cfg := &Config{
		Context:  ctx,
		Logger:   logger,
		idGen:    idGen,
		Storage:  bs,
		PoolSize: DefaultPoolSize,
	}

	return cfg
}
