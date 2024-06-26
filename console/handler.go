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

package console

import (
	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse = client.AuthenticateResponse

func login(oct *client.Client) func(ctx *gin.Context, in *LoginRequest) (*LoginResponse, error) {
	return func(ctx *gin.Context, in *LoginRequest) (*LoginResponse, error) {
		resp, err := oct.Authenticate(ctx, in.Username, in.Password)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}
