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

package app

import (
	"fmt"

	"github.com/olive-io/olive/runner/delegate"
	delegateGRPC "github.com/olive-io/olive/runner/delegate/service/grpc"
	delegateHttp "github.com/olive-io/olive/runner/delegate/service/http"
)

func registerPlugins() error {
	serviceTaskForHttp := delegateHttp.New()
	if err := delegate.RegisterDelegate(serviceTaskForHttp); err != nil {
		return fmt.Errorf("could not register delegate (%s): %w", serviceTaskForHttp.GetTheme().String(), err)
	}
	serviceTaskForGRPC := delegateGRPC.New()
	if err := delegate.RegisterDelegate(serviceTaskForGRPC); err != nil {
		return fmt.Errorf("could not register delegate (%s): %w", serviceTaskForGRPC.GetTheme().String(), err)
	}

	return nil
}
