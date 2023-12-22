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

package server

import (
	"fmt"
	"time"

	"github.com/olive-io/olive/client"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	flagSet = pflag.NewFlagSet("executor", pflag.ExitOnError)

	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultListenClientURL  = "http://127.0.0.1:5390"
	DefaultRegisterInterval = time.Second * 20
	DefaultRegisterTTL      = time.Second * 30
)

func init() {
	flagSet.StringArray("endpoints", DefaultEndpoints, "Set gRPC endpoints to connect the cluster of olive-meta")
	flagSet.String("id", "", "Set Executor Id.")
	flagSet.String("listen-client-url", DefaultListenClientURL, "Set the URL to listen on for gRPC traffic.")
	flagSet.String("advertise-client-url", DefaultListenClientURL, "Set advertise URL to listen on for gRPC traffic.")
	flagSet.Duration("register-interval", DefaultRegisterInterval, "Set Register interval.")
	flagSet.Duration("register-ttl", DefaultRegisterTTL, "Set Register ttl.")
}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(flagSet)
}

type Config struct {
	client.Config

	Id       string
	Metadata map[string]string

	ListenClientURL    string
	AdvertiseClientURL string
	// The interval on which to register
	RegisterInterval time.Duration
	// The register expiry time
	RegisterTTL time.Duration
}

func NewConfig() Config {

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = zap.NewExample()

	cfg := Config{
		Config:           clientCfg,
		ListenClientURL:  DefaultListenClientURL,
		RegisterInterval: DefaultRegisterInterval,
		RegisterTTL:      DefaultRegisterTTL,
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
	if cfg.Id, err = flags.GetString("id"); err != nil {
		return
	}
	if cfg.ListenClientURL, err = flags.GetString("listen-client-url"); err != nil {
		return
	}
	if cfg.AdvertiseClientURL, err = flags.GetString("advertise-client-url"); err != nil {
		return
	}
	if cfg.RegisterInterval, err = flags.GetDuration("register-interval"); err != nil {
		return
	}
	if cfg.RegisterTTL, err = flags.GetDuration("register-ttl"); err != nil {
		return
	}

	return cfg, nil
}

func (cfg *Config) Validate() error {
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("no endpoints")
	}
	if len(cfg.Id) == 0 {
		return fmt.Errorf("missing id")
	}

	return nil
}
