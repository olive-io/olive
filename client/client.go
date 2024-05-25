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
	"errors"
	"fmt"
	"sync"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client provides and manages an olive-meta client session.
type Client struct {
	// embed etcd client
	clientv3.Cluster
	clientv3.KV
	clientv3.Watcher
	clientv3.Maintenance

	// the client of etcd server
	ec *clientv3.Client

	ctx    context.Context
	cancel context.CancelFunc

	mrw sync.RWMutex
	mds map[string]string

	cfg  *Config
	conn *grpc.ClientConn

	callOpts []grpc.CallOption
}

func New(cfg Config) (*Client, error) {
	if len(cfg.Endpoints) == 0 {
		return nil, ErrNoAvailableEndpoints
	}

	return newClient(&cfg)
}

func newClient(cfg *Config) (*Client, error) {
	if cfg.DialOptions == nil {
		cfg.DialOptions = []grpc.DialOption{}
	}

	if ipt := cfg.Interceptor; ipt != nil {
		cfg.DialOptions = append(cfg.DialOptions, grpc.WithUnaryInterceptor(ipt.Unary()))
	}
	etcdClient, err := clientv3.New(cfg.Config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(etcdClient.Ctx())
	client := &Client{
		ec:       etcdClient,
		conn:     etcdClient.ActiveConnection(),
		ctx:      ctx,
		cancel:   cancel,
		callOpts: defaultCallOpts,
	}

	if cfg.MaxCallSendMsgSize > 0 || cfg.MaxCallRecvMsgSize > 0 {
		if cfg.MaxCallRecvMsgSize > 0 && cfg.MaxCallSendMsgSize > cfg.MaxCallRecvMsgSize {
			return nil, fmt.Errorf("gRPC message recv limit (%d bytes) must be greater than send limit (%d bytes)", cfg.MaxCallRecvMsgSize, cfg.MaxCallSendMsgSize)
		}
		callOpts := []grpc.CallOption{
			defaultWaitForReady,
			defaultMaxCallSendMsgSize,
			defaultMaxCallRecvMsgSize,
		}
		if cfg.MaxCallSendMsgSize > 0 {
			callOpts[1] = grpc.MaxCallSendMsgSize(cfg.MaxCallSendMsgSize)
		}
		if cfg.MaxCallRecvMsgSize > 0 {
			callOpts[2] = grpc.MaxCallRecvMsgSize(cfg.MaxCallRecvMsgSize)
		}
		client.callOpts = callOpts
	}

	client.Cluster = etcdClient.Cluster
	client.KV = etcdClient.KV
	client.Watcher = etcdClient.Watcher
	client.Maintenance = etcdClient.Maintenance

	return client, nil
}

func (c *Client) ActiveEtcdClient() *clientv3.Client {
	return c.ec
}

func (c *Client) ActiveConnection() *grpc.ClientConn {
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

func toErr(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	err = rpctypes.Error(err)
	var etcdError rpctypes.EtcdError
	if errors.As(err, &etcdError) {
		return err
	}
	if ev, ok := status.FromError(err); ok {
		code := ev.Code()
		switch code {
		case codes.DeadlineExceeded:
			fallthrough
		case codes.Canceled:
			if ctx.Err() != nil {
				err = ctx.Err()
			}
		}
	}
	return err
}

func (c *Client) Close() error {
	return c.ec.Close()
}
