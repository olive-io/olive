// Copyright 2023 Lack (xingyys@gmail.com).
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
	"net/url"

	"github.com/spf13/pflag"
	"go.etcd.io/etcd/server/v3/embed"
)

const (
	DefaultName                  = "default"
	DefaultListenerClientAddress = "http://localhost:4379"
	DefaultListenerPeerAddress   = "http://localhost:4380"
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
	metaFlagSet.String("listener-peer-address", DefaultListenerPeerAddress,
		"Sets the address to listen on for peer traffic.")
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
	peerURL, _ := url.Parse(DefaultListenerPeerAddress)
	ec.ListenPeerUrls = []url.URL{*peerURL}
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
