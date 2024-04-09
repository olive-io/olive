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
