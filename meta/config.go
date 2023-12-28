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

package meta

import (
	"strings"

	"github.com/spf13/pflag"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/server/v3/embed"

	"github.com/olive-io/olive/pkg/component-base/cli/flags"
	"github.com/olive-io/olive/pkg/component-base/logs"
)

const (
	DefaultName                  = "default"
	DefaultListenerClientAddress = "http://localhost:4379"
	DefaultListenerPeerAddress   = "http://localhost:4380"
	DefaultRegionLimit           = 100
	DefaultRegionDefinitionLimit = 500
)

type Config struct {
	*embed.Config

	fs           *pflag.FlagSet
	clusterState *flags.SelectiveStringValue

	// The maximum number of regions in a runner
	RegionLimit int
	// The maximum number of bpmn definitions in a region
	RegionDefinitionLimit int
}

func NewConfig() Config {
	ec := embed.NewConfig()
	ec.ListenClientUrls, _ = types.NewURLs(strings.Split(DefaultListenerClientAddress, ","))
	ec.AdvertiseClientUrls = ec.ListenClientUrls
	ec.ListenPeerUrls, _ = types.NewURLs(strings.Split(DefaultListenerPeerAddress, ","))
	ec.AdvertisePeerUrls = ec.ListenPeerUrls

	cfg := Config{
		Config: ec,
		clusterState: flags.NewSelectiveStringValue(
			embed.ClusterStateFlagNew,
			embed.ClusterStateFlagExisting,
		),
		RegionLimit:           DefaultRegionLimit,
		RegionDefinitionLimit: DefaultRegionDefinitionLimit,
	}
	cfg.fs = cfg.newFlags()

	return cfg
}

