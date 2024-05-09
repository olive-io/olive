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
	"path"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/olive-io/olive/api/authpb"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/pkg/crypto"
	"github.com/olive-io/olive/pkg/jwt"
	"github.com/olive-io/olive/pkg/runtime"
)

type authServer struct {
	pb.UnsafeAuthRPCServer
	pb.UnsafeRbacRPCServer

	*Server
}

func newAuthServer(s *Server) (*authServer, error) {
	as := &authServer{Server: s}

	return as, nil
}

func (s *authServer) Prepare(ctx context.Context) error {
	getOpts := []clientv3.OpOption{
		clientv3.WithSerializable(),
		clientv3.WithKeysOnly(),
	}
	// initialize, check or create role and user
	rootRoleKey := path.Join(runtime.DefaultRolePrefix, runtime.DefaultRootRole)
	_, _, err := s.get(ctx, rootRoleKey, new(authv1.Role), getOpts...)
	if err != nil {
		rootRole := &authv1.Role{
			Name:              runtime.DefaultRootRole,
			Metadata:          map[string]string{},
			Namespace:         runtime.DefaultNamespace,
			CreationTimestamp: time.Now().Unix(),
		}
		_, err := s.put(ctx, rootRoleKey, rootRole)
		if err != nil {
			return err
		}
	}

	rootUserKey := path.Join(runtime.DefaultUserPrefix, runtime.DefaultRootUser)
	_, _, err = s.get(ctx, rootUserKey, new(authv1.User), getOpts...)
	if err != nil {
		passwd := crypto.NewSha256().Hash([]byte(runtime.DefaultPassword))
		rootUser := &authv1.User{
			Name:              runtime.DefaultRootUser,
			Metadata:          map[string]string{},
			Role:              runtime.DefaultRootRole,
			Namespace:         runtime.DefaultNamespace,
			Password:          passwd,
			CreationTimestamp: time.Now().Unix(),
		}
		_, err = s.put(ctx, rootUserKey, rootUser)
		if err != nil {
			return err
		}
	}

	systemUserKey := path.Join(runtime.DefaultUserPrefix, runtime.DefaultSystemUser)
	_, _, err = s.get(ctx, systemUserKey, new(authv1.User), getOpts...)
	if err != nil {
		passwd := crypto.NewSha256().Hash([]byte(runtime.DefaultPassword))
		systemUser := &authv1.User{
			Name:              runtime.DefaultSystemUser,
			Metadata:          map[string]string{},
			Role:              runtime.DefaultRootRole,
			Namespace:         runtime.DefaultNamespace,
			Password:          passwd,
			CreationTimestamp: time.Now().Unix(),
		}
		_, err = s.put(ctx, systemUserKey, systemUser)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *authServer) ListRole(ctx context.Context, req *pb.ListRoleRequest) (resp *pb.ListRoleResponse, err error) {
	if err = s.prepareReq(ctx, authv1.RoleReadScope); err != nil {
		return
	}

	lg := s.lg
	resp = &pb.ListRoleResponse{}
	key := runtime.DefaultRolePrefix

	roles := make([]*authv1.Role, 0)
	var continueToken string
	continueToken, err = s.pageList(ctx, key, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
		role := new(authv1.Role)
		if e1 := proto.Unmarshal(kv.Value, role); e1 != nil {
			lg.Error("unmarshal role", zap.String("key", string(kv.Key)), zap.Error(err))
			return e1
		}
		roles = append(roles, role)

		return nil
	})
	if err != nil {
		return
	}

	resp.Header = s.responseHeader()
	resp.Roles = roles
	resp.ContinueToken = continueToken
	return
}

func (s *authServer) GetRole(ctx context.Context, req *pb.GetRoleRequest) (resp *pb.GetRoleResponse, err error) {
	key := path.Join(runtime.DefaultRolePrefix, req.Name)
	role := &authv1.Role{}
	header, _, err := s.get(ctx, key, role, clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}
	resp = &pb.GetRoleResponse{
		Header: toHeader(header),
		Role:   role,
	}
	return
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

func (s *authServer) ListUser(ctx context.Context, req *pb.ListUserRequest) (resp *pb.ListUserResponse, err error) {
	if err = s.prepareReq(ctx, authv1.RoleReadScope); err != nil {
		return
	}

	lg := s.lg
	resp = &pb.ListUserResponse{}
	key := runtime.DefaultUserPrefix

	users := make([]*authv1.User, 0)
	var continueToken string
	continueToken, err = s.pageList(ctx, key, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
		user := new(authv1.User)
		if e1 := proto.Unmarshal(kv.Value, user); e1 != nil {
			lg.Error("unmarshal user", zap.String("key", string(kv.Key)), zap.Error(err))
			return e1
		}
		users = append(users, user)
		return nil
	})
	if err != nil {
		return
	}

	resp.Header = s.responseHeader()
	resp.Users = users
	resp.ContinueToken = continueToken
	return
}

func (s *authServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (resp *pb.GetUserResponse, err error) {
	key := path.Join(runtime.DefaultUserPrefix, req.Name)
	user := &authv1.User{}
	header, _, err := s.get(ctx, key, user, clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}
	resp = &pb.GetUserResponse{
		Header: toHeader(header),
		User:   user,
	}
	return
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
	options := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}
	key := path.Join(runtime.DefaultUserPrefix, req.Name)
	user := &authv1.User{}
	rh, _, err := s.get(ctx, key, user, options...)
	if err != nil {
		return nil, rpctypes.ErrGRPCAuthFailed
	}

	passwd := crypto.NewSha256().Hash([]byte(req.Password))
	if passwd != user.Password {
		return nil, rpctypes.ErrGRPCAuthFailed
	}

	expire := time.Second * time.Duration(s.cfg.AuthTokenTTL)
	token, err := jwt.NewClaims(user).GenerateToken(expire)
	if err != nil {
		return nil, err
	}

	resp = &pb.AuthenticateResponse{
		Header: toHeader(rh),
		Token:  token,
	}
	return
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
