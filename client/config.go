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

package client

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/olive-io/olive/client/interceptor"
)

var (
	DefaultEndpoints        = []string{"http://127.0.0.1:4379"}
	DefaultTimeout          = time.Second * 30
	DefaultKeepAliveTime    = time.Second * 20
	DefaultKeepAliveTimeout = time.Second * 30
	DefaultMaxMsgSize       = 50 * 1024 * 1024 // 10MB
)

type Config struct {
	clientv3.Config `json:",inline"`

	Interceptor interceptor.Interceptor `json:"interceptor,omitempty"`
}

func NewConfig(lg *zap.Logger) *Config {
	return NewConfigWithContext(context.Background(), lg)
}

func NewConfigWithContext(ctx context.Context, lg *zap.Logger) *Config {
	if lg == nil {
		lg = zap.NewNop()
	}
	cc := clientv3.Config{
		Endpoints:             DefaultEndpoints,
		AutoSyncInterval:      0,
		DialTimeout:           DefaultTimeout,
		DialKeepAliveTime:     DefaultKeepAliveTime,
		DialKeepAliveTimeout:  DefaultKeepAliveTimeout,
		MaxCallSendMsgSize:    DefaultMaxMsgSize,
		MaxCallRecvMsgSize:    DefaultMaxMsgSize,
		Context:               ctx,
		Logger:                lg,
		PermitWithoutStream:   true,
		MaxUnaryRetries:       3,
		BackoffWaitBetween:    time.Millisecond * 500,
		BackoffJitterFraction: 0.5,
	}

	return &Config{Config: cc}
}
