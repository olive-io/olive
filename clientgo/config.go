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

package clientgo

import (
	"time"
)

const (
	DefaultTimeout = time.Second * 30
)

type ConfigTLS struct {
	CertFile string
	KeyFile  string
	CaFile   string
}

type Config struct {
	Target         string
	DialTimeout    time.Duration
	RequestTimeout time.Duration

	TLS *ConfigTLS
}

func NewConfig(target string) *Config {
	opts := &Config{
		Target:         target,
		DialTimeout:    DefaultTimeout,
		RequestTimeout: DefaultTimeout,
	}
	return opts
}
