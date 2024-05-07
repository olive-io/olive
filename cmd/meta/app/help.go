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
	"fmt"
	"strconv"

	"go.etcd.io/etcd/server/v3/embed"

	"github.com/olive-io/olive/meta"
)

var (
	usageline = `Usage:

  olive-meta [flags]
    Start an olive-meta server.

  olive-meta --version
    Show the version of olive-meta.

  olive-meta -h | --help
    Show the help information about olive-meta.

  olive-meta --config-file
    Path to the server configuration file. Note that if a configuration file is provided, other command line flags and environment variables will be ignored.
`

	flagsline = `
Member:
  --name 'default'
    Human-readable name for this member.
  --data-dir '${name}.olive'
    Path to the data directory.
  --wal-dir ''
    Path to the dedicated wal directory.
  --snapshot-count '100000'
    Number of committed transactions to trigger a snapshot to disk.
  --heartbeat-interval '100'
    Time (in milliseconds) of a heartbeat interval.
  --election-timeout '1000'
    Time (in milliseconds) for an election to timeout. See tuning documentation for details.
  --initial-election-tick-advance 'true'
    Whether to fast-forward initial election ticks on boot for faster election.
  --listen-peer-urls 'http://localhost:4380'
    List of URLs to listen on for peer traffic.
  --listen-client-urls 'http://localhost:4379'
    List of URLs to listen on for client traffic.
  --max-snapshots '` + strconv.Itoa(embed.DefaultMaxSnapshots) + `'
    Maximum number of snapshot files to retain (0 is unlimited).
  --max-wals '` + strconv.Itoa(embed.DefaultMaxWALs) + `'
    Maximum number of wal files to retain (0 is unlimited).
  --quota-backend-bytes '0'
    Raise alarms when backend size exceeds the given quota (0 defaults to low space quota).
  --backend-bbolt-freelist-type 'map'
    BackendFreelistType specifies the type of freelist that boltdb backend uses(array and map are supported types).
  --backend-batch-interval ''
    BackendBatchInterval is the maximum time before commit the backend transaction.
  --backend-batch-limit '0'
    BackendBatchLimit is the maximum operations before commit the backend transaction.
  --max-txn-ops '128'
    Maximum number of operations permitted in a transaction.
  --max-request-bytes '1572864'
    Maximum client request size in bytes the server will accept.
  --max-concurrent-streams 'math.MaxUint32'
    Maximum concurrent streams that each client can open at a time.
  --grpc-keepalive-min-time '5s'
    Minimum duration interval that a client should wait before pinging server.
  --grpc-keepalive-interval '2h'
    Frequency duration of server-to-client ping to check if a connection is alive (0 to disable).
  --grpc-keepalive-timeout '20s'
    Additional duration of wait before closing a non-responsive connection (0 to disable).
  --socket-reuse-port 'false'
    Enable to set socket option SO_REUSEPORT on listeners allowing rebinding of a port already in use.
  --socket-reuse-address 'false'
	Enable to set socket option SO_REUSEADDR on listeners allowing binding to an address in TIME_WAIT state.

Clustering:
  --initial-advertise-peer-urls 'http://localhost:4380'
    List of this member's peer URLs to advertise to the rest of the cluster.
  --initial-cluster 'default=http://localhost:4380'
    Initial cluster configuration for bootstrapping.
  --initial-cluster-state 'new'
    Initial cluster state ('new' or 'existing').
  --initial-cluster-token 'olive-cluster'
    Initial cluster token for the olive cluster during bootstrap.
    Specifying this can protect you from unintended cross-cluster interaction when running multiple clusters.
  --advertise-client-urls 'http://localhost:4379'
    List of this member's client URLs to advertise to the public.
    The client URLs advertised should be accessible to machines that talk to olive cluster. olive client libraries parse these URLs to connect to the cluster.
  --pre-vote 'true'
    Enable to run an additional Raft election phase.
  --auto-compaction-retention '0'
    Auto compaction retention length. 0 means disable auto compaction.
  --auto-compaction-mode 'periodic'
    Interpret 'auto-compaction-retention' one of: periodic|revision. 'periodic' for duration based retention, defaulting to hours if no time unit is provided (e.g. '5m'). 'revision' for revision number based retention.

Regions:
  --region-limit '` + fmt.Sprintf("%d", meta.DefaultRegionLimit) + `'
    Sets the maximum number of regions in one runner
  --region-definitions-limit '` + fmt.Sprintf("%d", meta.DefaultRegionDefinitionsLimit) + `'
    Sets the maximum number of bpmn definitions in one runner-region

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
