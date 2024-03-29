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
)

var (
	// DefaultContentType is the default content type for client
	DefaultContentType = "application/protobuf"
	// DefaultBackoff is the default backoff function for retries
	DefaultBackoff = exponentialBackoff
	// DefaultRetry is the default check-for-retry function for retries
	DefaultRetry = RetryOnError
	// DefaultRetries is the default number of times a request is tried
	DefaultRetries = 1
	// DefaultDialTimeout is the default dial timeout
	DefaultDialTimeout = time.Second * 30
	// DefaultRequestTimeout is the default request timeout
	DefaultRequestTimeout = time.Second * 30
	// DefaultPoolSize sets the connection pool size
	DefaultPoolSize = 100
	// DefaultPoolTTL sets the connection pool ttl
	DefaultPoolTTL = time.Minute
)

// IClient is the interface used to make requests to services.
// It supports Request/Response via Transport and Publishing via the Broker.
// It also supports bidirectional streaming of requests.
type IClient interface {
	Options() Options
	NewRequest(service, endpoint string, req interface{}, reqOpts ...RequestOption) IRequest
	Call(ctx context.Context, req IRequest, rsp interface{}, opts ...CallOption) error
	Stream(ctx context.Context, req IRequest, opts ...CallOption) (IStream, error)
	String() string
}

// IRequest is the interface for a synchronous request used by Call or Stream
type IRequest interface {
	// Service the service to call
	Service() string
	// Method the action to take
	Method() string
	// Endpoint the endpoint to invoke
	Endpoint() string
	// ContentType the content type
	ContentType() string
	// Body the unencoded request body
	Body() interface{}
	// Stream indicates whether the request will be a streaming one rather than unary
	Stream() bool
}

// IResponse is the response received from a service
type IResponse interface {
	// Header reads the header
	Header() map[string]string
	// Read the undecoded response
	Read() ([]byte, error)
}

// IStream is the interface for a bidirectional synchronous stream
type IStream interface {
	// Context for the stream
	Context() context.Context
	// Request the request made
	Request() IRequest
	// Response the response read
	Response() IResponse
	// Send will encode and send a request
	Send(interface{}) error
	// Recv will decode and read a response
	Recv(interface{}) error
	// Error returns the stream error
	Error() error
	// Close closes the stream and close Conn
	Close() error
	// CloseSend closes the stream
	CloseSend() error
}
