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

package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/lni/dragonboat/v4/logger"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/pkg/component-base/cli/flags"
	"github.com/olive-io/olive/pkg/component-base/logs"
)

var (
	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultDataDir   = "default"
	DefaultCacheSize = 4 * 1024 * 1024

	DefaultBackendBatchInterval = time.Hour
	DefaultBackendBatchLimit    = 10000

	DefaultListenPeerURL   = "http://127.0.0.1:5380"
	DefaultListenClientURL = "http://127.0.0.1:5379"

	DefaultHeartbeatMs = 5000

	DefaultRaftRTTMillisecond = 500
)

type Config struct {
	logs.LogConfig

	fs *pflag.FlagSet

	Client client.Config

	DataDir   string
	CacheSize uint64

	// BackendBatchInterval is the maximum time before commit the backend transaction.
	BackendBatchInterval time.Duration
	// BackendBatchLimit is the maximum operations before commit the backend transaction.
	BackendBatchLimit int

	ListenPeerURL      string
	AdvertisePeerURL   string
	ListenClientURL    string
	AdvertiseClientURL string

	HeartbeatMs        int64
	RaftRTTMillisecond uint64
}

func NewConfig() *Config {

	logging := logs.NewLogConfig()

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = logging.GetLogger()

	cfg := Config{
		LogConfig: logging,

		Client: clientCfg,

		DataDir:   DefaultDataDir,
		CacheSize: DefaultCacheSize,

		BackendBatchInterval: DefaultBackendBatchInterval,
		BackendBatchLimit:    DefaultBackendBatchLimit,

		ListenPeerURL:      DefaultListenPeerURL,
		ListenClientURL:    DefaultListenClientURL,
		HeartbeatMs:        DefaultHeartbeatMs,
		RaftRTTMillisecond: DefaultRaftRTTMillisecond,
	}
	cfg.fs = cfg.newFlagSet()

	return &cfg
}

func (cfg *Config) FlagSet() *pflag.FlagSet {
	return cfg.fs
}

func (cfg *Config) newFlagSet() *pflag.FlagSet {
	fs := pflag.NewFlagSet("runner", pflag.ExitOnError)

	// Node
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "Path to the data directory.")
	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints, "Set gRPC endpoints to connect the cluster of olive-meta")
	fs.StringVar(&cfg.ListenClientURL, "listen-client-url", cfg.ListenClientURL, "Set the URL to listen on for client traffic.")
	fs.StringVar(&cfg.AdvertiseClientURL, "advertise-client-url", cfg.AdvertiseClientURL, "Set advertise URL to listen on for client traffic.")

	// Region
	fs.StringVar(&cfg.ListenPeerURL, "listen-peer-url", cfg.ListenPeerURL, "Set the URL to listen on for peer traffic.")
	fs.StringVar(&cfg.AdvertisePeerURL, "advertise-peer-url", cfg.AdvertisePeerURL, "Set advertise URL to listen on for peer traffic.")

	// logging
	fs.Var(flags.NewUniqueStringsValue(logs.DefaultLogOutput), "log-outputs",
		"Specify 'stdout' or 'stderr' to skip journald logging even when running under systemd, or list of comma separated output targets.")
	fs.StringVar(&cfg.LogLevel, "log-level", logs.DefaultLogLevel,
		"Configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.")
	fs.BoolVar(&cfg.EnableLogRotation, "enable-log-rotation", false,
		"Enable log rotation of a single log-outputs file target.")
	fs.StringVar(&cfg.LogRotationConfigJSON, "log-rotation-config-json", logs.DefaultLogRotationConfig,
		"Configures log rotation if enabled with a JSON logger config. Default: MaxSize=100(MB), MaxAge=0(days,no limit), MaxBackups=0(no limit), LocalTime=false(UTC), Compress=false(gzip)")

	return fs
}

func (cfg *Config) Parse() error {
	if err := cfg.setupLogging(); err != nil {
		return err
	}

	return cfg.configFromCmdLine()
}

func (cfg *Config) Validate() error {
	stat, err := os.Stat(cfg.DataDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if err = os.MkdirAll(cfg.DataDir, os.ModePerm); err != nil {
			return err
		}
	}
	if !stat.IsDir() {
		return fmt.Errorf("data-dir is not a directory")
	}

	return nil
}

func (cfg *Config) setupLogging() error {
	if err := cfg.SetupLogging(); err != nil {
		return err
	}

	lg := cfg.GetLogger()
	cfg.Client.Logger = lg

	logger.SetLoggerFactory(func(pkgName string) logger.ILogger {
		options := []zap.Option{
			zap.WithCaller(true),
			zap.AddCallerSkip(2),
			zap.Fields(zap.String("pkg", pkgName)),
		}
		sugarLog := cfg.GetLogger().Sugar().
			WithOptions(options...)
		return &raftLogger{level: logger.INFO, log: sugarLog}
	})
	level := lg.Level()
	logger.GetLogger("raft").SetLevel(raftLevel(level))
	logger.GetLogger("rsm").SetLevel(raftLevel(level))
	logger.GetLogger("transport").SetLevel(raftLevel(level))
	logger.GetLogger("grpc").SetLevel(raftLevel(level))

	return nil
}

func (cfg *Config) LockDataDir() (*flock.Flock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	lock := flock.New(filepath.Join(cfg.DataDir, "olive-runner.lock"))
	ok, err := lock.TryLockContext(ctx, time.Millisecond*100)
	if err != nil || !ok {
		return nil, errors.New("data directory be used")
	}

	return lock, nil
}

func (cfg *Config) DBDir() string {
	return filepath.Join(cfg.DataDir, "db")
}

func (cfg *Config) WALDir() string {
	return filepath.Join(cfg.DataDir, "wal")
}

func (cfg *Config) RegionDir() string {
	return filepath.Join(cfg.DataDir, "regions")
}

func (cfg *Config) HeartbeatInterval() time.Duration {
	return time.Duration(cfg.HeartbeatMs) * time.Millisecond
}

func (cfg *Config) configFromCmdLine() error {
	return nil
}

type raftLogger struct {
	level logger.LogLevel
	log   *zap.SugaredLogger
}

func (lg *raftLogger) SetLevel(level logger.LogLevel) {
	lg.level = level
}

func (lg *raftLogger) Debugf(format string, args ...interface{}) {
	if lg.level < logger.DEBUG {
		return
	}
	lg.log.Debugf(format, args...)
}

func (lg *raftLogger) Infof(format string, args ...interface{}) {
	if lg.level < logger.INFO {
		return
	}
	lg.log.Infof(format, args...)
}

func (lg *raftLogger) Warningf(format string, args ...interface{}) {
	if lg.level < logger.WARNING {
		return
	}
	lg.log.Warnf(format, args...)
}

func (lg *raftLogger) Errorf(format string, args ...interface{}) {
	if lg.level < logger.ERROR {
		return
	}
	lg.log.Errorf(format, args...)
}

func (lg *raftLogger) Panicf(format string, args ...interface{}) {
	lg.log.Panicf(format, args...)
}

func raftLevel(level zapcore.Level) logger.LogLevel {
	switch level {
	case zapcore.DebugLevel:
		return logger.DEBUG
	case zapcore.InfoLevel:
		return logger.INFO
	case zapcore.WarnLevel:
		return logger.WARNING
	case zapcore.ErrorLevel:
		return logger.WARNING
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return logger.CRITICAL
	default:
		return logger.WARNING
	}
}
