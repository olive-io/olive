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
	"fmt"
	"time"

	"github.com/olive-io/olive/server/datadir"
	"go.etcd.io/etcd/client/pkg/v3/types"
)

const (
	DefaultLogOutput = "default"
	JournalLogOutput = "systemd/journal"
	StdErrLogOutput  = "stderr"
	StdOutLogOutput  = "stdout"

	DefaultLogPkgs = `{"transport": "warning","rsm": "warning", "raft": "error", "grpc": "warning"}`
	// DefaultLogRotationConfig is the default configuration used for log rotation.
	// Log rotation is disabled by default.
	// MaxSize    = 100 // MB
	// MaxAge     = 0 // days (no limit)
	// MaxBackups = 0 // no limit
	// LocalTime  = false // use computers local time, UTC by default
	// Compress   = false // compress the rotated log in gzip format
	DefaultLogRotationConfig = `{"maxsize": 100, "maxage": 0, "maxbackups": 0, "localtime": false, "compress": false}`

	DefaultCacheSize = 10 * 1024

	DefaultRTTMillisecond = 200

	DefaultSnapshotCount = 10

	DefaultBackendBatchInterval = time.Second * 3
	DefaultBatchLimit           = 100

	DefaultElectionTicks = 10

	DefaultTickMs = 1

	DefaultCompactionBatchLimit = 100

	DefaultMaxRequestBytes      = 10 * 1024 * 1024
	DefaultWarningApplyDuration = time.Second * 5
)

var (
	ErrLogRotationInvalidLogOutput = fmt.Errorf("--log-outputs requires a single file path when --log-rotate-config-json is defined")
)

type ServerConfig struct {
	Logger *LoggerConfig

	Name string

	CacheSize      int64
	DataDir        string
	WALDir         string
	RTTMillisecond uint64

	SnapshotCount uint64

	// BackendBatchInterval is the maximum time before commit the backend transaction.
	BackendBatchInterval time.Duration
	// BackendBatchLimit is the maximum operations before commit the backend transaction.
	BackendBatchLimit int

	TickMs        uint64
	ElectionTicks uint64

	// PreVote is true to enable Raft Pre-Vote.
	PreVote bool

	CompactionBatchLimit int

	// MaxRequestBytes is the maximum request size to send over raft.
	MaxRequestBytes uint

	WarningApplyDuration time.Duration

	// ExperimentalTxnModeWriteWithSharedBuffer enable write transaction to use
	// a shared buffer in its readonly check operations.
	ExperimentalTxnModeWriteWithSharedBuffer bool `json:"experimental-txn-mode-write-with-shared-buffer"`

	RaftAddress string
}

func NewServiceConfig(Name, dataDir, RaftAddress string) ServerConfig {
	lg := NewLoggerConfig()
	cfg := ServerConfig{
		Logger:                                   lg,
		Name:                                     Name,
		CacheSize:                                DefaultCacheSize,
		DataDir:                                  dataDir,
		RTTMillisecond:                           DefaultRTTMillisecond,
		SnapshotCount:                            DefaultSnapshotCount,
		BackendBatchInterval:                     DefaultBackendBatchInterval,
		BackendBatchLimit:                        DefaultBatchLimit,
		TickMs:                                   DefaultTickMs,
		ElectionTicks:                            DefaultElectionTicks,
		PreVote:                                  false,
		CompactionBatchLimit:                     DefaultCompactionBatchLimit,
		MaxRequestBytes:                          DefaultMaxRequestBytes,
		WarningApplyDuration:                     DefaultWarningApplyDuration,
		ExperimentalTxnModeWriteWithSharedBuffer: false,
		RaftAddress:                              RaftAddress,
	}

	return cfg
}

func (c *ServerConfig) BackendPath() string { return datadir.ToBackendFileName(c.DataDir) }

func (c *ServerConfig) WALPath() string {
	if c.WALDir != "" {
		return c.WALDir
	}
	return datadir.ToWalDir(c.DataDir)
}

// ReqTimeout returns timeout for request to finish.
func (c *ServerConfig) ReqTimeout() time.Duration {
	// 5s for queue waiting, computation and disk IO delay
	// + 2 * election timeout for possible leader election
	return 5*time.Second + 2*time.Duration(c.ElectionTicks*c.TickMs)*time.Millisecond
}

func (c *ServerConfig) Apply() error {
	if err := c.Logger.Apply(); err != nil {
		return err
	}

	return nil
}

type ShardConfig struct {
	Name string `json:"name"`

	PeerURLs types.URLsMap

	NewCluster bool
}
