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

package mon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/mon/config"
	genericserver "github.com/olive-io/olive/pkg/server"
)

func TestNewOliveMonServer(t *testing.T) {
	cfg, cancel := config.TestConfig()
	if !assert.NoError(t, cfg.Validate()) {
		return
	}
	defer cancel()

	s, err := New(cfg)
	if !assert.NoError(t, err) {
		return
	}

	err = s.Start(genericserver.SetupSignalContext())
	if !assert.NoError(t, err) {
		return
	}
}
