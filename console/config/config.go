/*
Copyright 2025 The olive Authors

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

package config

import (
	"errors"

	"github.com/spf13/pflag"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/cli/flags"
	"github.com/olive-io/olive/pkg/logutil"
)

var (
	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultListenURL = "http://127.0.0.1:8080"
)

type Config struct {
	logutil.LogConfig

	fs *pflag.FlagSet

	Client client.Config

	DSN string

	ListenURL string
}

func NewConfig() *Config {

	logging := logutil.NewLogConfig()

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = logging.GetLogger()

	cfg := Config{
		LogConfig: logging,

		Client: clientCfg,

		ListenURL: DefaultListenURL,
	}
	cfg.fs = cfg.newFlagSet()

	return &cfg
}

func (cfg *Config) Validate() error {
	if cfg.DSN == "" {
		return errors.New("dsn is required")
	}

	return nil
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

func (cfg *Config) newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("console", pflag.ExitOnError)

	// Console
	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints, "Set gRPC endpoints to connect the cluster of olive-mon")
	fs.StringVar(&cfg.ListenURL, "listen-url", cfg.ListenURL, "Set the URL to listen on for grpc traffic.")

	// Database
	fs.StringVar(&cfg.DSN, "db-dsn", cfg.DSN, "Set the database connection string.")

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

	return cfg.configFromCmdLine()
}

func (cfg *Config) setupLogging() error {
	if err := cfg.SetupLogging(); err != nil {
		return err
	}

	lg := cfg.GetLogger()
	cfg.Client.Logger = lg

	return nil
}

func (cfg *Config) configFromCmdLine() error {
	return nil
}
