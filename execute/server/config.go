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

	"github.com/spf13/pflag"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/component-base/logs"
)

var (
	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultListenURL        = "http://127.0.0.1:5390"
	DefaultRegisterInterval = time.Second * 20
	DefaultRegisterTTL      = time.Second * 30
)

type Config struct {
	logs.LogConfig

	fs *pflag.FlagSet

	Client client.Config

	Id       string
	Metadata map[string]string

	ListenURL    string
	AdvertiseURL string
	// The interval on which to register
	RegisterInterval time.Duration
	// The register expiry time
	RegisterTTL time.Duration
}

func NewConfig() *Config {

	logging := logs.NewLogConfig()

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = logging.GetLogger()

	cfg := Config{
		LogConfig:        logging,
		Client:           clientCfg,
		ListenURL:        DefaultListenURL,
		RegisterInterval: DefaultRegisterInterval,
		RegisterTTL:      DefaultRegisterTTL,
	}
	cfg.fs = cfg.newFlagSet()

	return &cfg
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

func (cfg *Config) newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("executor", pflag.ExitOnError)
	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints,
		"Set gRPC endpoints to connect the cluster of olive-meta")
	fs.StringVar(&cfg.Id, "id", cfg.Id, "Set Executor Id.")
	fs.StringVar(&cfg.ListenURL, "listen-url", cfg.ListenURL, "Set the URL to listen on for gRPC traffic.")
	fs.StringVar(&cfg.AdvertiseURL, "advertise-url", cfg.AdvertiseURL, "Set advertise URL to listen on for gRPC traffic.")
	fs.DurationVar(&cfg.RegisterInterval, "register-interval", cfg.RegisterInterval, "Set Register interval.")
	fs.DurationVar(&cfg.RegisterTTL, "register-ttl", cfg.RegisterTTL, "Set Register ttl.")

	return fs
}

func (cfg *Config) Parse() error {
	if err := cfg.setupLogging(); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) Validate() error {
	if len(cfg.Client.Endpoints) == 0 {
		return fmt.Errorf("no endpoints")
	}
	if len(cfg.Id) == 0 {
		return fmt.Errorf("missing id")
	}

	return nil
}

func (cfg *Config) setupLogging() error {
	if err := cfg.SetupLogging(); err != nil {
		return err
	}

	lg := cfg.GetLogger()
	cfg.Client.Logger = lg
	return nil
}
