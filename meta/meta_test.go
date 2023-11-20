package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOliveMetaServer(t *testing.T) {
	cfg, cancel := TestConfig()
	if !assert.NoError(t, cfg.Validate()) {
		return
	}
	defer cancel()

	s, err := NewServer(cfg)
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
