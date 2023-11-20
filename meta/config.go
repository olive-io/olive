package meta

import (
	"net/url"

	"github.com/spf13/pflag"
	"go.etcd.io/etcd/server/v3/embed"
)

const (
	DefaultName                  = "default"
	DefaultListenerClientAddress = "http://localhost:4379"
)

var (
	metaFlagSet = pflag.NewFlagSet("meta", pflag.ExitOnError)
)

func init() {
	metaFlagSet.String("name", DefaultName, "Human-readable name for this member.")
	metaFlagSet.String("initial-cluster", "",
		"Initial cluster configuration for bootstrapping.")
	metaFlagSet.String("initial-cluster-state", NewCluster,
		"Initial cluster state ('new' or 'existing').")
	metaFlagSet.String("listener-client-address", DefaultListenerClientAddress,
		"Sets the address to listen on for client traffic.")
	metaFlagSet.Duration("election-timeout", 0,
		"Sets the timeout to waiting for electing")
}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(metaFlagSet)
}

const (
	NewCluster      string = "new"
	ExistingCluster string = "existing"
)

type Config struct {
	*embed.Config
}

func NewConfig() Config {
	ec := embed.NewConfig()
	ec.Dir = DefaultName
	clientURL, _ := url.Parse(DefaultListenerClientAddress)
	ec.ListenClientUrls = []url.URL{*clientURL}
	ec.AdvertiseClientUrls = ec.ListenClientUrls
	cfg := Config{Config: ec}

	return cfg
}

func ConfigFromFlagSet(flags *pflag.FlagSet) (cfg Config, err error) {
	cfg = NewConfig()

	return
}

// TestConfig get Config for testing
func TestConfig() (Config, func()) {
	cfg := NewConfig()

	cancel := func() {}

	return cfg, cancel
}

func (cfg *Config) Validate() (err error) {
	if err = cfg.Config.Validate(); err != nil {
		return
	}

	return
}
