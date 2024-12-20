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
Runner:
  --name '` + runner.DefaultName + `
    Set name of olive-runner.
  --data-dir 'default'
    Path to the data directory.
  --oliveconfig '` + runner.DefaultConfigPath + `'
    Set the file path from configuration the cluster of olive-plane
  --listen-client-urls '` + runner.DefaultListenURL + `'
    List of URLs to listen on for client traffic.
  --advertise-client-url '
    Set advertise URL to listen on for client traffic.

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
