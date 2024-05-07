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

package meta

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
)

type authServer struct {
	pb.UnsafeAuthRPCServer
	pb.UnsafeRbacRPCServer

	*Server
}

func (s *authServer) ListRole(ctx context.Context, req *pb.ListRoleRequest) (resp *pb.ListRoleResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) GetRole(ctx context.Context, req *pb.GetRoleRequest) (resp *pb.GetRoleResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) ListUser(ctx context.Context, req *pb.ListUserRequest) (resp *pb.ListUserResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (resp *pb.GetUserResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (resp *pb.CreateRoleResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (resp *pb.UpdateRoleResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (resp *pb.RemoveRoleResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (resp *pb.CreateUserResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (resp *pb.UpdateUserResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (resp *pb.RemoveUserResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) Authenticate(ctx context.Context, req *pb.AuthenticateRequest) (resp *pb.AuthenticateResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) ListPolicy(ctx context.Context, req *pb.ListPolicyRequest) (resp *pb.ListPolicyResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) AddPolicy(ctx context.Context, req *pb.AddPolicyRequest) (resp *pb.AddPolicyResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) RemovePolicy(ctx context.Context, req *pb.RemovePolicyRequest) (resp *pb.RemovePolicyResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) AddGroupPolicy(ctx context.Context, req *pb.AddGroupPolicyRequest) (resp *pb.AddGroupPolicyResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) RemoveGroupPolicy(ctx context.Context, req *pb.RemoveGroupPolicyRequest) (resp *pb.RemoveGroupPolicyResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *authServer) Admit(ctx context.Context, req *pb.AdmitRequest) (resp *pb.AdmitResponse, err error) {
	//TODO implement me
	panic("implement me")
}
