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

package options

import (
	"github.com/spf13/pflag"

	"github.com/olive-io/olive/mon/embed"
	"github.com/olive-io/olive/pkg/cliutil/flags"
)

type EmbedEtcdOptions struct {
	*embed.Config

	ClusterState *flags.SelectiveStringValue
}

func NewEmbedEtcdOptions() *EmbedEtcdOptions {
	etcdCfg := embed.NewConfig()
	options := &EmbedEtcdOptions{
		Config: etcdCfg,
		ClusterState: flags.NewSelectiveStringValue(
			embed.ClusterStateFlagNew,
			embed.ClusterStateFlagExisting,
		),
	}
	return options
}

func (o *EmbedEtcdOptions) Validate() []error {
	allErrors := []error{}
	if err := o.Config.Validate(); err != nil {
		allErrors = append(allErrors, err)
	}
	return allErrors
}

func (o *EmbedEtcdOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Config.Name, "name", DefaultName,
		"Human-readable name for this member.")
	fs.StringVar(&o.Config.Dir, "data-dir", DefaultName+".olive",
		"Path to the data directory.")
	fs.StringVar(&o.Config.WalDir, "wal-dir", o.Config.WalDir, "Path to the dedicated wal directory.")
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerPeerAddress, ""),
		"listen-peer-urls",
		"List of URLs to listen on for peer traffic.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerClientAddress, ""), "listen-client-urls",
		"List of URLs to listen on for client traffic.",
	)
	fs.UintVar(&o.Config.MaxSnapFiles, "max-snapshots", o.Config.MaxSnapFiles, "Maximum number of snapshot files to retain (0 is unlimited).")
	fs.UintVar(&o.Config.MaxWalFiles, "max-wals", o.Config.MaxWalFiles, "Maximum number of wal files to retain (0 is unlimited).")
	fs.Uint64Var(&o.Config.SnapshotCount, "snapshot-count", o.Config.SnapshotCount, "Number of committed transactions to trigger a snapshot to disk.")
	fs.UintVar(&o.Config.TickMs, "heartbeat-interval", o.Config.TickMs, "Time (in milliseconds) of a heartbeat interval.")
	fs.UintVar(&o.Config.ElectionMs, "election-timeout", o.Config.ElectionMs, "Time (in milliseconds) for an election to timeout.")
	fs.BoolVar(&o.Config.InitialElectionTickAdvance, "initial-election-tick-advance", o.Config.InitialElectionTickAdvance, "Whether to fast-forward initial election ticks on boot for faster election.")
	fs.Int64Var(&o.Config.QuotaBackendBytes, "quota-backend-bytes", o.Config.QuotaBackendBytes, "Raise alarms when backend size exceeds the given quota. 0 means use the default quota.")
	fs.StringVar(&o.Config.BackendFreelistType, "backend-bbolt-freelist-type", o.Config.BackendFreelistType, "BackendFreelistType specifies the type of freelist that boltdb backend uses(array and map are supported types)")
	fs.DurationVar(&o.Config.BackendBatchInterval, "backend-batch-interval", o.Config.BackendBatchInterval, "BackendBatchInterval is the maximum time before commit the backend transaction.")
	fs.IntVar(&o.Config.BackendBatchLimit, "backend-batch-limit", o.Config.BackendBatchLimit, "BackendBatchLimit is the maximum operations before commit the backend transaction.")
	fs.UintVar(&o.Config.MaxTxnOps, "max-txn-ops", o.Config.MaxTxnOps, "Maximum number of operations permitted in a transaction.")
	fs.UintVar(&o.Config.MaxRequestBytes, "max-request-bytes", o.Config.MaxRequestBytes, "Maximum client request size in bytes the server will accept.")
	fs.DurationVar(&o.Config.GRPCKeepAliveMinTime, "grpc-keepalive-min-time", o.Config.GRPCKeepAliveMinTime, "Minimum interval duration that a client should wait before pinging server.")
	fs.DurationVar(&o.Config.GRPCKeepAliveInterval, "grpc-keepalive-interval", o.Config.GRPCKeepAliveInterval, "Frequency duration of server-to-client ping to check if a connection is alive (0 to disable).")
	fs.DurationVar(&o.Config.GRPCKeepAliveTimeout, "grpc-keepalive-timeout", o.Config.GRPCKeepAliveTimeout, "Additional duration of wait before closing a non-responsive connection (0 to disable).")
	fs.BoolVar(&o.Config.SocketOpts.ReusePort, "socket-reuse-port", o.Config.SocketOpts.ReusePort, "Enable to set socket option SO_REUSEPORT on listeners allowing rebinding of a port already in use.")
	fs.BoolVar(&o.Config.SocketOpts.ReuseAddress, "socket-reuse-address", o.Config.SocketOpts.ReuseAddress, "Enable to set socket option SO_REUSEADDR on listeners allowing binding to an address in `TIME_WAIT` state.")

	fs.StringVar(&o.Config.InitialCluster, "initial-cluster", o.Config.InitialCluster,
		"Initial cluster configuration for bootstrapping.")
	fs.Var(o.ClusterState, "initial-cluster-state", "Initial cluster state ('new' or 'existing').")

	fs.Uint32Var(&o.Config.MaxConcurrentStreams, "max-concurrent-streams", o.Config.MaxConcurrentStreams,
		"Maximum concurrent streams that each client can open at a time.")

	// clustering
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerPeerAddress, ""),
		"initial-advertise-peer-urls",
		"List of this member's peer URLs to advertise to the rest of the cluster.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerClientAddress, ""),
		"advertise-client-urls",
		"List of this member's client URLs to advertise to the public.",
	)

	fs.StringVar(&o.Config.AutoCompactionRetention, "auto-compaction-retention", "0", "Auto compaction retention for mvcc key value store. 0 means disable auto compaction.")
	fs.StringVar(&o.Config.AutoCompactionMode, "auto-compaction-mode", "periodic", "interpret 'auto-compaction-retention' one of: periodic|revision. 'periodic' for duration based retention, defaulting to hours if no time unit is provided (e.g. '5m'). 'revision' for revision number based retention.")
}

func (o *EmbedEtcdOptions) ApplyTo(c *embed.Config) error {
	*c = *o.Config
	return nil
}

func (o *EmbedEtcdOptions) ApplyToEtcdStorage(c *EtcdStorageOptions) error {
	etcdServerList := []string{}
	etcdAdvertiseClientURLs := o.AdvertiseClientUrls
	if len(etcdAdvertiseClientURLs) == 0 {
		etcdAdvertiseClientURLs = o.ListenClientUrls
	}
	for _, url := range etcdAdvertiseClientURLs {
		etcdServerList = append(etcdServerList, url.Host)
	}
	c.StorageConfig.Transport.ServerList = etcdServerList
	return nil
}
