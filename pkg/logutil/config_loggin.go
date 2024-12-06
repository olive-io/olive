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

package logutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sync"

	"go.etcd.io/etcd/client/pkg/v3/logutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	ErrLogRotationInvalidLogOutput = fmt.Errorf("--log-outputs requires a single file path when --log-rotate-config-json is defined")

	DefaultLogLevel = "info"

	DefaultLogOutput = "default"
	JournalLogOutput = "systemd/journal"
	StdErrLogOutput  = "stderr"
	StdOutLogOutput  = "stdout"

	// DefaultLogRotationConfig is the default configuration used for log rotation.
	// Log rotation is disabled by default.
	// MaxSize    = 100 // MB
	// MaxAge     = 0 // days (no limit)
	// MaxBackups = 0 // no limit
	// LocalTime  = false // use computers local time, UTC by default
	// Compress   = false // compress the rotated log in gzip format
	DefaultLogRotationConfig = `{"maxsize": 100, "maxage": 0, "maxbackups": 0, "localtime": false, "compress": false}`
)

// ConvertToZapLevel converts log level string to zapcore.Level.
func ConvertToZapLevel(lvl string) zapcore.Level {
	var level zapcore.Level
	if err := level.Set(lvl); err != nil {
		panic(err)
	}
	return level
}

type LogConfig struct {
	// LogLevel configures log level. Only supports debug, info, warn, error, panic, or fatal. Default 'info'.
	LogLevel string `json:"log-level"`
	// LogOutputs is either:
	//  - "default" as os.Stderr,
	//  - "stderr" as os.Stderr,
	//  - "stdout" as os.Stdout,
	//  - file path to append server logs to.
	// It can be multiple when "Logger" is zap.
	LogOutputs []string `json:"log-outputs"`
	// EnableLogRotation enables log rotation of a single LogOutputs file target.
	EnableLogRotation bool `json:"enable-log-rotation"`
	// LogRotationConfigJSON is a passthrough allowing a log rotation JSON config to be passed directly.
	LogRotationConfigJSON string `json:"log-rotation-config-json"`
	// ZapLoggerBuilder is used to build the zap logger.
	ZapLoggerBuilder func(*LogConfig) error

	// logger logs server-side operations. The default is nil,
	// and "setupLogging" must be called before starting server.
	// Do not set logger directly.
	loggerMu *sync.RWMutex
	logger   *zap.Logger
}

func NewLogConfig() LogConfig {
	return LogConfig{
		loggerMu:              new(sync.RWMutex),
		logger:                zap.NewNop(),
		LogOutputs:            []string{DefaultLogOutput},
		LogLevel:              DefaultLogLevel,
		EnableLogRotation:     false,
		LogRotationConfigJSON: DefaultLogRotationConfig,
	}
}

// GetLogger returns the logger.
func (cfg LogConfig) GetLogger() *zap.Logger {
	cfg.loggerMu.RLock()
	l := cfg.logger
	cfg.loggerMu.RUnlock()
	return l
}

