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

package jwt

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/golang-jwt/jwt/v5"

	authv1 "github.com/olive-io/olive/apis/rpc/auth"
	"github.com/olive-io/olive/apis/rpctypes"
)

var DefaultSalt = "Olive.io"

type Claims struct {
	jwt.RegisteredClaims
	User *authv1.User
}

func NewClaims(user *authv1.User) *Claims {
	return &Claims{User: user}
}

func (c *Claims) GenerateToken(duration time.Duration) (*authv1.Token, error) {
	startTime := time.Now()
	endTime := time.Now().Add(duration)
	c.IssuedAt = jwt.NewNumericDate(startTime)
	c.ExpiresAt = jwt.NewNumericDate(endTime)

	// 生成token字符串
	tokenText, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(DefaultSalt))
	if err != nil {
		return nil, err
	}

	token := &authv1.Token{
		TokenText:      tokenText,
		Role:           c.User.Role,
		User:           c.User.Name,
		StartTimestamp: startTime.Unix(),
		EndTimestamp:   endTime.Unix(),
	}

	return token, nil
}

func ParseToken(tokenText string) (*authv1.User, error) {
	keyFn := func(token *jwt.Token) (interface{}, error) {
		return []byte(DefaultSalt), nil
	}
	token, err := jwt.ParseWithClaims(tokenText, &Claims{}, keyFn)
	if err != nil {
		return nil, errors.Wrap(rpctypes.ErrGRPCInvalidAuthToken, err.Error())
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, rpctypes.ErrGRPCInvalidAuthToken
	}
	if !token.Valid {
		return nil, rpctypes.ErrGRPCInvalidAuthToken
	}
	return claims.User, nil
}
