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
	"crypto/tls"
	"fmt"
	"time"

	"github.com/lni/goutils/netutil"
	"github.com/olive-io/olive/server/datadir"
	"github.com/olive-io/olive/server/execute"
	"github.com/olive-io/olive/server/hooks"
	"github.com/spf13/pflag"
	"go.etcd.io/etcd/client/pkg/v3/types"
)

const (
	DefaultLogOutput = "default"
	JournalLogOutput = "systemd/journal"
	StdErrLogOutput  = "stderr"
	StdOutLogOutput  = "stdout"

	DefaultLogPkgsConfig = `{"transport": "warning","rsm": "warning", "raft": "error", "grpc": "warning"}`
	// DefaultLogRotationConfig is the default configuration used for log rotation.
	// Log rotation is disabled by default.
	// MaxSize    = 100 // MB
	// MaxAge     = 0 // days (no limit)
	// MaxBackups = 0 // no limit
	// LocalTime  = false // use computers local time, UTC by default
	// Compress   = false // compress the rotated log in gzip format
	DefaultLogRotationConfig = `{"maxsize": 100, "maxage": 0, "maxbackups": 0, "localtime": false, "compress": false}`

	DefaultCacheSize = 10 * 1024 * 1024

	DefaultRTTMillisecond = 200

	DefaultBackendBatchInterval = 0
	DefaultBatchLimit           = 0

	DefaultHeartBeatTTL = 1

	DefaultElectionTTL = 10

	DefaultCompactionBatchLimit = 1000

	DefaultMaxRequestBytes      = 10 * 1024 * 1024
	DefaultWarningApplyDuration = time.Millisecond * 100

	DefaultListenerPeerAddress = "localhost:7380"
)

var (
	ErrLogRotationInvalidLogOutput = fmt.Errorf("--log-outputs requires a single file path when --log-rotate-config-json is defined")

	serverFlagSet = pflag.NewFlagSet("server", pflag.ExitOnError)
)

func init() {
	serverFlagSet.Uint64("cache-size", DefaultCacheSize, "Configures the size of cache of backend db")
	serverFlagSet.String("data-dir", "default", "Path to the data directory.")
	serverFlagSet.String("wal-dir", "", "Path to the dedicated wal directory.")
	serverFlagSet.Uint64("node-rtt-millisecond", DefaultRTTMillisecond,
		"the average Round Trip Time (RTT) in milliseconds between two NodeHost instances.")
	serverFlagSet.Duration("backend-batch-interval", DefaultBackendBatchInterval,
		"BackendBatchInterval is the maximum time before commit the backend transaction.")
	serverFlagSet.Int("backend-batch-limit", DefaultBatchLimit,
		"BackendBatchLimit is the maximum operations before commit the backend transaction.")
	serverFlagSet.Int("compaction-batch-limit", DefaultCompactionBatchLimit,
		"CompactionBatchLimit sets the maximum revisions deleted in each compaction batch.")
	serverFlagSet.Uint64("max-request-bytes", DefaultMaxRequestBytes,
		"Maximum client request size in bytes the server will accept.")
	serverFlagSet.Duration("warning-apply-duration", DefaultWarningApplyDuration,
		"Warning is generated if requests take more than this duration.")
	serverFlagSet.String("listener-peer-address", DefaultListenerPeerAddress,
		"Sets the address to listen on for peer traffic.")
	serverFlagSet.Uint64("heartbeat-ttl", DefaultHeartBeatTTL,
		"Sets the number of message RTT between heartbeats")
	serverFlagSet.Uint64("election-ttl", DefaultElectionTTL,
		"Sets the minimum number of message RTT between elections.")
	serverFlagSet.Bool("pre-vote", true,
		"PreVote is true to enable Raft Pre-Vote")
	serverFlagSet.Bool("mutual-tls", false,
		"MutualTLS defines whether to use mutual TLS for authenticating servers and clients.")
	serverFlagSet.String("cert-file", "",
		"Path to the client server TLS cert file.")
	serverFlagSet.String("key-file", "",
		"Path to the client server TLS key file.")
	serverFlagSet.String("ca-file", "",
		"Path to the client server TLS ca file.")
}

