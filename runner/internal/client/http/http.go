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

package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/runner/internal/client"
	"github.com/olive-io/olive/runner/internal/client/selector"
	"github.com/olive-io/olive/runner/internal/codec"
	"github.com/olive-io/olive/runner/internal/context/metadata"
)

type httpClient struct {
	once sync.Once
	opts client.Options
}

func (h *httpClient) next(request client.Request, opts client.CallOptions) (selector.Next, error) {
	service := request.Service()

	// get proxy
	if prx := os.Getenv("OLIVE_PROXY"); len(prx) > 0 {
		service = prx
	}

	// get proxy address
	if prx := os.Getenv("OLIVE_PROXY_ADDRESS"); len(prx) > 0 {
		opts.Address = []string{prx}
	}

	// return remote address
	if len(opts.Address) > 0 {
		return func() (*pb.Node, error) {
			return &pb.Node{
				Address: opts.Address[0],
				Metadata: map[string]string{
					"protocol": "http",
				},
			}, nil
		}, nil
	}

	// only get the things that are of mucp protocol
	selectOptions := append(opts.SelectOptions, selector.WithFilter(
		selector.FilterLabel("protocol", "http"),
	))

	// get next nodes from the selector
	next, err := h.opts.Selector.Select(service, selectOptions...)
	if err != nil {
		return nil, err
	}

	return next, nil
}

func (h *httpClient) call(ctx context.Context, node *pb.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
	// set the address
	address := node.Address
	header := make(http.Header)
	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header.Set(k, v)
		}
	}

	// set timeout in nanoseconds
	header.Set("Timeout", fmt.Sprintf("%d", opts.RequestTimeout))
	// set the content type for the request
	header.Set("Content-Type", req.ContentType())

	// get codec
	cf, err := h.newHTTPCodec(req.ContentType())
	if err != nil {
		return err
	}

	// marshal request
	b, err := cf.Marshal(req.Body())
	if err != nil {
		return err
	}

	buf := &buffer{bytes.NewBuffer(b)}
	defer buf.Close()

	// start with / or not
	endpoint := req.Endpoint()
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	rawurl := "http://" + address + endpoint

	// parse rawurl
	URL, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	hreq := &http.Request{
		Method:        "POST",
		URL:           URL,
		Header:        header,
		Body:          buf,
		ContentLength: int64(len(b)),
		Host:          address,
	}

	// make the request
	hrsp, err := http.DefaultClient.Do(hreq.WithContext(ctx))
	if err != nil {
		return err
	}
	defer hrsp.Body.Close()

	// parse response
	b, err = io.ReadAll(hrsp.Body)
	if err != nil {
		return err
	}

	// unmarshal
	if err := cf.Unmarshal(b, rsp); err != nil {
		return err
	}

	return nil
}

func (h *httpClient) stream(ctx context.Context, node *pb.Node, req client.Request, opts client.CallOptions) (client.Stream, error) {
	// set the address
	address := node.Address
	header := make(http.Header)
	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header.Set(k, v)
		}
	}

	// set timeout in nanoseconds
	header.Set("Timeout", fmt.Sprintf("%d", opts.RequestTimeout))
	// set the content type for the request
	header.Set("Content-Type", req.ContentType())

	// get codec
	cf, err := h.newHTTPCodec(req.ContentType())
	if err != nil {
		return nil, err
	}

	cc, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	return &httpStream{
		address: address,
		context: ctx,
		closed:  make(chan bool),
		conn:    cc,
		codec:   cf,
		header:  header,
		reader:  bufio.NewReader(cc),
		request: req,
	}, nil
}

func (h *httpClient) newHTTPCodec(contentType string) (Codec, error) {
	if c, ok := defaultHTTPCodecs[contentType]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func (h *httpClient) newCodec(contentType string) (codec.NewCodec, error) {
	if c, ok := h.opts.Codecs[contentType]; ok {
		return c, nil
	}
	if cf, ok := defaultRPCCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}

func (h *httpClient) Init(opts ...client.Option) error {
	for _, o := range opts {
		o(&h.opts)
	}
	return nil
}

func (h *httpClient) Options() client.Options {
	return h.opts
}

func (h *httpClient) NewRequest(service, method string, req interface{}, reqOpts ...client.RequestOption) client.Request {
	return newHTTPRequest(service, method, req, h.opts.ContentType, reqOpts...)
}

func (h *httpClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	// make a copy of call opts
	callOpts := h.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes from the selector
	next, err := h.next(req, callOpts)
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
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// make copy of call method
	hcall := h.call

	// wrap the call in reverse
	for i := len(callOpts.CallWrappers); i > 0; i-- {
		hcall = callOpts.CallWrappers[i-1](hcall)
	}

	// return errors.New("go.vine.client", "request timeout", 408)
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
		if err != nil {
			return err
		}

		// make the call
		err = hcall(ctx, node, req, rsp, callOpts)
		h.opts.Selector.Mark(req.Service(), node, err)
		return err
	}

	ch := make(chan error, callOpts.Retries)
	var gerr error

	for i := 0; i < callOpts.Retries; i++ {
		go func() {
			ch <- call(i)
		}()

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

func (h *httpClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	// make a copy of call opts
	callOpts := h.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes from the selector
	next, err := h.next(req, callOpts)
	if err != nil {
		return nil, err
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
		opt := client.WithRequestTimeout(d.Sub(time.Now()))
		opt(&callOpts)
	}

	// should we noop right here?
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	call := func(i int) (client.Stream, error) {
		// call backoff first. Someone may want an initial start delay
		t, err := callOpts.Backoff(ctx, req, i)
		if err != nil {
			return nil, err
		}

		// only sleep if greater than 0
		if t.Seconds() > 0 {
			time.Sleep(t)
		}

		node, err := next()
		if err != nil {
			return nil, err
		}

		stream, err := h.stream(ctx, node, req, callOpts)
		h.opts.Selector.Mark(req.Service(), node, err)
		return stream, err
	}

	type response struct {
		stream client.Stream
		err    error
	}

	ch := make(chan response, callOpts.Retries)
	var grr error

	for i := 0; i < callOpts.Retries; i++ {
		go func() {
			s, err := call(i)
			ch <- response{s, err}
		}()

		select {
		case <-ctx.Done():
			grr = ctx.Err()
			goto Err
		case rsp := <-ch:
			// if the call succeeded lets bail early
			if rsp.err == nil {
				return rsp.stream, nil
			}

			retry, rerr := callOpts.Retry(ctx, req, i, err)
			if rerr != nil {
				grr = rerr
				goto Err
			}

			if !retry {
				grr = rsp.err
				goto Err
			}

			grr = rsp.err
		}
	}
Err:

	return nil, grr
}

func (h *httpClient) String() string {
	return "http"
}

func newClient(opts ...client.Option) (client.Client, error) {
	options := client.Options{
		CallOptions: client.CallOptions{
			Backoff:        client.DefaultBackoff,
			Retry:          client.DefaultRetry,
			Retries:        client.DefaultRetries,
			RequestTimeout: client.DefaultRequestTimeout,
			DialTimeout:    client.DefaultDialTimeout,
		},
	}

	for _, o := range opts {
		o(&options)
	}

	if len(options.ContentType) == 0 {
		options.ContentType = "application/proto"
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

	rc := &httpClient{
		once: sync.Once{},
		opts: options,
	}

	c := client.Client(rc)

	// wrap in reverse
	for i := len(options.Wrappers); i > 0; i-- {
		c = options.Wrappers[i-1](c)
	}

	return c, nil
}

func NewClient(opts ...client.Option) (client.Client, error) {
	return newClient(opts...)
}
