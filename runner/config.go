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
	"github.com/olive-io/olive/pkg/cliutil/flags"
	"github.com/olive-io/olive/pkg/logutil"
)

const (
	DefaultDataDir   = "default"
	DefaultCacheSize = 4 * 1024 * 1024

	DefaultBackendBatchInterval = time.Minute * 10
	DefaultBackendBatchLimit    = 10000

	DefaultListenPeerURL   = "http://127.0.0.1:5380"
	DefaultListenClientURL = "http://127.0.0.1:5379"

	DefaultHeartbeatMs = 5000

	DefaultRaftRTTMillisecond = 500
)

type Config struct {
	logutil.LogConfig

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

	logging := logutil.NewLogConfig()

	clientCfg := client.NewConfig(logging.GetLogger())
	cfg := Config{
		LogConfig: logging,

		Client: *clientCfg,

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

	// Runner
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "Path to the data directory.")
	fs.StringArrayVar(&cfg.Client.Endpoints, "endpoints", cfg.Client.Endpoints, "Set gRPC endpoints to connect the cluster of olive-meta")
	fs.StringVar(&cfg.ListenClientURL, "listen-client-url", cfg.ListenClientURL, "Set the URL to listen on for client traffic.")
	fs.StringVar(&cfg.AdvertiseClientURL, "advertise-client-url", cfg.AdvertiseClientURL, "Set advertise URL to listen on for client traffic.")

	// Backend
	fs.DurationVar(&cfg.BackendBatchInterval, "backend-batch-interval", cfg.BackendBatchInterval, "the maximum time before commit the backend transaction.")
	fs.IntVar(&cfg.BackendBatchLimit, "backend-batch-limit", cfg.BackendBatchLimit, "the maximum operations before commit the backend transaction.")

	// Region
	fs.StringVar(&cfg.ListenPeerURL, "listen-peer-url", cfg.ListenPeerURL, "Set the URL to listen on for peer traffic.")
	fs.StringVar(&cfg.AdvertisePeerURL, "advertise-peer-url", cfg.AdvertisePeerURL, "Set advertise URL to listen on for peer traffic.")

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
	if stat != nil && !stat.IsDir() {
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

	level := lg.Level()
	logger.SetLoggerFactory(func(pkgName string) logger.ILogger {
		options := []zap.Option{
			zap.WithCaller(true),
			zap.AddCallerSkip(2),
			zap.Fields(zap.String("pkg", pkgName)),
			zap.AddStacktrace(zap.FatalLevel),
		}
		sugarLog := cfg.GetLogger().Sugar().
			WithOptions(options...)
		return &raftLogger{level: raftLevel(level), log: sugarLog}
	})
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
		return logger.ERROR
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return logger.CRITICAL
	default:
		return logger.WARNING
	}
}
