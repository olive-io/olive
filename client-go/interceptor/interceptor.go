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

package interceptor

import (
	"context"
	"encoding/base64"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/olive-io/olive/apis/rpctypes"
)

type Interceptor interface {
	Unary() grpc.UnaryClientInterceptor
}

type BasicAuth struct {
	Username string
	Password string
}

func NewBasicAuth(username, password string) *BasicAuth {
	return &BasicAuth{Username: username, Password: password}
}

func (auth *BasicAuth) generate() string {
	text := auth.Username + ":" + auth.Password
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func (auth *BasicAuth) Unary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		endpoint := method
		if strings.HasPrefix(endpoint, "/etcdserverpb") {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		if auth.Username != "" && auth.Password != "" {
			token := auth.generate()
			ctx = metadata.AppendToOutgoingContext(ctx, rpctypes.TokenNameGRPC, rpctypes.TokenBasic+" "+token)
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}
