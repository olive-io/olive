//go:build windows
// +build windows

package config

import (
	"os"

	"go.uber.org/zap/zapcore"
)

func getJournalWriteSyncer() (zapcore.WriteSyncer, error) {
	return zapcore.AddSync(os.Stderr), nil
}
