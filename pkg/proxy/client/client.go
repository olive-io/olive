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
