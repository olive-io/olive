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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/olive-io/olive/client-go"
	"github.com/olive-io/olive/pkg/logutil"
)

const (
	DefaultConfigPath = "oliveconfig"
	DefaultDataDir    = "default"
	DefaultCacheSize  = 4 * 1024 * 1024

	DefaultBackendBatchInterval = time.Minute * 10
	DefaultBackendBatchLimit    = 10000

	DefaultListenPeerURL   = "http://127.0.0.1:5380"
	DefaultListenClientURL = "http://127.0.0.1:5379"

	DefaultHeartbeatMs = 5000

	DefaultRaftRTTMillisecond = 500
)

type Config struct {
	logutil.LogConfig

	ConfigPath string

	clientConfig *client.Config

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

	cfg := Config{
		LogConfig:  logging,
		ConfigPath: DefaultConfigPath,
		DataDir:    DefaultDataDir,
		CacheSize:  DefaultCacheSize,

		BackendBatchInterval: DefaultBackendBatchInterval,
		BackendBatchLimit:    DefaultBackendBatchLimit,

		ListenPeerURL:      DefaultListenPeerURL,
		ListenClientURL:    DefaultListenClientURL,
		HeartbeatMs:        DefaultHeartbeatMs,
		RaftRTTMillisecond: DefaultRaftRTTMillisecond,
	}

	return &cfg
}

func (cfg *Config) Complete() error {
	var err error
	if err = cfg.setupLogging(); err != nil {
		return err
	}

	cfg.clientConfig, err = client.NewConfig(cfg.ConfigPath, cfg.GetLogger())
	if err != nil {
		return err
	}
	return nil
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
