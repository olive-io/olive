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

package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/executor/client"
	"github.com/olive-io/olive/executor/client/selector"
	"github.com/olive-io/olive/pkg/discovery"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	gmetadata "google.golang.org/grpc/metadata"
)

var (
	// DefaultPoolMaxStreams maximum streams on a connections (20)
	DefaultPoolMaxStreams = 20

	// DefaultPoolMaxIdle maximum idle conns of a pool (50)
	DefaultPoolMaxIdle = 50

	// DefaultMaxRecvMsgSize maximum message that client can receive (100 MB)
	DefaultMaxRecvMsgSize = 1024 * 1024 * 100

	// DefaultMaxSendMsgSize maximum message that client can send (100 MB)
	DefaultMaxSendMsgSize = 1024 * 1024 * 100
)

type grpcClient struct {
	opts client.Options
	pool *pool
	once atomic.Value
}

// secure returns the dial option for whether it's a secure or insecure connection
func (g *grpcClient) secure(addr string, opts *client.CallOptions) grpc.DialOption {
	// first we check if there's tls config
	if tc := opts.TLSConfig; tc != nil {
		creds := credentials.NewTLS(tc)
		// return tls config if it exists
		return grpc.WithTransportCredentials(creds)
	}

	// default config
	tlsCfg := &tls.Config{}
	defaultCreds := grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))

	// check if the address is prepended with https
	if strings.HasPrefix(addr, "https://") {
		return defaultCreds
	}

	// if no port is specified or port is 443 default to tls
	_, port, err := net.SplitHostPort(addr)
	// assuming with no port it's going to be secured
	if port == "443" {
		return defaultCreds
	} else if err != nil && strings.Contains(err.Error(), "missing port in address") {
		return defaultCreds
	}

	// other fallback to insecure
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

func (g *grpcClient) next(request client.IRequest, opts client.CallOptions) (selector.Next, error) {
	service, address := request.Service(), opts.Address

	// return remote address
	if len(address) > 0 {
		fn := func() (*dsypb.Node, error) { return &dsypb.Node{Address: address[0]}, nil }
		return fn, nil
	}

	// get next nodes from the selector
	next, err := g.opts.Selector.Select(service, opts.SelectOptions...)
	if err != nil {
		if errors.Is(err, selector.ErrNotFound) {
			return nil, errors.Wrapf(err, "service %s", service)
		}
		return nil, errors.Wrapf(err, "selecting %s node", service)
	}

	return next, nil
}

func (g *grpcClient) call(ctx context.Context, node *dsypb.Node, req client.IRequest, rsp interface{}, opts client.CallOptions) error {
	address := node.Address

	header := make(map[string]string)
	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", opts.RequestTimeout)
	// set the content type for the request
	header["x-content-type"] = req.ContentType()

	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	maxRecvMsgSize := g.maxRecvMsgSizeValue()
	maxSendMsgSize := g.maxSendMsgSizeValue()

	var grr error
	gOpts := []grpc.DialOption{
		g.secure(address, &opts),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
			grpc.MaxCallSendMsgSize(maxSendMsgSize),
		),
	}

	if lopts := g.getGrpcDialOptions(); lopts != nil {
		gOpts = append(gOpts, lopts...)
	}

	cc, err := g.pool.getConn(address, gOpts...)
	if err != nil {
		return errors.Wrap(err, "sending request")
	}
	// defer execution of release
	defer g.pool.release(address, cc, grr)

	ch := make(chan error, 1)

	go func() {
		callOpts := []grpc.CallOption{}
		if lopts := g.getGrpcCallOptions(); lopts != nil {
			callOpts = append(callOpts, lopts...)
		}

		method := methodToGRPC(req.Service(), req.Endpoint())
		invokeErr := cc.Invoke(ctx, method, req.Body(), rsp, callOpts...)
		ch <- parseErr(invokeErr)
	}()

	select {
	case err = <-ch:
		grr = err
	case <-ctx.Done():
		grr = ctx.Err()
	}

	return grr
}

func (g *grpcClient) poolMaxStreams() int {
	return DefaultPoolMaxStreams
}

func (g *grpcClient) poolMaxIdle() int {
	return DefaultPoolMaxIdle
}

func (g *grpcClient) maxRecvMsgSizeValue() int {
	return DefaultMaxRecvMsgSize
}

