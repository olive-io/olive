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
)

var (
	ErrLogRotationInvalidLogOutput = fmt.Errorf("--log-outputs requires a single file path when --log-rotate-config-json is defined")
)

type ServerConfig struct {
	Logger *LoggerConfig

	// MaxRequestBytes is the maximum request size to send over raft.
	MaxRequestBytes uint

	TickMs        uint
	ElectionTicks int

	WarningApplyDuration time.Duration

	// ExperimentalTxnModeWriteWithSharedBuffer enable write transaction to use
	// a shared buffer in its readonly check operations.
	ExperimentalTxnModeWriteWithSharedBuffer bool `json:"experimental-txn-mode-write-with-shared-buffer"`
}

// ReqTimeout returns timeout for request to finish.
func (c *ServerConfig) ReqTimeout() time.Duration {
	// 5s for queue waiting, computation and disk IO delay
	// + 2 * election timeout for possible leader election
	return 5*time.Second + 2*time.Duration(c.ElectionTicks*int(c.TickMs))*time.Millisecond
}
