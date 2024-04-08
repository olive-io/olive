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

package gateway

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/cliutil/flags"
	"github.com/olive-io/olive/pkg/logutil"
	grpcproxy "github.com/olive-io/olive/pkg/proxy/server/grpc"
)

var (
	DefaultId        = "gateway"
	DefaultDataDir   = "gateway"
	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultListenURL        = "http://127.0.0.1:5390"
	DefaultRegisterInterval = time.Second * 20
	DefaultRegisterTTL      = time.Second * 30
)

type Config struct {
	logutil.LogConfig
	grpcproxy.Config

	fs *pflag.FlagSet

	Client client.Config

	DataDir string `json:"data-dir"`

	// EnableGRPCGateway enables grpc gateway.
	// The gateway translates a RESTful HTTP API into gRPC.
	EnableGRPCGateway bool `json:"enable-grpc-gateway"`
}

func NewConfig() *Config {

	logging := logutil.NewLogConfig()
	pcfg := grpcproxy.NewConfig()
	pcfg.Id = DefaultId

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = logging.GetLogger()

	cfg := Config{
		LogConfig: logging,
		Config:    pcfg,
		Client:    clientCfg,
		DataDir:   DefaultDataDir,
	}
	cfg.fs = cfg.newFlagSet()

	return &cfg
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

func (cfg *Config) newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("gateway", pflag.ExitOnError)

	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints,
		"Set gRPC endpoints to connect the cluster of olive-meta")
	fs.StringVar(&cfg.Id, "id", cfg.Id, "Set Gateway Id.")
	fs.StringVar(&cfg.OpenAPI, "openapiv3", "", "Set Path of openapi v3 docs")
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "Path to the data directory.")
	fs.StringVar(&cfg.ListenURL, "listen-url", cfg.ListenURL, "Set the URL to listen on for gRPC traffic.")
	fs.StringVar(&cfg.AdvertiseURL, "advertise-url", cfg.AdvertiseURL, "Set advertise URL to listen on for gRPC traffic.")
	fs.DurationVar(&cfg.RegisterInterval, "register-interval", cfg.RegisterInterval, "Set Register interval.")
	fs.DurationVar(&cfg.RegisterTTL, "register-ttl", cfg.RegisterTTL, "Set Register ttl.")

	// logging
	fs.Var(flags.NewUniqueStringsValue(logutil.DefaultLogOutput), "log-outputs",
		"Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.")
	fs.StringVar(&cfg.LogLevel, "log-level", logutil.DefaultLogLevel,
		"Configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.")
	fs.BoolVar(&cfg.EnableLogRotation, "enable-log-rotation", false,
		"Enable log rotation of a single log-outputs file target.")
	fs.StringVar(&cfg.LogRotationConfigJSON, "log-rotation-config-json", logutil.DefaultLogRotationConfig,
		"Configures log rotation if enabled with a JSON logger config. Default: MaxSize=100(MB), MaxAge=0(days,no limit), MaxBackups=0(no limit), LocalTime=false(UTC), Compress=false(gzip)")

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
	if len(cfg.DataDir) == 0 {
		return fmt.Errorf("missing data-dir")
	}
	if cfg.RegisterInterval == 0 {
		cfg.RegisterInterval = DefaultRegisterInterval
	}
	if cfg.RegisterTTL == 0 {
		cfg.RegisterTTL = DefaultRegisterTTL
	}

	return nil
}

func (cfg *Config) setupLogging() error {
	if err := cfg.SetupLogging(); err != nil {
		return err
	}

	lg := cfg.GetLogger()
	cfg.Client.Logger = lg
	cfg.Config.SetLogger(lg)
	return nil
}
