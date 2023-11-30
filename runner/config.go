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

package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/olive-io/olive/client"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	flagSet = pflag.NewFlagSet("runner", pflag.ExitOnError)

	DefaultEndpoints = []string{"http://127.0.0.1:4379"}
)

const (
	DefaultDataDir   = "default"
	DefaultCacheSize = 4 * 1024 * 1024

	DefaultBackendBatchInterval = time.Hour
	DefaultBackendBatchLimit    = 10000

	DefaultListenPeerURL   = "http://127.0.0.1:5380"
	DefaultListenClientURL = "http://127.0.0.1:5379"

	DefaultHeartbeatMs = 5000

	DefaultRaftRTTMillisecond = 500
)

func init() {
	flagSet.String("data-dir", DefaultDataDir, "Path to the data directory.")
	flagSet.StringArray("endpoints", DefaultEndpoints, "Set gRPC endpoints to connect the cluster of olive-meta")
	flagSet.String("listen-peer-url", DefaultListenPeerURL, "Set the URL to listen on for peer traffic.")
	flagSet.String("advertise-peer-url", DefaultListenPeerURL, "Set advertise URL to listen on for peer traffic.")
	flagSet.String("listen-client-url", DefaultListenClientURL, "Set the URL to listen on for client traffic.")
	flagSet.String("advertise-client-url", DefaultListenClientURL, "Set advertise URL to listen on for client traffic.")
}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(flagSet)
}

type Config struct {
	client.Config

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

func NewConfig() Config {

	clientCfg := client.Config{}
	clientCfg.Endpoints = DefaultEndpoints
	clientCfg.Logger = zap.NewExample()

	cfg := Config{
		Config: clientCfg,

		DataDir:   DefaultDataDir,
		CacheSize: DefaultCacheSize,

		BackendBatchInterval: DefaultBackendBatchInterval,
		BackendBatchLimit:    DefaultBackendBatchLimit,

		ListenPeerURL:      DefaultListenPeerURL,
		ListenClientURL:    DefaultListenClientURL,
		HeartbeatMs:        DefaultHeartbeatMs,
		RaftRTTMillisecond: DefaultRaftRTTMillisecond,
	}

	return cfg
}

func NewConfigFromFlagSet(flags *pflag.FlagSet) (cfg Config, err error) {
	cfg = NewConfig()
	if cfg.Logger == nil {
		cfg.Logger, err = zap.NewProduction()
		if err != nil {
			return
		}
	}
	if cfg.DataDir, err = flags.GetString("data-dir"); err != nil {
		return
	}
	if cfg.Endpoints, err = flags.GetStringArray("endpoints"); err != nil {
		return
	}
	if cfg.ListenPeerURL, err = flags.GetString("listen-peer-url"); err != nil {
		return
	}
	if cfg.AdvertisePeerURL, err = flags.GetString("advertise-peer-url"); err != nil {
		return
	}
	if cfg.ListenClientURL, err = flags.GetString("listen-client-url"); err != nil {
		return
	}
	if cfg.AdvertiseClientURL, err = flags.GetString("advertise-client-url"); err != nil {
		return
	}

	return cfg, nil
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
	if !stat.IsDir() {
		return fmt.Errorf("data-dir is not a directory")
	}

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
