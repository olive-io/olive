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
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"github.com/olive-io/olive/client"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	runnerFlagSet = pflag.NewFlagSet("runner", pflag.ExitOnError)
)

const (
	DefaultDataDir   = "default"
	DefaultCacheSize = 4 * 1024 * 1024

	DefaultEndpoints    = "http://127.0.0.1:4379"
	DefaultPeerListen   = "127.0.0.1:5380"
	DefaultClientListen = "127.0.0.1:5379"

	DefaultHeartbeatMs = 5000

	DefaultRaftRTTMillisecond = 500
)

func init() {}

func AddFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(runnerFlagSet)
}

type Config struct {
	client.Config

	DataDir   string
	CacheSize uint64

	// BackendBatchInterval is the maximum time before commit the backend transaction.
	BackendBatchInterval time.Duration
	// BackendBatchLimit is the maximum operations before commit the backend transaction.
	BackendBatchLimit int

	PeerListen      string
	ClientListen    string
	AdvertiseListen string

	HeartbeatMs        int64
	RaftRTTMillisecond uint64
}

func NewConfig() Config {

	lg := zap.NewExample()

	clientCfg := client.Config{}
	clientCfg.Endpoints = []string{DefaultEndpoints}
	clientCfg.Logger = lg

	cfg := Config{
		Config: clientCfg,

		DataDir:   DefaultDataDir,
		CacheSize: DefaultCacheSize,

		PeerListen:         DefaultPeerListen,
		ClientListen:       DefaultClientListen,
		AdvertiseListen:    DefaultClientListen,
		HeartbeatMs:        DefaultHeartbeatMs,
		RaftRTTMillisecond: DefaultRaftRTTMillisecond,
	}

	return cfg
}

func NewConfigFromFlagSet(flags *pflag.FlagSet) (Config, error) {
	cfg := NewConfig()
	return cfg, nil
}

func (cfg *Config) Validate() error {
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