// SetupLogging initializes logging.
// Must be called after flag parsing or finishing configuring LogConfig.
func (cfg *LogConfig) SetupLogging() error {
	if len(cfg.LogOutputs) == 0 {
		cfg.LogOutputs = []string{DefaultLogOutput}
	}
	if len(cfg.LogOutputs) > 1 {
		for _, v := range cfg.LogOutputs {
			if v == DefaultLogOutput {
				return fmt.Errorf("multi logoutput for %q is not supported yet", DefaultLogOutput)
			}
		}
	}
	if cfg.EnableLogRotation {
		if err := setupLogRotation(cfg.LogOutputs, cfg.LogRotationConfigJSON); err != nil {
			return err
		}
	}

	outputPaths, errOutputPaths := make([]string, 0), make([]string, 0)
	isJournal := false
	for _, v := range cfg.LogOutputs {
		switch v {
		case DefaultLogOutput:
			outputPaths = append(outputPaths, StdErrLogOutput)
			errOutputPaths = append(errOutputPaths, StdErrLogOutput)

		case JournalLogOutput:
			isJournal = true

		case StdErrLogOutput:
			outputPaths = append(outputPaths, StdErrLogOutput)
			errOutputPaths = append(errOutputPaths, StdErrLogOutput)

		case StdOutLogOutput:
			outputPaths = append(outputPaths, StdOutLogOutput)
			errOutputPaths = append(errOutputPaths, StdOutLogOutput)

		default:
			var path string
			if cfg.EnableLogRotation {
				// append rotate scheme to logs managed by lumberjack log rotation
				if v[0:1] == "/" {
					path = fmt.Sprintf("rotate:/%%2F%s", v[1:])
				} else {
					path = fmt.Sprintf("rotate:/%s", v)
				}
			} else {
				path = v
			}
			outputPaths = append(outputPaths, path)
			errOutputPaths = append(errOutputPaths, path)
		}
	}

	if !isJournal {
		copied := logutil.DefaultZapLoggerConfig
		copied.OutputPaths = outputPaths
		copied.ErrorOutputPaths = errOutputPaths
		copied = logutil.MergeOutputPaths(copied)
		copied.Level = zap.NewAtomicLevelAt(logutil.ConvertToZapLevel(cfg.LogLevel))
		if cfg.ZapLoggerBuilder == nil {
			lg, err := copied.Build()
			if err != nil {
				return err
			}
			cfg.ZapLoggerBuilder = NewZapLoggerBuilder(lg)
		}
	} else {
		if len(cfg.LogOutputs) > 1 {
			for _, v := range cfg.LogOutputs {
				if v != DefaultLogOutput {
					return fmt.Errorf("running with systemd/journal but other '--log-outputs' values (%q) are configured with 'default'; override 'default' value with something else", cfg.LogOutputs)
				}
			}
		}

		// use stderr as fallback
		syncer, lerr := getJournalWriteSyncer()
		if lerr != nil {
			return lerr
		}

		lvl := zap.NewAtomicLevelAt(logutil.ConvertToZapLevel(cfg.LogLevel))

		// WARN: do not change field names in encoder config
		// journald logging writer assumes field names of "level" and "caller"
		cr := zapcore.NewCore(
			zapcore.NewJSONEncoder(logutil.DefaultZapLoggerConfig.EncoderConfig),
			syncer,
			lvl,
		)
		if cfg.ZapLoggerBuilder == nil {
			cfg.ZapLoggerBuilder = NewZapLoggerBuilder(zap.New(cr, zap.AddCaller(), zap.ErrorOutput(syncer)))
		}
	}

	err := cfg.ZapLoggerBuilder(cfg)
	if err != nil {
		return err
	}

	return nil
}

// NewZapLoggerBuilder generates a zap logger builder that sets given loger
func NewZapLoggerBuilder(lg *zap.Logger) func(*LogConfig) error {
	return func(cfg *LogConfig) error {
		cfg.loggerMu.Lock()
		defer cfg.loggerMu.Unlock()
		cfg.logger = lg
		return nil
	}
}

// SetupGlobalLoggers configures 'global' loggers (grpc, zapGlobal) based on the cfg.
//
// The method is not executed by embed server by default (since 3.5) to
// enable setups where grpc/zap.Global logging is configured independently
// or spans separate lifecycle (like in tests).
func (cfg *LogConfig) SetupGlobalLoggers() {
	lg := cfg.GetLogger()
	if lg != nil {
		if cfg.LogLevel == "debug" {
			grpc.EnableTracing = true
			grpclog.SetLoggerV2(zapgrpc.NewLogger(lg))
		} else {
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(ioutil.Discard, os.Stderr, os.Stderr))
		}
		zap.ReplaceGlobals(lg)
	}
}

type logRotationConfig struct {
	*lumberjack.Logger
}

// Sync implements zap.Sink
func (logRotationConfig) Sync() error { return nil }

// setupLogRotation initializes log rotation for a single file path target.
func setupLogRotation(logOutputs []string, logRotateConfigJSON string) error {
	var logRotationConfig logRotationConfig
	outputFilePaths := 0
	for _, v := range logOutputs {
		switch v {
		case DefaultLogOutput, StdErrLogOutput, StdOutLogOutput:
			continue
		default:
			outputFilePaths++
		}
	}
	// log rotation requires file target
	if len(logOutputs) == 1 && outputFilePaths == 0 {
		return ErrLogRotationInvalidLogOutput
	}
	// support max 1 file target for log rotation
	if outputFilePaths > 1 {
		return ErrLogRotationInvalidLogOutput
	}

	if err := json.Unmarshal([]byte(logRotateConfigJSON), &logRotationConfig); err != nil {
		var unmarshalTypeError *json.UnmarshalTypeError
		var syntaxError *json.SyntaxError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("improperly formatted log rotation config: %w", err)
		case errors.As(err, &unmarshalTypeError):
			return fmt.Errorf("invalid log rotation config: %w", err)
		}
	}
	zap.RegisterSink("rotate", func(u *url.URL) (zap.Sink, error) {
		logRotationConfig.Filename = u.Path[1:]
		return &logRotationConfig, nil
	})
	return nil
}
