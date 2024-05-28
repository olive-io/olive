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
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	krt "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	"sigs.k8s.io/yaml"

	"github.com/olive-io/olive/apis/mon/install"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/client/interceptor"
)

var (
	defaultEndpoints        = []string{"http://127.0.0.1:4379"}
	defaultTimeout          = time.Second * 30
	defaultKeepAliveTime    = time.Second * 20
	defaultKeepAliveTimeout = time.Second * 30
	defaultMaxMsgSize       = 50 * 1024 * 1024 // 10MB
)

func init() {
	install.Install(clientcmdlatest.Scheme)
}

type Config struct {
	etcdCfg clientv3.Config
	apiCfg  *clientcmdapi.Config

	Interceptor interceptor.Interceptor
}

func NewConfig(filename string, lg *zap.Logger) (*Config, error) {
	return NewConfigWithContext(context.Background(), filename, lg)
}

func NewConfigWithContext(ctx context.Context, filename string, lg *zap.Logger) (*Config, error) {
	if lg == nil {
		lg = zap.NewNop()
	}

	apiConfig, err := clientcmd.LoadFromFile(filename)
	if err != nil {
		return nil, err
	}

	cluster, ok := apiConfig.Clusters["mon"]
	if !ok {
		return nil, fmt.Errorf("mon cluster not found")
	}
	etcdExtension, ok := cluster.Extensions["etcd"]
	if !ok {
		return nil, fmt.Errorf("etcd cluster extension not found")
	}

	var ec monv1.EtcdCluster
	if value, ok := etcdExtension.(*krt.Unknown); ok {
		switch value.ContentType {
		case "application/json":
			err = json.Unmarshal(value.Raw, &ec)
		case "application/yaml":
			err = yaml.Unmarshal(value.Raw, &ec)
		}
		if err != nil {
			return nil, err
		}
	} else {
		data, err := yaml.Marshal(etcdExtension)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(data, &ec); err != nil {
			return nil, err
		}
	}

	if len(ec.Endpoints) == 0 {
		ec.Endpoints = defaultEndpoints
	}

	timeout, _ := time.ParseDuration(ec.Timeout)
	if timeout == 0 {
		timeout = defaultTimeout
	}

	etcdCfg := clientv3.Config{
		Endpoints:             ec.Endpoints,
		AutoSyncInterval:      time.Second * 120,
		DialTimeout:           timeout,
		DialKeepAliveTime:     defaultKeepAliveTime,
		DialKeepAliveTimeout:  defaultKeepAliveTimeout,
		MaxCallSendMsgSize:    defaultMaxMsgSize,
		MaxCallRecvMsgSize:    defaultMaxMsgSize,
		Context:               ctx,
		Logger:                lg,
		PermitWithoutStream:   true,
		MaxUnaryRetries:       uint(ec.MaxUnaryRetries),
		BackoffWaitBetween:    time.Millisecond * 500,
		BackoffJitterFraction: 0.5,
	}

	cfg := &Config{
		etcdCfg: etcdCfg,
		apiCfg:  apiConfig,
	}

	return cfg, nil
}