func AddServerFlagSet(flags *pflag.FlagSet) {
	flags.AddFlagSet(serverFlagSet)
}

type ServerConfig struct {
	CacheSize uint64 `json:"cache-size"`
	DataDir   string `json:"data-dir"`
	WALDir    string `json:"wal-dir"`

	RTTMillisecond uint64 `json:"node-rtt-millisecond"`

	// BackendBatchInterval is the maximum time before commit the backend transaction.
	BackendBatchInterval time.Duration `json:"backend-batch-interval"`
	// BackendBatchLimit is the maximum operations before commit the backend transaction.
	BackendBatchLimit int `json:"backend-batch-limit"`

	CompactionBatchLimit int `json:"compaction-batch-limit"`

	// MaxRequestBytes is the maximum request size to send over raft.
	MaxRequestBytes uint64 `json:"max-request-bytes"`

	WarningApplyDuration time.Duration `json:"warning-apply-duration"`

	// TxnModeWriteWithSharedBuffer enable write transaction to use
	// a shared buffer in its readonly check operations.
	TxnModeWriteWithSharedBuffer bool `json:"txn-mode-write-with-shared-buffer"`

	ListenerPeerAddress string `json:"listener-peer-address"`

	HeartBeatTTL uint64 `json:"heartbeat-ttl"`
	ElectionTTL  uint64 `json:"election-ttl"`

	// PreVote is true to enable Raft Pre-Vote.
	PreVote bool `json:"pre-vote"`

	// EnableLeaseCheckpoint enables leader to send regular checkpoints to other members to prevent reset of remaining TTL on leader change.
	EnableLeaseCheckpoint bool
	// LeaseCheckpointInterval time.Duration is the wait duration between lease checkpoints.
	LeaseCheckpointInterval time.Duration
	// LeaseCheckpointPersist enables persisting remainingTTL to prevent indefinite auto-renewal of long lived leases.
	LeaseCheckpointPersist bool

	// MutualTLS defines whether to use mutual TLS for authenticating servers
	// and clients. Insecure communication is used when MutualTLS is set to
	// False.
	MutualTLS bool `json:"mutual-tls"`

	// CAFile is the path of the CA certificate file. This field is ignored when
	// MutualTLS is false.
	CAFile string `json:"ca-file"`
	// CertFile is the path of the node certificate file. This field is ignored
	// when MutualTLS is false.
	CertFile string `json:"cert-file"`
	// KeyFile is the path of the node key file. This field is ignored when
	// MutualTLS is false.
	KeyFile string `json:"key-file"`

	ExecuteHooks []hooks.IExecuteHook

	Executor execute.IExecutor
}

func NewServerConfig(dataDir, listenerAddress string) ServerConfig {
	cfg := ServerConfig{
		CacheSize:                    DefaultCacheSize,
		DataDir:                      dataDir,
		RTTMillisecond:               DefaultRTTMillisecond,
		BackendBatchInterval:         DefaultBackendBatchInterval,
		BackendBatchLimit:            DefaultBatchLimit,
		CompactionBatchLimit:         DefaultCompactionBatchLimit,
		MaxRequestBytes:              DefaultMaxRequestBytes,
		WarningApplyDuration:         DefaultWarningApplyDuration,
		TxnModeWriteWithSharedBuffer: false,
		ListenerPeerAddress:          listenerAddress,
		HeartBeatTTL:                 DefaultHeartBeatTTL,
		ElectionTTL:                  DefaultElectionTTL,
		PreVote:                      false,
	}

	return cfg
}

