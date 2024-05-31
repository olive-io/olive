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
	"fmt"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
)

// Client provides and manages an olive-mon client session.
type Client struct {
	// embed etcd client
	clientv3.Cluster
	clientv3.KV
	clientv3.Watcher
	clientv3.Maintenance

	// the client of etcd server
	ec *clientv3.Client

	// the client of api server
	*clientset.Clientset
	*dynamic.DynamicClient

	ctx    context.Context
	cancel context.CancelFunc

	mrw sync.RWMutex
	mds map[string]string

	cfg  *Config
	conn *grpc.ClientConn

	callOpts []grpc.CallOption
}

func New(cfg *Config) (*Client, error) {
	ctx := cfg.etcdCfg.Context
	if len(cfg.etcdCfg.Endpoints) == 0 {
		return nil, ErrNoAvailableEndpoints
	}

	etcdCfg := cfg.etcdCfg
	if etcdCfg.DialOptions == nil {
		etcdCfg.DialOptions = []grpc.DialOption{}
	}

	if ipt := cfg.Interceptor; ipt != nil {
		etcdCfg.DialOptions = append(etcdCfg.DialOptions, grpc.WithUnaryInterceptor(ipt.Unary()))
	}
	etcdClient, err := clientv3.New(etcdCfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(etcdClient.Ctx())
	client := &Client{
		ec:       etcdClient,
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		conn:     etcdClient.ActiveConnection(),
		callOpts: defaultCallOpts,
	}

	if etcdCfg.MaxCallSendMsgSize > 0 || etcdCfg.MaxCallRecvMsgSize > 0 {
		if etcdCfg.MaxCallRecvMsgSize > 0 && etcdCfg.MaxCallSendMsgSize > etcdCfg.MaxCallRecvMsgSize {
			return nil, fmt.Errorf("gRPC message recv limit (%d bytes) must be greater than send limit (%d bytes)", etcdCfg.MaxCallRecvMsgSize, etcdCfg.MaxCallSendMsgSize)
		}
		callOpts := []grpc.CallOption{
			defaultWaitForReady,
			defaultMaxCallSendMsgSize,
			defaultMaxCallRecvMsgSize,
		}
		if etcdCfg.MaxCallSendMsgSize > 0 {
			callOpts[1] = grpc.MaxCallSendMsgSize(etcdCfg.MaxCallSendMsgSize)
		}
		if etcdCfg.MaxCallRecvMsgSize > 0 {
			callOpts[2] = grpc.MaxCallRecvMsgSize(etcdCfg.MaxCallRecvMsgSize)
		}
		client.callOpts = callOpts
	}

	client.Cluster = etcdClient.Cluster
	client.KV = etcdClient.KV
	client.Watcher = etcdClient.Watcher
	client.Maintenance = etcdClient.Maintenance

	configOverrides := &clientcmd.ConfigOverrides{}

	kubeConfig := clientcmd.NewDefaultClientConfig(*cfg.apiCfg, configOverrides)
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	//TODO: Support to dynamic client Config
	client.Clientset, err = clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	client.DynamicClient, err = dynamic.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (c *Client) ActiveEtcdClient() *clientv3.Client {
	return c.ec
}

func (c *Client) ActiveGRPCConn() *grpc.ClientConn {
	return c.conn
}

func (c *Client) leaderEndpoints(ctx context.Context) ([]string, error) {
	resp, err := c.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	endpoints := make([]string, 0)
	for _, member := range resp.Members {
		if member.ID == resp.Header.ClusterId {
			endpoints = member.ClientURLs
			break
		}
	}
	return endpoints, nil
}

//func toErr(ctx context.Context, err error) error {
//	if err == nil {
//		return nil
//	}
//	err = rpctypes.Error(err)
//	var etcdError rpctypes.EtcdError
//	if errors.As(err, &etcdError) {
//		return err
//	}
//	if ev, ok := status.FromError(err); ok {
//		code := ev.Code()
//		switch code {
//		case codes.DeadlineExceeded:
//			fallthrough
//		case codes.Canceled:
//			if ctx.Err() != nil {
//				err = ctx.Err()
//			}
//		}
//	}
//	return err
//}

func (c *Client) Close() error {
	return c.ec.Close()
}
