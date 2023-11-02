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
	DefaultName = "default"
)

var (
	metaFlagSet = pflag.NewFlagSet("meta", pflag.ExitOnError)
)

func init() {
	metaFlagSet.String("name", DefaultName, "Human-readable name for this member.")
}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(metaFlagSet)
}

type Config struct {
	Server config.ServerConfig

	Name string

	PeerURLs types.URLsMap

	ShardTimeout time.Duration

	ListenerAddress string

	MaxGRPCReceiveMessageSize int64
	MaxGRPCSendMessageSize    int64
}

// TestConfig get Config for testing
func TestConfig() (Config, func()) {
	scfg := config.NewServerConfig(config.DefaultLogOutput, "localhost:7380")
	peer, _ := types.NewURLsMap("test=http://localhost:7380")
	cfg := Config{
		Server:          scfg,
		PeerURLs:        peer,
		ShardTimeout:    time.Second * 5,
		ListenerAddress: "localhost:7379",
	}

	cancel := func() { os.RemoveAll("default") }

	return cfg, cancel
}

func (cfg *Config) Apply() (err error) {
	if err = cfg.Server.Apply(); err != nil {
		return err
	}

	if cfg.Name == "" {
		return fmt.Errorf("missing the name of server")
	}

	if cfg.ListenerAddress == "" {
		return fmt.Errorf("missing the address of server")
	}

	if cfg.PeerURLs.Len() == 0 {
		cfg.PeerURLs, _ = types.NewURLsMap(cfg.Name + "=" + cfg.Server.ListenerPeerAddress)
	}

	return
}
