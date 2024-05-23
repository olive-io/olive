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
	"context"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/mon/embed"
	"github.com/olive-io/olive/mon/leader"
	monrest "github.com/olive-io/olive/mon/registry/mon/rest"
	"github.com/olive-io/olive/pkg/cliutil/flags"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
	"github.com/olive-io/olive/pkg/logutil"
)

const (
	DefaultName                   = "default"
	DefaultListenerClientAddress  = "http://localhost:4379"
	DefaultListenerPeerAddress    = "http://localhost:4380"
	DefaultRegionLimit            = 100
	DefaultRegionDefinitionsLimit = 500
	DefaultTokenTTL               = uint(600)
)

type Config struct {
	EtcdConfig    *embed.Config
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   *ExtraConfig
}

func NewConfig() *Config {
	etcdConfig := embed.NewConfig()
	etcdConfig.ListenClientUrls, _ = types.NewURLs(strings.Split(DefaultListenerClientAddress, ","))
	etcdConfig.AdvertiseClientUrls = etcdConfig.ListenClientUrls
	etcdConfig.ListenPeerUrls, _ = types.NewURLs(strings.Split(DefaultListenerPeerAddress, ","))
	etcdConfig.AdvertisePeerUrls = etcdConfig.ListenPeerUrls
	etcdConfig.InitialCluster = DefaultName + "=" + DefaultListenerPeerAddress
	etcdConfig.AuthTokenTTL = DefaultTokenTTL

	genericConfig := genericapiserver.NewRecommendedConfig(apis.Codecs)

	cfg := Config{
		EtcdConfig:    etcdConfig,
		GenericConfig: genericConfig,
		ExtraConfig: &ExtraConfig{
			RegionLimit:            DefaultRegionLimit,
			RegionDefinitionsLimit: DefaultRegionDefinitionsLimit,
		},
	}

	return &cfg
}

func (cfg *Config) newFlags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("mon", pflag.ExitOnError)
	// member
	fs.StringVar(&cfg.EtcdConfig.Name, "name", DefaultName,
		"Human-readable name for this member.")
	fs.StringVar(&cfg.EtcdConfig.Dir, "data-dir", DefaultName+".olive",
		"Path to the data directory.")
	fs.StringVar(&cfg.EtcdConfig.WalDir, "wal-dir", cfg.EtcdConfig.WalDir, "Path to the dedicated wal directory.")
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerPeerAddress, ""),
		"listen-peer-urls",
		"List of URLs to listen on for peer traffic.",
	)
	fs.Var(
		flags.NewUniqueURLsWithExceptions(DefaultListenerClientAddress, ""), "listen-client-urls",
		"List of URLs to listen on for client traffic.",
	)
	fs.UintVar(&cfg.EtcdConfig.MaxSnapFiles, "max-snapshots", cfg.EtcdConfig.MaxSnapFiles, "Maximum number of snapshot files to retain (0 is unlimited).")
	fs.UintVar(&cfg.EtcdConfig.MaxWalFiles, "max-wals", cfg.EtcdConfig.MaxWalFiles, "Maximum number of wal files to retain (0 is unlimited).")
	fs.Uint64Var(&cfg.EtcdConfig.SnapshotCount, "snapshot-count", cfg.EtcdConfig.SnapshotCount, "Number of committed transactions to trigger a snapshot to disk.")
	fs.UintVar(&cfg.EtcdConfig.TickMs, "heartbeat-interval", cfg.EtcdConfig.TickMs, "Time (in milliseconds) of a heartbeat interval.")
	fs.UintVar(&cfg.EtcdConfig.ElectionMs, "election-timeout", cfg.EtcdConfig.ElectionMs, "Time (in milliseconds) for an election to timeout.")
	fs.BoolVar(&cfg.EtcdConfig.InitialElectionTickAdvance, "initial-election-tick-advance", cfg.EtcdConfig.InitialElectionTickAdvance, "Whether to fast-forward initial election ticks on boot for faster election.")
	fs.Int64Var(&cfg.EtcdConfig.QuotaBackendBytes, "quota-backend-bytes", cfg.EtcdConfig.QuotaBackendBytes, "Raise alarms when backend size exceeds the given quota. 0 means use the default quota.")
	fs.StringVar(&cfg.EtcdConfig.BackendFreelistType, "backend-bbolt-freelist-type", cfg.EtcdConfig.BackendFreelistType, "BackendFreelistType specifies the type of freelist that boltdb backend uses(array and map are supported types)")
	fs.DurationVar(&cfg.EtcdConfig.BackendBatchInterval, "backend-batch-interval", cfg.EtcdConfig.BackendBatchInterval, "BackendBatchInterval is the maximum time before commit the backend transaction.")
	fs.IntVar(&cfg.EtcdConfig.BackendBatchLimit, "backend-batch-limit", cfg.EtcdConfig.BackendBatchLimit, "BackendBatchLimit is the maximum operations before commit the backend transaction.")
	fs.UintVar(&cfg.EtcdConfig.MaxTxnOps, "max-txn-ops", cfg.EtcdConfig.MaxTxnOps, "Maximum number of operations permitted in a transaction.")
	fs.UintVar(&cfg.EtcdConfig.MaxRequestBytes, "max-request-bytes", cfg.EtcdConfig.MaxRequestBytes, "Maximum client request size in bytes the server will accept.")
	fs.DurationVar(&cfg.EtcdConfig.GRPCKeepAliveMinTime, "grpc-keepalive-min-time", cfg.EtcdConfig.GRPCKeepAliveMinTime, "Minimum interval duration that a client should wait before pinging server.")
	fs.DurationVar(&cfg.EtcdConfig.GRPCKeepAliveInterval, "grpc-keepalive-interval", cfg.EtcdConfig.GRPCKeepAliveInterval, "Frequency duration of server-to-client ping to check if a connection is alive (0 to disable).")
	fs.DurationVar(&cfg.EtcdConfig.GRPCKeepAliveTimeout, "grpc-keepalive-timeout", cfg.EtcdConfig.GRPCKeepAliveTimeout, "Additional duration of wait before closing a non-responsive connection (0 to disable).")
	fs.BoolVar(&cfg.EtcdConfig.SocketOpts.ReusePort, "socket-reuse-port", cfg.EtcdConfig.SocketOpts.ReusePort, "Enable to set socket option SO_REUSEPORT on listeners allowing rebinding of a port already in use.")
	fs.BoolVar(&cfg.EtcdConfig.SocketOpts.ReuseAddress, "socket-reuse-address", cfg.EtcdConfig.SocketOpts.ReuseAddress, "Enable to set socket option SO_REUSEADDR on listeners allowing binding to an address in `TIME_WAIT` state.")

	fs.StringVar(&cfg.EtcdConfig.InitialCluster, "initial-cluster", cfg.EtcdConfig.InitialCluster,
		"Initial cluster configuration for bootstrapping.")

	fs.Uint32Var(&cfg.EtcdConfig.MaxConcurrentStreams, "max-concurrent-streams", cfg.EtcdConfig.MaxConcurrentStreams,
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

	fs.StringVar(&cfg.EtcdConfig.AutoCompactionRetention, "auto-compaction-retention", "0", "Auto compaction retention for mvcc key value store. 0 means disable auto compaction.")
	fs.StringVar(&cfg.EtcdConfig.AutoCompactionMode, "auto-compaction-mode", "periodic", "interpret 'auto-compaction-retention' one of: periodic|revision. 'periodic' for duration based retention, defaulting to hours if no time unit is provided (e.g. '5m'). 'revision' for revision number based retention.")

	// region
	fs.IntVar(&cfg.ExtraConfig.RegionLimit, "region-limit", cfg.ExtraConfig.RegionLimit, "Sets the maximum number of regions in a runner")
	fs.IntVar(&cfg.ExtraConfig.RegionDefinitionsLimit, "region-definitions-limit", cfg.ExtraConfig.RegionDefinitionsLimit, "Sets the maximum number of bpmn definitions in a region")

	// auth
	fs.UintVar(&cfg.EtcdConfig.AuthTokenTTL, "auth-token-ttl", cfg.EtcdConfig.AuthTokenTTL, "Time (in seconds) of the auth-token-ttl.")

	// logging
	fs.StringVar(&cfg.EtcdConfig.Logger, "logger", "zap",
		"Currently only supports 'zap' for structured logging.")
	fs.Var(flags.NewUniqueStringsValue(logutil.DefaultLogOutput), "log-outputs",
		"Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.")
	fs.StringVar(&cfg.EtcdConfig.LogLevel, "log-level", logutil.DefaultLogLevel,
		"Configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.")
	fs.BoolVar(&cfg.EtcdConfig.EnableLogRotation, "enable-log-rotation", false,
		"Enable log rotation of a single log-outputs file target.")
	fs.StringVar(&cfg.EtcdConfig.LogRotationConfigJSON, "log-rotation-config-json", logutil.DefaultLogRotationConfig,
		"Configures log rotation if enabled with a JSON logger config. Default: MaxSize=100(MB), MaxAge=0(days,no limit), MaxBackups=0(no limit), LocalTime=false(UTC), Compress=false(gzip)")

	return fs
}

