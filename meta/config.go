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
	"fmt"
	"os"
	"time"

	"github.com/olive-io/olive/server/config"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/client/pkg/v3/types"
)

const (
	DefaultName                  = "default"
	DefaultListenerClientAddress = "localhost:7379"
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
	config.ServerConfig

	Name string

	InitialCluster      types.URLsMap
	InitialClusterState string

	ElectionTimeout time.Duration

	ListenerClientAddress string

	MaxGRPCReceiveMessageSize int64
	MaxGRPCSendMessageSize    int64
}

func ConfigFromFlagSet(flags *pflag.FlagSet) (cfg Config, err error) {
	var scfg config.ServerConfig
	scfg, err = config.ServerConfigFromFlagSet(flags)
	if err != nil {
		return
	}

	cfg.ServerConfig = scfg
	cfg.Name, err = flags.GetString("name")
	if err != nil {
		return
	}

	var peerString string
	peerString, err = flags.GetString("initial-cluster")
	if err != nil {
		return
	}
	cfg.InitialCluster, err = types.NewURLsMap(peerString)
	if err != nil {
		return
	}

	cfg.InitialClusterState, err = flags.GetString("initial-cluster-state")
	if err != nil {
		return
	}

	cfg.ListenerClientAddress, err = flags.GetString("listener-client-address")
	if err != nil {
		return
	}

	cfg.ElectionTimeout, err = flags.GetDuration("election-timeout")
	if err != nil {
		return
	}

	return
}

// TestConfig get Config for testing
func TestConfig() (Config, func()) {
	scfg := config.NewServerConfig(config.DefaultLogOutput, config.DefaultListenerPeerAddress)
	peer, _ := types.NewURLsMap("test=http://" + config.DefaultListenerPeerAddress)
	cfg := Config{
		ServerConfig:          scfg,
		InitialCluster:        peer,
		ElectionTimeout:       time.Second * 5,
		ListenerClientAddress: DefaultListenerClientAddress,
		InitialClusterState:   NewCluster,
	}

	cancel := func() { os.RemoveAll("default") }

	return cfg, cancel
}

func (cfg *Config) Apply() (err error) {
	if err = cfg.ServerConfig.Apply(); err != nil {
		return err
	}

	if cfg.Name == "" {
		return fmt.Errorf("missing the name of server")
	}

	if cfg.ListenerClientAddress == "" {
		return fmt.Errorf("missing the address to listen on for client traffic")
	}

	if cfg.InitialCluster.Len() == 0 {
		cfg.InitialCluster, _ = types.NewURLsMap(cfg.Name + "=" + "http://" + cfg.ListenerPeerAddress)
	}

	if cfg.MaxGRPCReceiveMessageSize == 0 {
		cfg.MaxGRPCReceiveMessageSize = int64(cfg.MaxRequestBytes)
	}
	if cfg.MaxGRPCSendMessageSize == 0 {
		cfg.MaxGRPCSendMessageSize = int64(cfg.MaxRequestBytes)
	}

	return
}
