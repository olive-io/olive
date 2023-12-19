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
	"time"

	"github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/runner/internal/client/selector"
	"github.com/olive-io/olive/runner/internal/codec"
)

type Options struct {
	// Used to select codec
	ContentType string

	// Plugged interfaces
	Codecs    map[string]codec.NewCodec
	Discovery discovery.IDiscovery
	Selector  selector.ISelector

	// Connection Pool
	PoolSize int
	PoolTTL  time.Duration

	// Middleware for client
	Wrappers []Wrapper

	// Default Call Options
	CallOptions CallOptions

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type CallOptions struct {
	SelectOptions []selector.SelectOption

	// Address of remote hosts
	Address []string
	// Backoff func
	Backoff BackoffFunc
	// Check if retryable func
	Retry RetryFunc
	// Transport Dial Timeout
	DialTimeout time.Duration
	// Number of Call attempts
	Retries int
	// Request/Response timeout
	RequestTimeout time.Duration
	// Stream timeout for the stream
	StreamTimeout time.Duration
	// Use the services own auth token
	ServiceToken bool
	// Duration to cache the response for
	CacheExpiry time.Duration

	// Middleware for low level call func
	CallWrappers []CallWrapper

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

type PublishOptions struct {
	// Exchange is the routing exchange for the message
	Exchange string
}

type MessageOptions struct {
	ContentType string
}

type RequestOptions struct {
	ContentType string
	Stream      bool

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

func NewOptions(options ...Option) Options {
	opts := Options{
		Context:     context.Background(),
		ContentType: DefaultContentType,
		Codecs:      make(map[string]codec.NewCodec),
		CallOptions: CallOptions{
			Backoff:        DefaultBackoff,
			Retry:          DefaultRetry,
			Retries:        DefaultRetries,
			DialTimeout:    DefaultDialTimeout,
			RequestTimeout: DefaultRequestTimeout,
			StreamTimeout:  DefaultRequestTimeout,
			Context:        context.Background(),
		},
		PoolSize: DefaultPoolSize,
		PoolTTL:  DefaultPoolTTL,
	}

	for _, o := range options {
		o(&opts)
	}

	return opts
}

// Option used by the Client
type Option func(*Options)

// CallOption used by Call or Stream
type CallOption func(*CallOptions)

// PublishOption used by Publish
type PublishOption func(*PublishOptions)

// MessageOption used by NewMessage
type MessageOption func(*MessageOptions)

// RequestOption used by NewRequest
type RequestOption func(*RequestOptions)

// Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c codec.NewCodec) Option {
	return func(o *Options) {
		o.Codecs[contentType] = c
	}
}

// ContentType default content type of the client
func ContentType(ct string) Option {
	return func(o *Options) {
		o.ContentType = ct
	}
}

// PoolSize sets the connection pool size
func PoolSize(d int) Option {
	return func(o *Options) {
		o.PoolSize = d
	}
}

// PoolTTL sets the connection pool ttl
func PoolTTL(d time.Duration) Option {
	return func(o *Options) {
		o.PoolTTL = d
	}
}

// Discovery to find nodes for a given service
func Discovery(dsy discovery.IDiscovery) Option {
	return func(o *Options) {
		o.Discovery = dsy
		// set in the selector
		if o.Selector != nil {
			_ = o.Selector.Init(selector.Discovery(dsy))
		}
	}
}

// Selector select is used to select a node to route a request to
func Selector(s selector.ISelector) Option {
	return func(o *Options) {
		o.Selector = s
	}
}

// Wrap adds a Wrapper to a list of options passed into the client
func Wrap(w Wrapper) Option {
	return func(o *Options) {
		o.Wrappers = append(o.Wrappers, w)
	}
}

// WrapCall adds a Wrapper to the list of CallFunc wrappers
func WrapCall(cw ...CallWrapper) Option {
	return func(o *Options) {
		o.CallOptions.CallWrappers = append(o.CallOptions.CallWrappers, cw...)
	}
}

// Backoff is used to set the backoff function used
// when retrying Calls
func Backoff(fn BackoffFunc) Option {
	return func(o *Options) {
		o.CallOptions.Backoff = fn
	}
}

// Retries number of retries when making the request.
// Should this be a Call Option?
func Retries(i int) Option {
	return func(o *Options) {
		o.CallOptions.Retries = i
	}
}

// Retry sets the retry function to be used when re-trying.
func Retry(fn RetryFunc) Option {
	return func(o *Options) {
		o.CallOptions.Retry = fn
	}
}

// RequestTimeout the request timeout.
// Should this be a Call Option?
func RequestTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.RequestTimeout = d
	}
}

// StreamTimeout sets the stream timeout
func StreamTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.StreamTimeout = d
	}
}

// DialTimeout transport dial timeout
func DialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.CallOptions.DialTimeout = d
	}
}

// Call Options

// WithExchange sets the exchange to route a message through
func WithExchange(e string) PublishOption {
	return func(o *PublishOptions) {
		o.Exchange = e
	}
}

// WithAddress sets the remote addresses to use rather than using service discovery
func WithAddress(a ...string) CallOption {
	return func(o *CallOptions) {
		o.Address = a
	}
}

func WithSelectOption(so ...selector.SelectOption) CallOption {
	return func(o *CallOptions) {
		o.SelectOptions = append(o.SelectOptions, so...)
	}
}

// WithCallWrapper is a CallOption which adds to the existing CallFunc wrappers
func WithCallWrapper(cw ...CallWrapper) CallOption {
	return func(o *CallOptions) {
		o.CallWrappers = append(o.CallWrappers, cw...)
	}
}

// WithBackoff is a CallOption which overrides that which
// set in Options.CallOptions
func WithBackoff(fn BackoffFunc) CallOption {
	return func(o *CallOptions) {
		o.Backoff = fn
	}
}

// WithRetry is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetry(fn RetryFunc) CallOption {
	return func(o *CallOptions) {
		o.Retry = fn
	}
}

// WithRetries is a CallOption which overrides that which
// set in Options.CallOptions
func WithRetries(i int) CallOption {
	return func(o *CallOptions) {
		o.Retries = i
	}
}

// WithRequestTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithRequestTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.RequestTimeout = d
	}
}

// WithStreamTimeout sets the stream timeout
func WithStreamTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.StreamTimeout = d
	}
}

// WithDialTimeout is a CallOption which overrides that which
// set in Options.CallOptions
func WithDialTimeout(d time.Duration) CallOption {
	return func(o *CallOptions) {
		o.DialTimeout = d
	}
}

// WithServiceToken is a CallOption which overrides the
// authorization header with the services own auth token
func WithServiceToken() CallOption {
	return func(o *CallOptions) {
		o.ServiceToken = true
	}
}

// WithCache is a CallOption which sets the duration the response
// should be cached for
func WithCache(c time.Duration) CallOption {
	return func(o *CallOptions) {
		o.CacheExpiry = c
	}
}

func WithMessageContentType(ct string) MessageOption {
	return func(o *MessageOptions) {
		o.ContentType = ct
	}
}

// Request Options

func WithContentType(ct string) RequestOption {
	return func(o *RequestOptions) {
		o.ContentType = ct
	}
}

func StreamingRequest() RequestOption {
	return func(o *RequestOptions) {
		o.Stream = true
	}
}
