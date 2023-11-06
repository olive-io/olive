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

package client

import (
	"log"
	"os"

	"go.etcd.io/etcd/client/pkg/v3/logutil"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapgrpc"
	"google.golang.org/grpc/grpclog"
)

func init() {
	// We override grpc logger only when the environment variable is set
	// in order to not interfere by default with user's code or other libraries.
	if os.Getenv("OLIVE_CLIENT_DEBUG") != "" {
		lg, err := logutil.CreateDefaultZapLogger(oliveClientDebugLevel())
		if err != nil {
			panic(err)
		}
		lg = lg.Named("olive-client")
		grpclog.SetLoggerV2(zapgrpc.NewLogger(lg))
	}
}

// SetLogger sets grpc logger.
//
// Deprecated: use grpclog.SetLoggerV2 directly or grpc_zap.ReplaceGrpcLoggerV2.
func SetLogger(l grpclog.LoggerV2) {
	grpclog.SetLoggerV2(l)
}

// oliveClientDebugLevel translates OLIVE_CLIENT_DEBUG into zap log level.
func oliveClientDebugLevel() zapcore.Level {
	envLevel := os.Getenv("OLIVE_CLIENT_DEBUG")
	if envLevel == "" || envLevel == "true" {
		return zapcore.InfoLevel
	}
	var l zapcore.Level
	if err := l.Set(envLevel); err != nil {
		log.Printf("Invalid value for environment variable 'OLIVE_CLIENT_DEBUG'. Using default level: 'info'")
		return zapcore.InfoLevel
	}
	return l
}