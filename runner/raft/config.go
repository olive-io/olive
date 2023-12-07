// Copyright 2023 The olive Authors
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

type RegionConfig struct {
	RaftRTTMillisecond   uint64
	ElectionRTT          uint64
	HeartbeatRTT         uint64
	StatHeartBeatMs      int64
	WarningApplyDuration time.Duration
	MaxRequestBytes      int64
}

func NewRegionConfig() RegionConfig {
	return RegionConfig{
		RaftRTTMillisecond:   DefaultRaftRTTMillisecond,
		StatHeartBeatMs:      DefaultRegionHeartbeatMs,
		WarningApplyDuration: DefaultRegionWarningApplyDuration,
		MaxRequestBytes:      DefaultRegionMaxRequestBytes,
	}
}

func (cfg RegionConfig) ElectionDuration() time.Duration {
	return time.Duration(cfg.RaftRTTMillisecond*cfg.ElectionRTT) * time.Millisecond
}

func (cfg RegionConfig) ReqTimeout() time.Duration {
	return 5*time.Second + 2*cfg.ElectionDuration()
}