func ServerConfigFromFlagSet(flags *pflag.FlagSet) (cfg ServerConfig, err error) {
	cfg = NewServerConfig("default", DefaultListenerPeerAddress)

	cfg.CacheSize, err = flags.GetUint64("cache-size")
	if err != nil {
		return
	}
	cfg.DataDir, err = flags.GetString("data-dir")
	if err != nil {
		return
	}
	cfg.WALDir, err = flags.GetString("wal-dir")
	if err != nil {
		return
	}
	cfg.RTTMillisecond, err = flags.GetUint64("node-rtt-millisecond")
	if err != nil {
		return
	}
	cfg.BackendBatchInterval, err = flags.GetDuration("backend-batch-interval")
	if err != nil {
		return
	}
	cfg.BackendBatchLimit, err = flags.GetInt("backend-batch-limit")
	if err != nil {
		return
	}
	cfg.CompactionBatchLimit, err = flags.GetInt("compaction-batch-limit")
	if err != nil {
		return
	}
	cfg.MaxRequestBytes, err = flags.GetUint64("max-request-bytes")
	if err != nil {
		return
	}
	cfg.WarningApplyDuration, err = flags.GetDuration("warning-apply-duration")
	if err != nil {
		return
	}
	cfg.ListenerPeerAddress, err = flags.GetString("listener-peer-address")
	if err != nil {
		return
	}
	cfg.HeartBeatTTL, err = flags.GetUint64("heartbeat-ttl")
	if err != nil {
		return
	}
	cfg.ElectionTTL, err = flags.GetUint64("election-ttl")
	if err != nil {
		return
	}
	cfg.PreVote, err = flags.GetBool("pre-vote")
	if err != nil {
		return
	}
	cfg.MutualTLS, err = flags.GetBool("mutual-tls")
	if err != nil {
		return
	}
	cfg.CertFile, err = flags.GetString("cert-file")
	if err != nil {
		return
	}
	cfg.KeyFile, err = flags.GetString("key-file")
	if err != nil {
		return
	}
	cfg.CAFile, err = flags.GetString("ca-file")
	if err != nil {
		return
	}

	return
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
	return 5*time.Second + 2*time.Duration(c.ElectionTTL*c.RTTMillisecond)*time.Millisecond
}

// TLSConfig returns the tls.Config
func (c *ServerConfig) TLSConfig() (*tls.Config, bool, error) {
	if !c.MutualTLS {
		return nil, false, nil
	}

	tc, err := netutil.GetServerTLSConfig(c.CAFile, c.CertFile, c.KeyFile)
	return tc, true, err
}

func (c *ServerConfig) Apply() error {
	if c.CacheSize == 0 {
		c.CacheSize = DefaultCacheSize
	}

	if c.CompactionBatchLimit == 0 {
		c.CompactionBatchLimit = DefaultCompactionBatchLimit
	}
	if c.MaxRequestBytes == 0 {
		c.MaxRequestBytes = DefaultMaxRequestBytes
	}
	if c.WarningApplyDuration == 0 {
		c.WarningApplyDuration = DefaultWarningApplyDuration
	}

	if c.RTTMillisecond == 0 {
		c.RTTMillisecond = DefaultRTTMillisecond
	}
	if c.HeartBeatTTL == 0 {
		c.HeartBeatTTL = DefaultHeartBeatTTL
	}
	if c.ElectionTTL == 0 {
		c.ElectionTTL = DefaultElectionTTL
	}

	return nil
}

type ShardConfig struct {
	Name string `json:"name"`

	ShardID uint64 `json:"shardID"`

	PeerURLs types.URLsMap

	NewCluster bool

	ElectionTimeout time.Duration
}

func (cfg *ShardConfig) Apply() error {
	if len(cfg.PeerURLs) != 0 {
		exists := false
		for name, _ := range cfg.PeerURLs {
			if name == cfg.Name {
				exists = true
				break
			}
		}

		if !exists {
			return fmt.Errorf("missing URL match with name")
		}
	}

	return nil
}
