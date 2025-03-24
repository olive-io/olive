/*
Copyright 2025 The olive Authors

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

package jwtutil

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/olive-io/olive/api/types"
)

const (
	jwtSecret = "Olive System"
)

type Claims struct {
	jwt.RegisteredClaims
	Identified uint64 `json:"identified"`
	Secret     string `json:"secret"`
}

// GenerateToken generates token with 1 hour
func GenerateToken(id uint64, secret string) (*types.Token, error) {
	return GenerateTokenTTL(id, secret, time.Hour)
}

func GenerateTokenTTL(id uint64, secret string, ttl time.Duration) (*types.Token, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(ttl)

	claims := &Claims{
		Identified: id,
		Secret:     secret,
	}
	claims.ExpiresAt = jwt.NewNumericDate(expireTime)
	claims.Issuer = "olive"

	tokenClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, err := tokenClaims.SignedString([]byte(jwtSecret))
	if err != nil {
		return nil, err
	}

	token := &types.Token{
		Token:     tok,
		Timestamp: nowTime.UnixNano(),
		ExpiredAt: expireTime.Unix(),
		Enable:    true,
		UserId:    int64(id),
	}

	return token, nil
}

func ParseToken(token string) (*Claims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := tokenClaims.Claims.(*Claims)
	if !ok || !tokenClaims.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if time.Now().UTC().After(claims.ExpiresAt.UTC()) {
		return nil, fmt.Errorf("token is expired")
	}
	return claims, nil
}
