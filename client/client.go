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

package client

import (
	"context"
	gerrs "errors"
	"fmt"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client provides and manages an olive-meta client session.
type Client struct {
	Cluster
	MetaRPC
	BpmnRPC

	*clientv3.Client

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
	etcd, err := clientv3.New(cfg.Config)
	if err != nil {
		return nil, err
	}

	client := &Client{
		Client:   etcd,
		conn:     etcd.ActiveConnection(),
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

	client.Cluster = NewCluster(client)
	client.MetaRPC = NewMetaRPC(client)
	client.BpmnRPC = NewBpmnRPC(client)

	return client, nil
}

func (c *Client) ActiveEtcdClient() *clientv3.Client {
	return c.Client
}

func (c *Client) leaderEndpoints(ctx context.Context) ([]string, error) {
	meta, err := c.GetMeta(ctx)
	if err != nil {
		return nil, err
	}
	endpoints := make([]string, 0)
	for _, member := range meta.Members {
		if member.Id == meta.Leader {
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
	if gerrs.As(err, &etcdError) {
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
