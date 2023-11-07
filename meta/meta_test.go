package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewOliveMetaServer(t *testing.T) {
	cfg, cancel := TestConfig()
	if !assert.NoError(t, cfg.Apply()) {
		return
	}
	defer cancel()

	s, err := NewServer(zap.NewExample(), cfg)
	if !assert.NoError(t, err) {
		return
	}

	err = s.Start()
	if !assert.NoError(t, err) {
		return
	}

	err = s.GracefulStop()
	if !assert.NoError(t, err) {
		return
	}
}