func (g *grpcClient) maxSendMsgSizeValue() int {
	return DefaultMaxSendMsgSize
}

func (g *grpcClient) Init(opts ...client.Option) error {
	size := g.opts.PoolSize
	ttl := g.opts.PoolTTL

	for _, o := range opts {
		o(&g.opts)
	}

	// update pool configuration if the options changed
	if size != g.opts.PoolSize || ttl != g.opts.PoolTTL {
		g.pool.Lock()
		g.pool.size = g.opts.PoolSize
		g.pool.ttl = int64(g.opts.PoolTTL.Seconds())
		g.pool.Unlock()
	}

	return nil
}

func (g *grpcClient) Options() client.Options {
	return g.opts
}

func (g *grpcClient) NewRequest(service, method string, req interface{}, reqOpts ...client.RequestOption) client.IRequest {
	return newGRPCRequest(service, method, req, g.opts.ContentType, reqOpts...)
}

func (g *grpcClient) Call(ctx context.Context, req client.IRequest, rsp interface{}, opts ...client.CallOption) error {
	if req == nil {
		return errors.New("req is nil")
	} else if rsp == nil {
		return errors.New("rsp is nil")
	}

	// make a copy of call opts
	callOpts := g.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	next, err := g.next(req, callOpts)
	if err != nil {
		return err
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	if !ok {
		// no deadline so we create a new one
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, callOpts.RequestTimeout)
		defer cancel()
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := client.WithRequestTimeout(time.Until(d))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// make copy of call method
	gcall := g.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		gcall = callOpts.CallWrappers[i-1](gcall)
	}

	// return errors.Timeout("go.vine.client", "%v", ctx.Err())
	call := func(i int) error {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return err
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		// select next node
		node, err := next()
		service := req.Service()
		if err != nil {
			if errors.Is(err, selector.ErrNotFound) {
				return errors.Wrapf(err, "service %s", service)
			}
			return errors.Wrapf(err, "selecting %s node", service)
		}

		// make the call
		err = gcall(ctx, node, req, rsp, callOpts)
		g.opts.Selector.Mark(service, node, err)
		return err
	}

	ch := make(chan error, callOpts.Retries+1)
	var gerr error

	for i := 0; i <= callOpts.Retries; i++ {
		go func(i int) {
			ch <- call(i)
		}(i)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-ch:
			// if the call succeeded lets bail early
			if err == nil {
				return nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
			if rerr != nil {
				return rerr
			}

			if !retry {
				return err
			}

			gerr = err
		}
	}

	return gerr
}

func (g *grpcClient) String() string {
	return "grpc"
}

func (g *grpcClient) getGrpcDialOptions() []grpc.DialOption {
	if g.opts.CallOptions.Context == nil {
		return nil
	}

	v := g.opts.CallOptions.Context.Value(grpcDialOptions{})
	if v == nil {
		return nil
	}

	opts, ok := v.([]grpc.DialOption)
	if !ok {
		return nil
	}
	return opts
}

func (g *grpcClient) getGrpcCallOptions() []grpc.CallOption {
	if g.opts.CallOptions.Context == nil {
		return nil
	}

	v := g.opts.CallOptions.Context.Value(grpcCallOptions{})
	if v == nil {
		return nil
	}

	opts, ok := v.([]grpc.CallOption)
	if !ok {
		return nil
	}
	return opts
}

func newClient(opts ...client.Option) (client.IClient, error) {
	options := client.NewOptions()
	// default content type for grpc
	options.ContentType = "application/grpc+proto"

	for _, o := range opts {
		o(&options)
	}

	if options.Discovery == nil {
		return nil, discovery.ErrNoDiscovery
	}

	if options.Selector == nil {
		var err error
		options.Selector, err = selector.NewSelector(
			selector.Discovery(options.Discovery),
		)
		if err != nil {
			return nil, err
		}
	}

	rc := &grpcClient{
		opts: options,
	}
	rc.once.Store(false)

	rc.pool = newPool(options.PoolSize, options.PoolTTL, rc.poolMaxIdle(), rc.poolMaxStreams())

	c := client.IClient(rc)

	// wrap in reverse
	for i := len(options.Wrappers); i > 0; i-- {
		c = options.Wrappers[i-1](c)
	}

	return c, nil
}

func NewClient(opts ...client.Option) (client.IClient, error) {
	return newClient(opts...)
}
