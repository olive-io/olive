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

package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"

	"github.com/olive-io/olive/pkg/proxy/client"
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
