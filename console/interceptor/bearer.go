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
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/olive-io/olive/api/rpctypes"
)

type BearerAuth struct {
	rtw   sync.RWMutex
	token string
}

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

		ai.rtw.RLock()
		token := ai.token
		ai.rtw.RUnlock()

		ctx = metadata.AppendToOutgoingContext(ctx, rpctypes.TokenFieldNameGRPC, token)
		err := invoker(ctx, method, req, reply, cc, opts...)
		return err
	}
}

func (ai *BearerAuth) Handler(ctx *gin.Context) {

	authorization := ctx.GetHeader("Authorization")
	if authorization == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"msg": "no found token"})
		ctx.Abort()
		return
	}

	token := strings.TrimPrefix(authorization, "Bearer ")
	ai.rtw.Lock()
	ai.token = token
	ai.rtw.Unlock()

	ctx.Next()

	return
}
