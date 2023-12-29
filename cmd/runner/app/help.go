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

	"github.com/olive-io/olive/runner"
)

var (
	usageline = `Usage:

  olive-runner [flags]
    Start an olive-runner server.

  olive-runner --version
    Show the version of olive-runner.

  olive-runner -h | --help
    Show the help information about olive-runner.

  olive-runner --config-file
    Path to the server configuration file. Note that if a configuration file is provided, other command line flags and environment variables will be ignored.
`

	flagsline = `
Node:
  --data-dir 'default'
    Path to the data directory.
  --endpoints [` + strings.Join(runner.DefaultEndpoints, ",") + `]
    Set gRPC endpoints to connect the cluster of olive-meta
  --listen-client-urls '` + runner.DefaultListenClientURL + `'
    List of URLs to listen on for client traffic.
  --advertise-client-url
    Set advertise URL to listen on for client traffic.

Region:
  --listen-peer-urls '` + runner.DefaultListenPeerURL + `'
    List of URLs to listen on for peer traffic.
  --advertise-peer-url
    Set advertise URL to listen on for peer traffic.

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
