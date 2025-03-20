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

package server

import (
	"context"

	pb "github.com/olive-io/olive/api/rpc/consolepb"
	"github.com/olive-io/olive/console/service/auth"
)

type AuthRPC struct {
	pb.UnimplementedAuthRPCServer

	s *auth.Service
}

func NewAuthRPC(s *auth.Service) *AuthRPC {
	rpc := &AuthRPC{s: s}
	return rpc
}

func (rpc *AuthRPC) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	token, err := rpc.s.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	resp := &pb.LoginResponse{
		Token: token,
	}
	return resp, nil
}

func (rpc *AuthRPC) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	user, err := rpc.s.Register(ctx, req.Username, req.Password, req.Email, req.Phone)
	if err != nil {
		return nil, err
	}
	resp := &pb.RegisterResponse{
		User: user,
	}
	return resp, nil
}
