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
	"io"
	"sync"

	"google.golang.org/grpc"

	"github.com/olive-io/olive/pkg/proxy/client"
)

// Implements the streamer interface
type grpcStream struct {
	sync.RWMutex
	closed   bool
	err      error
	conn     *grpc.ClientConn
	stream   grpc.ClientStream
	request  client.IRequest
	response client.IResponse
	ctx      context.Context
	cancel   func()
}

func (g *grpcStream) Context() context.Context {
	return g.ctx
}

func (g *grpcStream) Request() client.IRequest {
	return g.request
}

func (g *grpcStream) Response() client.IResponse {
	return g.response
}

func (g *grpcStream) Send(msg interface{}) error {
	if err := g.stream.SendMsg(msg); err != nil {
		g.setError(err)
		return err
	}
	return nil
}

func (g *grpcStream) Recv(msg interface{}) (err error) {
	defer g.setError(err)
	if err = g.stream.RecvMsg(msg); err != nil {
		// #202 - inconsistent gRPC stream behavior
		// the only way to tell if the stream is done is when we get an EOF on the Recv
		// here we should close the underlying gRPC ClientConn
		closeErr := g.Close()
		if err == io.EOF && closeErr != nil {
			err = closeErr
		}
	}
	return
}

func (g *grpcStream) Error() error {
	g.RLock()
	defer g.RUnlock()
	return g.err
}

func (g *grpcStream) setError(e error) {
	g.Lock()
	g.err = e
	g.Unlock()
}

// Close the gRPC send stream and gRPC connection
// #202 - inconsistent gRPC stream behavior
// The underlying gRPC stream should not be closed here since the
// stream should still be able to receive after this function call
func (g *grpcStream) Close() error {
	g.Lock()
	defer g.Unlock()

	if g.closed {
		_ = g.conn.Close()
		return nil
	}
	// cancel the context
	defer g.cancel()
	g.closed = true
	_ = g.stream.CloseSend()
	return g.conn.Close()
}

// CloseSend the gRPC send stream
func (g *grpcStream) CloseSend() error {
	g.Lock()
	defer g.Unlock()

	if g.closed {
		return nil
	}

	g.closed = true
	return g.stream.CloseSend()
}
