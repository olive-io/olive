// Copyright 2024 The olive Authors
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

package server

import (
	"context"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/proxy/codec"
)

// IServer is a simple vine server abstraction
type IServer interface {
	// Handle register a handler
	Handle(IHandler) error
	// NewHandler create a new handler
	NewHandler(any, ...HandlerOption) IHandler
	// Start the server with closable channel, IServer closing when the channel being closed
	Start(stopc <-chan struct{}) error
}

// Request is a synchronous request interface
type Request interface {
	// Service name requested
	Service() string
	// Method the action requested
	Method() string
	// Endpoint name requested
	Endpoint() string
	// ContentType Content Type provided
	ContentType() string
	// Header of the request
	Header() map[string]string
	// Body is the initial decoded value
	Body() any
	// Read the undecoded request body
	Read() ([]byte, error)
	// Codec the encoded message body
	Codec() codec.Reader
	// Stream indicates whether it's a stream
	Stream() bool
}

// Response is the response write for unencoded messages
type Response interface {
	// Codec encoded writer
	Codec() codec.Writer
	// WriteHeader write the header
	WriteHeader(map[string]string)
	// Write a response directly to the client
	Write([]byte) error
}

// IStream represents a stream established with a client.
// A stream can be bidirectional which is indicated by the request.
// The last error will be left in Error().
// EOF indicates end of the stream.
type IStream interface {
	Context() context.Context
	Request() Request
	Send(any) error
	Recv(any) error
	Error() error
	Close() error
}

// IHandler interface represents a request handler. It's generated
// by passing any type of public concrete object with endpoints into server.NewHandler.
// Most will pass in a struct.
//
// Example:
//
//	type Greeter struct{}
//
//	func (g *Greeter) Hello(context, request) (response, error) {
//		return nil
//	}
type IHandler interface {
	Name() string
	Handler() any
	Endpoints() []*dsypb.Endpoint
	Options() HandlerOptions
}
