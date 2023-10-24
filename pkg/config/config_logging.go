// Copyright 2023 Lack (xingyys@gmail.com).
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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"

	dlg "github.com/lni/dragonboat/v4/logger"
	"go.etcd.io/etcd/client/pkg/v3/logutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerConfig struct {
	loggerMu *sync.RWMutex
	logger   *zap.Logger

	pkgLoggers map[string]*pkgLogger

	// ZapLoggerBuilder is used to build the zap logger.
	ZapLoggerBuilder func(*LoggerConfig) error

	LogLevel string `json:"log-level"`
	// Pkgs sets the Level for the packages of dragonboat
	Pkgs string `json:"log-pkgs"`
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
}

func NewLoggerConfig() *LoggerConfig {
	cfg := &LoggerConfig{
		loggerMu:              new(sync.RWMutex),
		logger:                nil,
		pkgLoggers:            map[string]*pkgLogger{},
		ZapLoggerBuilder:      nil,
		LogLevel:              logutil.DefaultLogLevel,
		Pkgs:                  DefaultLogPkgs,
		LogOutputs:            []string{DefaultLogOutput},
		EnableLogRotation:     false,
		LogRotationConfigJSON: DefaultLogRotationConfig,
	}

	return cfg
}

func (cfg *LoggerConfig) Apply() error {
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

	cfg.setupPkgLoggers()

	return nil
}

// GetLogger returns the logger.
func (cfg *LoggerConfig) GetLogger() *zap.Logger {
	cfg.loggerMu.RLock()
	l := cfg.logger
	cfg.loggerMu.RUnlock()
	return l
}

// NewZapLoggerBuilder generates a zap logger builder that sets given loger
// for embedded etcd.
func NewZapLoggerBuilder(lg *zap.Logger) func(config *LoggerConfig) error {
	return func(cfg *LoggerConfig) error {
		cfg.loggerMu.Lock()
		defer cfg.loggerMu.Unlock()
		cfg.logger = lg
		return nil
	}
}

// NewZapCoreLoggerBuilder - is a deprecated setter for the logger.
// Deprecated: Use simpler NewZapLoggerBuilder. To be removed in etcd-3.6.
func NewZapCoreLoggerBuilder(lg *zap.Logger, _ zapcore.Core, _ zapcore.WriteSyncer) func(config *LoggerConfig) error {
	return NewZapLoggerBuilder(lg)
}

// SetupGlobalLoggers configures 'global' loggers (grpc, zapGlobal) based on the cfg.
//
// The method is not executed by embed server by default (since 3.5) to
// enable setups where grpc/zap.Global logging is configured independently
// or spans separate lifecycle (like in tests).
func (cfg *LoggerConfig) SetupGlobalLoggers() {
	lg := cfg.GetLogger()
	if lg != nil {
		if cfg.LogLevel == "debug" {
			grpc.EnableTracing = true
			grpclog.SetLoggerV2(zapgrpc.NewLogger(lg))
		} else {
			grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, os.Stderr, os.Stderr))
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

func (cfg *LoggerConfig) setupPkgLoggers() {
	lgs := map[string]*pkgLogger{}
	pkgs := map[string]string{}
	_ = json.Unmarshal([]byte(cfg.Pkgs), &pkgs)
	for name, pkgLevel := range pkgs {
		level := cfg.convertToPkgLevel(pkgLevel)
		dlg.GetLogger(name).SetLevel(level)
		lgs[name] = &pkgLogger{lg: cfg.GetLogger(), level: level}
	}
	cfg.pkgLoggers = lgs

	dlg.SetLoggerFactory(func(pkgName string) dlg.ILogger {
		rl, ok := cfg.pkgLoggers[pkgName]
		if !ok {
			return &pkgLogger{lg: cfg.GetLogger(), level: cfg.convertToPkgLevel(cfg.LogLevel)}
		}
		return rl
	})
}

func (cfg *LoggerConfig) convertToPkgLevel(pkgLevel string) dlg.LogLevel {
	var level dlg.LogLevel
	switch strings.ToLower(pkgLevel) {
	case "debug":
		level = dlg.DEBUG
	case "info":
		level = dlg.INFO
	case "warning":
		level = dlg.WARNING
	case "error":
		level = dlg.ERROR
	case "panic":
		level = dlg.CRITICAL
	default:
		level = dlg.ERROR
	}

	return level
}

type pkgLogger struct {
	lg    *zap.Logger
	level dlg.LogLevel
}

func (lg *pkgLogger) SetLevel(level dlg.LogLevel) {
	lg.level = level
}

func (lg *pkgLogger) Debugf(format string, args ...interface{}) {
	if lg.level < dlg.DEBUG {
		return
	}
	lg.lg.Sugar().Debugf(format, args...)
}

func (lg *pkgLogger) Infof(format string, args ...interface{}) {
	if lg.level < dlg.INFO {
		return
	}
	lg.lg.Sugar().Infof(format, args...)
}

func (lg *pkgLogger) Warningf(format string, args ...interface{}) {
	if lg.level < dlg.WARNING {
		return
	}
	lg.lg.Sugar().Warnf(format, args...)
}

func (lg *pkgLogger) Errorf(format string, args ...interface{}) {
	if lg.level < dlg.ERROR {
		return
	}
	lg.lg.Sugar().Errorf(format, args...)
}

func (lg *pkgLogger) Panicf(format string, args ...interface{}) {
	if lg.level < dlg.CRITICAL {
		return
	}
	lg.lg.Sugar().Panicf(format, args...)
}
