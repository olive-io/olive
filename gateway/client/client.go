/*
   Copyright 2024 The olive Authors

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
	urlpkg "net/url"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	dsypb "github.com/olive-io/olive/api/pb/discovery"
	pb "github.com/olive-io/olive/api/pb/gateway"
)

var (
	DefaultMaxSize        = 1024 * 1024 * 10
	DefaultTimeout        = time.Second * 30
	DefaultInjectInterval = time.Second * 30
	DefaultInjectTTL      = time.Second * 45
)

type Client struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *Config

	cmu  sync.RWMutex
	conn *grpc.ClientConn

	rmu  sync.RWMutex
	yard *dsypb.Yard

	stopc chan struct{}
	done  chan struct{}
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInjectInterval
	}
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultInjectTTL
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewExample()
	}

	yard := &dsypb.Yard{
		Id:        cfg.Id,
		Address:   cfg.Address,
		Metadata:  make(map[string]string),
		Consumers: make(map[string]*dsypb.Consumer),
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		yard:   yard,
		stopc:  make(chan struct{}),
		done:   make(chan struct{}),
	}

	timeout := cfg.Timeout
	cctx, cancel1 := context.WithTimeout(ctx, timeout)
	defer cancel1()

	conn, err := c.newConn(cctx)
	if err != nil {
		return nil, err
	}
	c.conn = conn

	go c.process()

	return c, nil
}

func (c *Client) AddConsumer(cs *dsypb.Consumer) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	c.yard.Consumers[cs.Id] = cs
}

func (c *Client) process() {
	lg := c.Logger()
	if err := c.register(); err != nil {
		lg.Error("failed to register consumer", zap.Error(err))
	}

	t := new(time.Ticker)

	// only process if it exists
	if internal := c.cfg.Interval; internal > time.Duration(0) {
		// new ticker
		t = time.NewTicker(internal)
	}

Loop:
	for {
		select {
		case <-t.C:
			if err := c.register(); err != nil {
				lg.Error("failed to register", zap.Error(err))
			}
		// wait for exit
		case <-c.stopc:
			break Loop
		}
	}

	if err := c.deregister(); err != nil {
		lg.Error("failed to deregister", zap.Error(err))
	}

	if err := c.conn.Close(); err != nil {
		lg.Error("failed to close connection", zap.Error(err))
	}

	close(c.done)
}

func (c *Client) register() error {
	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	conn, err := c.getConn()
	if err != nil {
		return err
	}

	c.rmu.RLock()
	yard := c.yard
	c.rmu.RUnlock()

	ttl := int64(c.cfg.TTL.Seconds()) + 10
	rr := &pb.InjectRequest{
		Yard: yard,
		Ttl:  ttl,
	}

	remote := pb.NewEndpointRouterClient(conn)
	opts := c.getOptions()
	_, err = remote.Inject(ctx, rr, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) deregister() error {
	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()

	conn, err := c.getConn()
	if err != nil {
		return err
	}

	c.rmu.RLock()
	yard := c.yard
	c.rmu.RUnlock()
	rr := &pb.DigOutRequest{Yard: yard}

	remote := pb.NewEndpointRouterClient(conn)
	opts := c.getOptions()
	_, err = remote.DigOut(ctx, rr, opts...)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getConn() (*grpc.ClientConn, error) {
	c.cmu.RLock()
	conn := c.conn
	c.cmu.RUnlock()
	state := conn.GetState()
	if state == connectivity.Shutdown ||
		state == connectivity.TransientFailure {
		c.Logger().Debug("client shutdown, reconnect to gateway")

		var err error
		conn, err = c.newConn(c.ctx)
		if err != nil {
			return nil, err
		}

		c.cmu.Lock()
		c.conn = conn
		c.cmu.Unlock()
	}

	return conn, nil
}

func (c *Client) newConn(ctx context.Context) (*grpc.ClientConn, error) {
	target := c.cfg.Endpoint
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	url, err := urlpkg.Parse(target)
	if err != nil {
		return nil, err
	}
	host := url.Host

	kp := keepalive.ClientParameters{
		Time:                15,
		Timeout:             20,
		PermitWithoutStream: true,
	}
	opts = append(opts, grpc.WithKeepaliveParams(kp))

	conn, err := grpc.DialContext(ctx, host, opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Client) Close() error {
	close(c.stopc)
	c.cancel()

	select {
	case <-c.done:
	case <-time.After(time.Second * 5):
		return errors.New("timeout")
	}

	return nil
}

func (c *Client) Logger() *zap.Logger {
	return c.cfg.Logger
}

func (c *Client) getOptions() []grpc.CallOption {
	options := []grpc.CallOption{
		grpc.MaxCallRecvMsgSize(DefaultMaxSize),
		grpc.MaxCallSendMsgSize(DefaultMaxSize),
	}
	return options
}