// TestConfig get Config for testing
func TestConfig() (Config, func()) {
	cfg := NewConfig()

	cancel := func() {}

	return *cfg, cancel
}

func (cfg *Config) Validate() (err error) {
	if err = cfg.EtcdConfig.Validate(); err != nil {
		return
	}

	return
}

type ExtraConfig struct {
	APIResourceConfigSource serverstorage.APIResourceConfigSource
	StorageFactory          serverstorage.StorageFactory

	// The maximum number of regions in a runner
	RegionLimit int
	// The maximum number of bpmn definitions in a region
	RegionDefinitionsLimit int
}

type completedConfig struct {
	EtcdConfig    *embed.Config
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		EtcdConfig:    cfg.EtcdConfig,
		GenericConfig: cfg.GenericConfig.Complete(),
		ExtraConfig: &ExtraConfig{
			APIResourceConfigSource: DefaultAPIResourceConfigSource(),
		},
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of Server from the given config.
func (c completedConfig) New() (*MonitorServer, error) {
	lg := c.EtcdConfig.GetLogger()
	embedDaemon := genericdaemon.NewEmbedDaemon(lg)

	etcd, err := embed.StartEtcd(c.EtcdConfig)
	if err != nil {
		return nil, err
	}
	<-etcd.Server.ReadyNotify()

	genericServer, err := c.GenericConfig.New("olive-mon", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	monServer := &MonitorServer{
		IDaemon: embedDaemon,

		ctx:    ctx,
		cancel: cancel,

		genericAPIServer: genericServer,

		lg:       lg,
		v3cli:    v3client.New(etcd.Server),
		idGen:    idutil.NewGenerator(uint16(etcd.Server.ID()), time.Now()),
		notifier: leader.NewNotify(etcd.Server),
	}

	restStorageProviders := []RESTStorageProvider{
		&monrest.RESTStorageProvider{},
	}

	if err = monServer.InstallAPIs(c.ExtraConfig.APIResourceConfigSource, c.GenericConfig.RESTOptionsGetter,
		restStorageProviders...); err != nil {
		return nil, err
	}

	return monServer, nil
}
