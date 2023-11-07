//go:build !windows
// +build !windows

package config

import (
	"fmt"
	"os"

	"go.etcd.io/etcd/client/pkg/v3/logutil"

	"go.uber.org/zap/zapcore"
)

// use stderr as fallback
func getJournalWriteSyncer() (zapcore.WriteSyncer, error) {
	jw, err := logutil.NewJournalWriter(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("can't find journal (%v)", err)
	}
	return zapcore.AddSync(jw), nil
}
