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

package interceptor

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/olive-io/olive/api/rpctypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type BearerAuth struct{}

func NewBearerAuth() *BearerAuth {
	auth := &BearerAuth{}
	return auth
}

// Unary returns a unary interceptor.
func (ai *BearerAuth) Unary() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		endpoint := method
		if strings.HasPrefix(endpoint, "/etcdserverpb") {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		if gc, ok := ctx.(*gin.Context); ok {
			ctx = gc.Request.Context()
		}
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}

func (ai *BearerAuth) Handler(c *gin.Context) {
	authorization := c.GetHeader(rpctypes.TokenNameSwagger)
	if authorization != "" {
		ctx := metadata.NewOutgoingContext(c.Request.Context(), metadata.Pairs(rpctypes.TokenNameGRPC, authorization))
		c.Request = c.Request.WithContext(ctx)
	}

	c.Next()

	return
}
