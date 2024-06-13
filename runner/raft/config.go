/*
Copyright 2023 The olive Authors

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

package raft

import (
	"time"

	"go.uber.org/zap"
)

const (
	DefaultDataDir            = "raft"
	DefaultRaftAddress        = "127.0.0.1:5380"
	DefaultRaftRTTMillisecond = 500

	DefaultRegionHeartbeatMs          = 2000
	DefaultRegionWarningApplyDuration = 100 * time.Millisecond
	DefaultRegionMaxRequestBytes      = 1.5 * 1024 * 1024
)

type Config struct {
	DataDir string

	RaftAddress        string
	RaftRTTMillisecond uint64
	HeartbeatMs        int64

	Logger *zap.Logger
}

func NewConfig() Config {
	cfg := Config{
		DataDir:            DefaultDataDir,
		RaftAddress:        DefaultRaftAddress,
		HeartbeatMs:        DefaultRegionHeartbeatMs,
		RaftRTTMillisecond: DefaultRaftRTTMillisecond,
		Logger:             zap.NewExample(),
	}

	return cfg
}

type ShardConfig struct {
	RaftRTTMillisecond   uint64
	ElectionRTT          uint64
	HeartbeatRTT         uint64
	StatHeartBeatMs      int64
	WarningApplyDuration time.Duration
	MaxRequestBytes      int64
}

func NewRegionConfig() ShardConfig {
	return ShardConfig{
		RaftRTTMillisecond:   DefaultRaftRTTMillisecond,
		StatHeartBeatMs:      DefaultRegionHeartbeatMs,
		WarningApplyDuration: DefaultRegionWarningApplyDuration,
		MaxRequestBytes:      DefaultRegionMaxRequestBytes,
	}
}

func (cfg ShardConfig) ElectionDuration() time.Duration {
	return time.Duration(cfg.RaftRTTMillisecond*cfg.ElectionRTT) * time.Millisecond
}

func (cfg ShardConfig) ReqTimeout() time.Duration {
	return 5*time.Second + 2*cfg.ElectionDuration()
}
