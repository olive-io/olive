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

package client

import (
	"context"

	"google.golang.org/grpc"

	authv1 "github.com/olive-io/olive/apis/pb/auth"
	pb "github.com/olive-io/olive/apis/pb/olive"
)

type (
	ListRoleResponse     pb.ListRoleResponse
	GetRoleResponse      pb.GetRoleResponse
	CreateRoleResponse   pb.CreateRoleResponse
	UpdateRoleResponse   pb.UpdateRoleResponse
	RemoveRoleResponse   pb.RemoveRoleResponse
	ListUserResponse     pb.ListUserResponse
	GetUserResponse      pb.GetUserResponse
	CreateUserResponse   pb.CreateUserResponse
	UpdateUserResponse   pb.UpdateUserResponse
	RemoveUserResponse   pb.RemoveUserResponse
	AuthenticateResponse pb.AuthenticateResponse
)

type AuthRPC interface {
	ListRole(ctx context.Context, opts ...PageOption) (*ListRoleResponse, error)
	GetRole(ctx context.Context, name string) (*GetRoleResponse, error)
	CreateRole(ctx context.Context, role *authv1.Role) (*CreateRoleResponse, error)
	UpdateRole(ctx context.Context, name string, patcher *authv1.RolePatcher) (*UpdateRoleResponse, error)
	RemoveRole(ctx context.Context, name string) (*RemoveRoleResponse, error)
	ListUser(ctx context.Context, opts ...PageOption) (*ListUserResponse, error)
	GetUser(ctx context.Context, name string) (*GetUserResponse, error)
	CreateUser(ctx context.Context, user *authv1.User) (*CreateUserResponse, error)
	UpdateUser(ctx context.Context, name string, patcher *authv1.UserPatcher) (*UpdateUserResponse, error)
	RemoveUser(ctx context.Context, name string) (*RemoveUserResponse, error)
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

func (ac *authRPC) ListRole(ctx context.Context, opts ...PageOption) (*ListRoleResponse, error) {
	var page PageOptions
	for _, opt := range opts {
		opt(&page)
	}

	conn := ac.client.conn
	in := pb.ListRoleRequest{
		Limit:    page.Limit,
		Continue: page.Token,
	}
	rsp, err := ac.remoteClient(conn).ListRole(ctx, &in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*ListRoleResponse)(rsp), nil
}

func (ac *authRPC) GetRole(ctx context.Context, name string) (*GetRoleResponse, error) {
	conn := ac.client.conn
	in := pb.GetRoleRequest{
		Name: name,
	}
	rsp, err := ac.remoteClient(conn).GetRole(ctx, &in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*GetRoleResponse)(rsp), nil
}

func (ac *authRPC) CreateRole(ctx context.Context, role *authv1.Role) (*CreateRoleResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.CreateRoleRequest{Role: role}
	resp, err := ac.remoteClient(conn).CreateRole(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*CreateRoleResponse)(resp), nil
}

func (ac *authRPC) UpdateRole(ctx context.Context, name string, patcher *authv1.RolePatcher) (*UpdateRoleResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.UpdateRoleRequest{
		Name:    name,
		Patcher: patcher,
	}
	resp, err := ac.remoteClient(conn).UpdateRole(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*UpdateRoleResponse)(resp), nil
}

func (ac *authRPC) RemoveRole(ctx context.Context, name string) (*RemoveRoleResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.RemoveRoleRequest{Name: name}
	resp, err := ac.remoteClient(conn).RemoveRole(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*RemoveRoleResponse)(resp), nil
}

func (ac *authRPC) ListUser(ctx context.Context, opts ...PageOption) (*ListUserResponse, error) {
	var page PageOptions
	for _, opt := range opts {
		opt(&page)
	}

	conn := ac.client.conn
	in := pb.ListUserRequest{
		Limit:    page.Limit,
		Continue: page.Token,
	}
	rsp, err := ac.remoteClient(conn).ListUser(ctx, &in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*ListUserResponse)(rsp), nil
}

func (ac *authRPC) GetUser(ctx context.Context, name string) (*GetUserResponse, error) {
	conn := ac.client.conn
	in := pb.GetUserRequest{
		Name: name,
	}
	rsp, err := ac.remoteClient(conn).GetUser(ctx, &in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*GetUserResponse)(rsp), nil
}

func (ac *authRPC) CreateUser(ctx context.Context, user *authv1.User) (*CreateUserResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.CreateUserRequest{User: user}
	resp, err := ac.remoteClient(conn).CreateUser(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*CreateUserResponse)(resp), nil
}

func (ac *authRPC) UpdateUser(ctx context.Context, name string, patcher *authv1.UserPatcher) (*UpdateUserResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.UpdateUserRequest{
		Name:    name,
		Patcher: patcher,
	}
	resp, err := ac.remoteClient(conn).UpdateUser(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*UpdateUserResponse)(resp), nil
}

func (ac *authRPC) RemoveUser(ctx context.Context, name string) (*RemoveUserResponse, error) {
	conn := ac.client.conn
	leaderEndpoints, err := ac.client.leaderEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	if len(leaderEndpoints) > 0 {
		conn, err = ac.client.ec.Dial(leaderEndpoints[0])
		if err != nil {
			return nil, err
		}
	}

	in := &pb.RemoveUserRequest{Name: name}
	resp, err := ac.remoteClient(conn).RemoveUser(ctx, in, ac.callOpts...)
	if err != nil {
		return nil, toErr(ctx, err)
	}

	return (*RemoveUserResponse)(resp), nil
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
