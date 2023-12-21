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

package executor

import (
	"fmt"

	"github.com/olive-io/olive/client"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	flagSet = pflag.NewFlagSet("executor", pflag.ExitOnError)

	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultListenURL = "http://127.0.0.1:5380"
)

func init() {
	flagSet.StringArray("endpoints", DefaultEndpoints, "Set gRPC endpoints to connect the cluster of olive-meta")
	flagSet.String("listen-url", DefaultListenURL, "Set the URL to listen on for gRPC traffic.")
	flagSet.String("advertise-url", DefaultListenURL, "Set advertise URL to listen on for gRPC traffic.")
}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(flagSet)
}

type Config struct {
	client.Config

	ListenURL    string
	AdvertiseURL string
}

func NewConfig() Config {

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = zap.NewExample()

	cfg := Config{
		Config:    clientCfg,
		ListenURL: DefaultListenURL,
	}

	return cfg
}

func NewConfigFromFlagSet(flags *pflag.FlagSet) (cfg Config, err error) {
	cfg = NewConfig()
	if cfg.Logger == nil {
		cfg.Logger, err = zap.NewProduction()
		if err != nil {
			return
		}
	}
	if cfg.Endpoints, err = flags.GetStringArray("endpoints"); err != nil {
		return
	}
	if cfg.ListenURL, err = flags.GetString("listen-url"); err != nil {
		return
	}
	if cfg.AdvertiseURL, err = flags.GetString("advertise-url"); err != nil {
		return
	}

	return cfg, nil
}

func (cfg *Config) Validate() error {
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("no endpoints")
	}

	return nil
}
