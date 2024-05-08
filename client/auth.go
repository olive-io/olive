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

package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/olive-io/olive/api/olivepb"
)

type (
	AuthenticateResponse pb.AuthenticateResponse
)

type AuthRPC interface {
	Authenticate(ctx context.Context, username, password string) (*AuthenticateResponse, error)
}

type authRPC struct {
	client   *Client
	callOpts []grpc.CallOption
}

func NewAuthRPC(c *Client) AuthRPC {
	api := &authRPC{client: c}
	if c != nil {
		api.callOpts = c.callOpts
	}
	return api
}

func (ac *authRPC) ListRole(ctx context.Context, in *pb.ListRoleRequest, opts ...grpc.CallOption) (*pb.ListRoleResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) GetRole(ctx context.Context, in *pb.GetRoleRequest, opts ...grpc.CallOption) (*pb.GetRoleResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) CreateRole(ctx context.Context, in *pb.CreateRoleRequest, opts ...grpc.CallOption) (*pb.CreateRoleResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) UpdateRole(ctx context.Context, in *pb.UpdateRoleRequest, opts ...grpc.CallOption) (*pb.UpdateRoleResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) RemoveRole(ctx context.Context, in *pb.RemoveRoleRequest, opts ...grpc.CallOption) (*pb.RemoveRoleResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) ListUser(ctx context.Context, in *pb.ListUserRequest, opts ...grpc.CallOption) (*pb.ListUserResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) GetUser(ctx context.Context, in *pb.GetUserRequest, opts ...grpc.CallOption) (*pb.GetUserResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) CreateUser(ctx context.Context, in *pb.CreateUserRequest, opts ...grpc.CallOption) (*pb.CreateUserResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) UpdateUser(ctx context.Context, in *pb.UpdateUserRequest, opts ...grpc.CallOption) (*pb.UpdateUserResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) RemoveUser(ctx context.Context, in *pb.RemoveUserRequest, opts ...grpc.CallOption) (*pb.RemoveUserResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (ac *authRPC) Authenticate(ctx context.Context, username, password string) (*AuthenticateResponse, error) {
	conn := ac.client.conn
	in := &pb.AuthenticateRequest{
		Name:     username,
		Password: password,
	}

	rsp, err := ac.remoteClient(conn).Authenticate(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}
	return (*AuthenticateResponse)(rsp), nil
}

func (ac *authRPC) remoteClient(conn *grpc.ClientConn) pb.AuthRPCClient {
	return RetryAuthClient(conn)
}
