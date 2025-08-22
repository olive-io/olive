/*
Copyright 2025 The olive Authors

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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"sigs.k8s.io/yaml"

	"github.com/olive-io/olive/pkg/logutil"
)

const (
	DefaultRaftAddr   = "localhost:4540"
	DefaultDataDir    = "./data"
	DefaultListenAddr = "localhost:5540"
)

type ConfigTLS struct {
	CaFile   string `json:"ca_file" toml:"ca_file"`
	CertFile string `json:"cert_file" toml:"cert_file"`
	KeyFile  string `json:"key_file" toml:"key_file"`
}

type Config struct {
	once sync.Once

	ListenAddr string `json:"listen" toml:"listen"`

	TLS *ConfigTLS `json:"tls" toml:"tls"`

	DataDir string `json:"data_dir" toml:"data_dir"`

	Log *logutil.LogConfig `json:"log" toml:"log"`
}

func NewConfig() *Config {
	lc := logutil.NewLogConfig()
	cfg := &Config{
		ListenAddr: DefaultListenAddr,
		DataDir:    DefaultDataDir,
		Log:        &lc,
	}

	return cfg
}

func (cfg *Config) Init() error {
	var err error
	cfg.once.Do(func() {
		err = cfg.init()
	})
	return err
}

func (cfg *Config) init() error {
	if cfg.Log == nil {
		lc := logutil.NewLogConfig()
		cfg.Log = &lc
	}
	err := cfg.Log.SetupLogging()
	cfg.Log.SetupGlobalLoggers()
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	if cfg.DataDir == "" {
		home, _ := os.UserHomeDir()
		cfg.DataDir = filepath.Join(home, ".olive")
		_ = os.MkdirAll(cfg.DataDir, 0755)
	} else {
		_, err := os.Stat(cfg.DataDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("read data root directory: %w", err)
			}
			_ = os.MkdirAll(cfg.DataDir, 0755)
		}
		if strings.HasPrefix(cfg.DataDir, "~") || strings.HasPrefix(cfg.DataDir, "./") {
			abs, err := filepath.Abs(cfg.DataDir)
			if err != nil {
				return fmt.Errorf("get data directory abs path: %w", err)
			}
			cfg.DataDir = abs
		}
	}
	return nil
}

func FromPath(filename string) (*Config, error) {
	var cfg Config
	var err error
	ext := filepath.Ext(filename)
	switch ext {
	case ".toml":
		_, err = toml.DecodeFile(filename, &cfg)
	case ".yaml", ".yml":
		err = yaml.Unmarshal([]byte(filename), &cfg)
	case ".json":
		err = json.Unmarshal([]byte(filename), &cfg)
	default:
		return nil, fmt.Errorf("invalid config format: %s", ext)
	}
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves config text to specific file path
func (cfg *Config) Save(filename string) error {
	var err error
	var data []byte
	ext := filepath.Ext(filename)
	switch ext {
	case ".toml":
		buf := bytes.NewBufferString("")
		err = toml.NewEncoder(buf).Encode(cfg)
		if err == nil {
			data = buf.Bytes()
		}
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
	case ".json":
		data, err = json.Marshal(cfg)
	default:
		return fmt.Errorf("invalid config format: %s", ext)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0755)
}

func (cfg *Config) Logger() *zap.Logger {
	return cfg.Log.GetLogger()
}
