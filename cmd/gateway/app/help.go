// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"strings"

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
Execute:
  --id 'gateway'
    Set Gateway Id
  --openapiv3 
    Set Path of openapi v3 docs
  --data-dir 'default'
    Set the Path to the data directory.
  --endpoints [` + strings.Join(gateway.DefaultEndpoints, ",") + `]
    Set gRPC endpoints to connect the cluster of olive-meta
  --listen-url '` + gateway.DefaultListenURL + `'
    Set the URL to listen on for gRPC traffic.
  --advertise-url
    Set advertise URL to listen on for gRPC traffic.
  --register-interval '` + gateway.DefaultRegisterInterval.String() + `'
    Set Register interval.
  --register-ttl '` + gateway.DefaultRegisterTTL.String() + `'
    Set Register ttl.

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
