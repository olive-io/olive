package runner

import (
	"github.com/olive-io/olive/runner/backend"
	"go.uber.org/zap"
)

func newBackend(cfg *Config) backend.IBackend {
	bcfg := backend.BackendConfig{
		Dir:       cfg.DataDir,
		WAL:       cfg.WALDir(),
		CacheSize: cfg.CacheSize,
		Logger:    cfg.Logger,
	}

	if cfg.BackendBatchInterval != 0 {
		bcfg.BatchInterval = cfg.BackendBatchInterval
		if cfg.Logger != nil {
			cfg.Logger.Info("setting backend batch interval", zap.Duration("batch interval", cfg.BackendBatchInterval))
		}
	}
	if cfg.BackendBatchLimit != 0 {
		bcfg.BatchLimit = cfg.BackendBatchLimit
		if cfg.Logger != nil {
			cfg.Logger.Info("setting backend batch limit", zap.Int("batch limit", cfg.BackendBatchLimit))
		}
	}

	return backend.New(bcfg)
}
