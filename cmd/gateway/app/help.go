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

package app

import (
	"strings"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/gateway"
)

var (
	usageline = `Usage:

  olive-gateway [flags]
    Start an olive-gateway server.

  olive-gateway --version
    Show the version of olive-gateway.

  olive-gateway -h | --help
    Show the help information about olive-gateway.

  olive-gateway --config-file
    Path to the server configuration file. Note that if a configuration file is provided, other command line flags and environment variables will be ignored.`

	flagsline = `
Gateway:
  --id 'gateway'
    Set Gateway Id
  --openapiv3 
    Set Path of openapi v3 docs
  --data-dir 'default'
    Set the Path to the data directory.
  --endpoints [` + strings.Join(client.DefaultEndpoints, ",") + `]
    Set gRPC endpoints to connect the cluster of olive-meta
  --listen-url '` + gateway.DefaultListenURL + `'
    Set the URL to listen on for gRPC traffic.
  --advertise-url
    Set advertise URL to listen on for gRPC traffic.
  --register-interval '` + gateway.DefaultRegisterInterval.String() + `'
    Set Register interval.
  --register-ttl '` + gateway.DefaultRegisterTTL.String() + `'
    Set Register ttl.
  --enable-grpc-gateway
    EnableGRPCGateway enables grpc gateway. The gateway translates a RESTful HTTP API into gRPC.

Logging:
  --log-outputs 'default'
    Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.
  --log-level 'info'
    Configures log level. Only supports debug, info, warn, error, panic, or fatal.
  --enable-log-rotation 'false'
    Enable log rotation of a single log-outputs file target.
  --log-rotation-config-json '{"maxsize": 100, "maxage": 0, "maxbackups": 0, "localtime": false, "compress": false}'
    Configures log rotation if enabled with a JSON logger config. MaxSize(MB), MaxAge(days,0=no limit), MaxBackups(0=no limit), LocalTime(use computers local time), Compress(gzip)".`
)
