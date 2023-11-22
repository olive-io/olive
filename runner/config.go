package runner

import (
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

var (
	runnerFlagSet = pflag.NewFlagSet("runner", pflag.ExitOnError)
)

const (
	DefaultDataDir   = "default"
	DefaultCacheSize = 4 * 1024 * 1024

	DefaultEndpoints    = "http://127.0.0.1:4379"
	DefaultPeerListen   = "127.0.0.1:5380"
	DefaultClientListen = "127.0.0.1:5379"

	DefaultHeartbeatMs = 5000
)

func init() {

}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(runnerFlagSet)
}

type Config struct {
	*clientv3.Config

	DataDir   string
	CacheSize uint64

	// BackendBatchInterval is the maximum time before commit the backend transaction.
	BackendBatchInterval time.Duration
	// BackendBatchLimit is the maximum operations before commit the backend transaction.
	BackendBatchLimit int

	PeerListen      string
	ClientListen    string
	AdvertiseListen string

	HeartbeatMs int64
}

func NewConfig() Config {

	lg := zap.NewExample()
	cfg := Config{
		Config: &clientv3.Config{
			Endpoints: []string{DefaultEndpoints},
			Logger:    lg,
		},

		DataDir:   DefaultDataDir,
		CacheSize: DefaultCacheSize,

		PeerListen:      DefaultPeerListen,
		ClientListen:    DefaultClientListen,
		AdvertiseListen: DefaultClientListen,
		HeartbeatMs:     DefaultHeartbeatMs,
	}

	return cfg
}

func NewConfigFromFlagSet(flags *pflag.FlagSet) (Config, error) {
	cfg := NewConfig()
	return cfg, nil
}

func (cfg *Config) Validate() error {
	return nil
}

func (cfg *Config) DBDir() string {
	return filepath.Join(cfg.DataDir, "db")
}

func (cfg *Config) WALDir() string {
	return filepath.Join(cfg.DataDir, "wal")
}

func (cfg *Config) RegionRoot() string {
	return filepath.Join(cfg.DataDir, "regions")
}

func (cfg *Config) HeartbeatInterval() time.Duration {
	return time.Duration(cfg.HeartbeatMs) * time.Millisecond
}
