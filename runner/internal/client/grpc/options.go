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

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"github.com/olive-io/olive/runner/internal/client"
)

type codecsKey struct{}
type grpcDialOptions struct{}
type grpcCallOptions struct{}

// Codec gRPC Codec to be used to encode/decode requests for a given content type
func Codec(contentType string, c encoding.Codec) client.Option {
	return func(o *client.Options) {
		codecs := make(map[string]encoding.Codec)
		if o.Context == nil {
			o.Context = context.Background()
		}
		if v := o.Context.Value(codecsKey{}); v != nil {
			codecs = v.(map[string]encoding.Codec)
		}
		codecs[contentType] = c
		o.Context = context.WithValue(o.Context, codecsKey{}, codecs)
	}
}

// DialOptions to be used to configure gRPC dial options
func DialOptions(opts ...grpc.DialOption) client.Option {
	return func(o *client.Options) {
		if o.CallOptions.Context == nil {
			o.CallOptions.Context = context.Background()
		}
		o.CallOptions.Context = context.WithValue(o.Context, grpcDialOptions{}, opts)
	}
}

// CallOptions to be used to configure gRPC call options
func CallOptions(opts ...grpc.CallOption) client.CallOption {
	return func(o *client.CallOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, grpcCallOptions{}, opts)
	}
}
