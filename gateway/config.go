/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
	DefaultId      = "gateway"
	DefaultDataDir = "gateway"
)

const (
	DefaultListenURL        = "http://127.0.0.1:5390"
	DefaultRegisterInterval = time.Second * 20
	DefaultRegisterTTL      = time.Second * 30
)

type Config struct {
	logutil.LogConfig `json:",inline" yaml:",inline"`
	grpcproxy.Config  `json:",inline" yaml:",inline"`

	fs *pflag.FlagSet

	Client client.Config

	DataDir string `json:"data-dir"`

	OpenAPI string `json:"openapi"`
	// EnableGRPCGateway enables grpc gateway.
	// The gateway translates a RESTful HTTP API into gRPC.
	EnableGRPCGateway bool `json:"enable-grpc-gateway"`
}

func NewConfig() *Config {

	logging := logutil.NewLogConfig()
	pcfg := grpcproxy.NewConfig()
	pcfg.Id = DefaultId
	pcfg.ListenURL = DefaultListenURL
	pcfg.RegisterInterval = DefaultRegisterInterval
	pcfg.RegisterTTL = DefaultRegisterTTL

	clientCfg := client.NewConfig(logging.GetLogger())

	cfg := &Config{
		LogConfig: logging,
		Config:    pcfg,
		Client:    *clientCfg,
		DataDir:   DefaultDataDir,
	}
	cfg.fs = cfg.newFlagSet()

	return cfg
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

func (cfg *Config) newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("gateway", pflag.ExitOnError)

	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints,
		"Set gRPC endpoints to connect the cluster of olive-mon")
	fs.StringVar(&cfg.Id, "id", cfg.Id, "Set Gateway Id.")
	fs.StringVar(&cfg.OpenAPI, "openapiv3", "", "Set Path of openapi v3 docs")
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "Path to the data directory.")
	fs.StringVar(&cfg.ListenURL, "listen-url", cfg.ListenURL, "Set the URL to listen on for gRPC traffic.")
	fs.StringVar(&cfg.AdvertiseURL, "advertise-url", cfg.AdvertiseURL, "Set advertise URL to listen on for gRPC traffic.")
	fs.DurationVar(&cfg.RegisterInterval, "register-interval", cfg.RegisterInterval, "Set Register interval.")
	fs.DurationVar(&cfg.RegisterTTL, "register-ttl", cfg.RegisterTTL, "Set Register ttl.")

	fs.BoolVar(&cfg.EnableGRPCGateway, "enable-grpc-gateway", cfg.EnableGRPCGateway, "EnableGRPCGateway enables grpc gateway. The gateway translates a RESTful HTTP API into gRPC.")

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
