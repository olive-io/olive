package runner

import (
	"time"

	"github.com/spf13/pflag"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	runnerFlagSet = pflag.NewFlagSet("runner", pflag.ExitOnError)
)

const (
	DefaultEndpoints    = "http://127.0.0.1:4379"
	DefaultPeerListen   = "127.0.0.1:5380"
	DefaultClientListen = "127.0.0.1:5379"

	DefaultHeartbeatInterval = time.Second * 5
)

func init() {

}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(runnerFlagSet)
}

type Config struct {
	*clientv3.Config

	PeerListen      string
	ClientListen    string
	AdvertiseListen string

	HeartbeatInterval time.Duration
}

func NewConfig() Config {
	cfg := Config{
		Config: &clientv3.Config{
			Endpoints: []string{DefaultEndpoints},
		},

		PeerListen:        DefaultPeerListen,
		ClientListen:      DefaultClientListen,
		AdvertiseListen:   DefaultClientListen,
		HeartbeatInterval: DefaultHeartbeatInterval,
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
