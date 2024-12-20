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

package main

import (
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"

	"github.com/olive-io/olive/cmd/plane/app"
	"github.com/olive-io/olive/pkg/cliutil"
	"github.com/olive-io/olive/plane/options"
)

func main() {
	defaultOptions := options.NewServerOptions(os.Stdout, os.Stderr)
	ctx := genericapiserver.SetupSignalContext()
	command := app.NewPlaneServer(ctx, defaultOptions, false)
	os.Exit(cliutil.Run(command))
}