func (cfg *Config) newFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("meta", pflag.ExitOnError)
	// member
	fs.StringVar(&cfg.Config.Name, "name", DefaultName,
		"Human-readable name for this member.")
	fs.StringVar(&cfg.Config.Dir, "data-dir", DefaultName+".olive",
		"Path to the data directory.")
	fs.StringVar(&cfg.Config.WalDir, "wal-dir", cfg.Config.WalDir, "Path to the dedicated wal directory.")
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerPeerAddress, ""),
		"listen-peer-urls",
		"List of URLs to listen on for peer traffic.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerClientAddress, ""), "listen-client-urls",
		"List of URLs to listen on for client traffic.",
	)
	fs.UintVar(&cfg.Config.MaxSnapFiles, "max-snapshots", cfg.Config.MaxSnapFiles, "Maximum number of snapshot files to retain (0 is unlimited).")
	fs.UintVar(&cfg.Config.MaxWalFiles, "max-wals", cfg.Config.MaxWalFiles, "Maximum number of wal files to retain (0 is unlimited).")
	fs.Uint64Var(&cfg.Config.SnapshotCount, "snapshot-count", cfg.Config.SnapshotCount, "Number of committed transactions to trigger a snapshot to disk.")
	fs.UintVar(&cfg.Config.TickMs, "heartbeat-interval", cfg.Config.TickMs, "Time (in milliseconds) of a heartbeat interval.")
	fs.UintVar(&cfg.Config.ElectionMs, "election-timeout", cfg.Config.ElectionMs, "Time (in milliseconds) for an election to timeout.")
	fs.BoolVar(&cfg.Config.InitialElectionTickAdvance, "initial-election-tick-advance", cfg.Config.InitialElectionTickAdvance, "Whether to fast-forward initial election ticks on boot for faster election.")
	fs.Int64Var(&cfg.Config.QuotaBackendBytes, "quota-backend-bytes", cfg.Config.QuotaBackendBytes, "Raise alarms when backend size exceeds the given quota. 0 means use the default quota.")
	fs.StringVar(&cfg.Config.BackendFreelistType, "backend-bbolt-freelist-type", cfg.Config.BackendFreelistType, "BackendFreelistType specifies the type of freelist that boltdb backend uses(array and map are supported types)")
	fs.DurationVar(&cfg.Config.BackendBatchInterval, "backend-batch-interval", cfg.Config.BackendBatchInterval, "BackendBatchInterval is the maximum time before commit the backend transaction.")
	fs.IntVar(&cfg.Config.BackendBatchLimit, "backend-batch-limit", cfg.Config.BackendBatchLimit, "BackendBatchLimit is the maximum operations before commit the backend transaction.")
	fs.UintVar(&cfg.Config.MaxTxnOps, "max-txn-ops", cfg.Config.MaxTxnOps, "Maximum number of operations permitted in a transaction.")
	fs.UintVar(&cfg.Config.MaxRequestBytes, "max-request-bytes", cfg.Config.MaxRequestBytes, "Maximum client request size in bytes the server will accept.")
	fs.DurationVar(&cfg.Config.GRPCKeepAliveMinTime, "grpc-keepalive-min-time", cfg.Config.GRPCKeepAliveMinTime, "Minimum interval duration that a client should wait before pinging server.")
	fs.DurationVar(&cfg.Config.GRPCKeepAliveInterval, "grpc-keepalive-interval", cfg.Config.GRPCKeepAliveInterval, "Frequency duration of server-to-client ping to check if a connection is alive (0 to disable).")
	fs.DurationVar(&cfg.Config.GRPCKeepAliveTimeout, "grpc-keepalive-timeout", cfg.Config.GRPCKeepAliveTimeout, "Additional duration of wait before closing a non-responsive connection (0 to disable).")
	fs.BoolVar(&cfg.Config.SocketOpts.ReusePort, "socket-reuse-port", cfg.Config.SocketOpts.ReusePort, "Enable to set socket option SO_REUSEPORT on listeners allowing rebinding of a port already in use.")
	fs.BoolVar(&cfg.Config.SocketOpts.ReuseAddress, "socket-reuse-address", cfg.Config.SocketOpts.ReuseAddress, "Enable to set socket option SO_REUSEADDR on listeners allowing binding to an address in `TIME_WAIT` state.")

	fs.StringVar(&cfg.Config.InitialCluster, "initial-cluster", cfg.Config.InitialClusterToken,
		"Initial cluster configuration for bootstrapping.")
	fs.Var(cfg.clusterState, "initial-cluster-state", "Initial cluster state ('new' or 'existing').")

	fs.Uint32Var(&cfg.Config.MaxConcurrentStreams, "max-concurrent-streams", cfg.Config.MaxConcurrentStreams,
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

	fs.StringVar(&cfg.Config.AutoCompactionRetention, "auto-compaction-retention", "0", "Auto compaction retention for mvcc key value store. 0 means disable auto compaction.")
	fs.StringVar(&cfg.Config.AutoCompactionMode, "auto-compaction-mode", "periodic", "interpret 'auto-compaction-retention' one of: periodic|revision. 'periodic' for duration based retention, defaulting to hours if no time unit is provided (e.g. '5m'). 'revision' for revision number based retention.")

	// logging
	fs.StringVar(&cfg.Config.Logger, "logger", "zap",
		"Currently only supports 'zap' for structured logging.")
	fs.Var(flags.NewUniqueStringsValue(logs.DefaultLogOutput), "log-outputs",
		"Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.")
	fs.StringVar(&cfg.Config.LogLevel, "log-level", logs.DefaultLogLevel,
		"Configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.")
	fs.BoolVar(&cfg.Config.EnableLogRotation, "enable-log-rotation", false,
		"Enable log rotation of a single log-outputs file target.")
	fs.StringVar(&cfg.Config.LogRotationConfigJSON, "log-rotation-config-json", logs.DefaultLogRotationConfig,
		"Configures log rotation if enabled with a JSON logger config. Default: MaxSize=100(MB), MaxAge=0(days,no limit), MaxBackups=0(no limit), LocalTime=false(UTC), Compress=false(gzip)")

	return fs
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

// TestConfig get Config for testing
func TestConfig() (Config, func()) {
	cfg := NewConfig()

	cancel := func() {}

	return cfg, cancel
}

func (cfg *Config) Parse() error {
	return cfg.configFromCmdLine()
}

func (cfg *Config) Validate() (err error) {
	if err = cfg.Config.Validate(); err != nil {
		return
	}

	return
}

func (cfg *Config) configFromCmdLine() error {
	var err error

	cfg.Config.ListenPeerUrls = flags.UniqueURLsFromFlag(cfg.fs, "listen-peer-urls")
	cfg.Config.AdvertisePeerUrls = flags.UniqueURLsFromFlag(cfg.fs, "initial-advertise-peer-urls")
	cfg.Config.ListenClientUrls = flags.UniqueURLsFromFlag(cfg.fs, "listen-client-urls")
	cfg.Config.AdvertiseClientUrls = flags.UniqueURLsFromFlag(cfg.fs, "advertise-client-urls")

	cfg.Config.MaxConcurrentStreams, err = cfg.fs.GetUint32("max-concurrent-streams")
	if err != nil {
		return err
	}

	cfg.Config.LogOutputs = flags.UniqueStringsFromFlag(cfg.fs, "log-outputs")

	cfg.Config.ClusterState = cfg.clusterState.String()

	return nil
}
