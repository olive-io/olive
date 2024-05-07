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

package client

import (
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Id       string `json:"id"`
	Address  string `json:"address"`
	Endpoint string `json:"endpoint"`
	// Timeout(s) the timeout of grpc calling
	Timeout time.Duration `json:"timeout"`
	// Interval
	Interval time.Duration `json:"interval"`
	// InjectTTL
	TTL time.Duration `json:"ttl"`

	Logger *zap.Logger `json:"-"`
}
