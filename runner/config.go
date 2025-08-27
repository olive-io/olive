/*
Copyright 2025 The olive Authors

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
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	DefaultHeartbeat = time.Second * 30
)

type Config struct {
	Logger *zap.Logger

	UID  string `json:"uid"`
	Name string `json:"name"`

	HeartbeatDuration time.Duration `json:"heartbeat"`
}

func NewConfig(lg *zap.Logger, name string) *Config {
	cfg := &Config{
		Logger: lg,
		UID:    uuid.New().String(),
		Name:   name,

		HeartbeatDuration: DefaultHeartbeat,
	}
	return cfg
}

func (cfg *Config) Validate() error {
	if cfg.Logger == nil {
		return errors.New("missing logger")
	}
	if cfg.UID == "" {
		return errors.New("missing uid")
	}
	if cfg.Name == "" {
		return errors.New("missing name")
	}
	return nil
}
