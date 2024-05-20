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

	"github.com/cockroachdb/errors"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/olive-io/olive/api/pb/auth"
	pb "github.com/olive-io/olive/api/pb/olive"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/pkg/crypto"
	"github.com/olive-io/olive/pkg/jwt"
	ort "github.com/olive-io/olive/pkg/runtime"
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
	rootRoleKey := path.Join(ort.DefaultRolePrefix, ort.DefaultRootRole)
	_, _, err := s.get(ctx, rootRoleKey, new(authv1.Role), getOpts...)
	if err != nil {
		rootRole := &authv1.Role{
			Name:              ort.DefaultRootRole,
			Metadata:          map[string]string{},
			Namespace:         ort.DefaultNamespace,
			CreationTimestamp: time.Now().Unix(),
		}
		_, err := s.put(ctx, rootRoleKey, rootRole)
		if err != nil {
			return err
		}
	}

	rootUserKey := path.Join(ort.DefaultUserPrefix, ort.DefaultRootUser)
	_, _, err = s.get(ctx, rootUserKey, new(authv1.User), getOpts...)
	if err != nil {
		passwd := crypto.NewSha256().Hash([]byte(ort.DefaultPassword))
		rootUser := &authv1.User{
			Name:              ort.DefaultRootUser,
			Metadata:          map[string]string{},
			Role:              ort.DefaultRootRole,
			Namespace:         ort.DefaultNamespace,
			Password:          passwd,
			CreationTimestamp: time.Now().Unix(),
		}
		_, err = s.put(ctx, rootUserKey, rootUser)
		if err != nil {
			return err
		}
	}

	systemUserKey := path.Join(ort.DefaultUserPrefix, ort.DefaultSystemUser)
	_, _, err = s.get(ctx, systemUserKey, new(authv1.User), getOpts...)
	if err != nil {
		passwd := crypto.NewSha256().Hash([]byte(ort.DefaultPassword))
		systemUser := &authv1.User{
			Name:              ort.DefaultSystemUser,
			Metadata:          map[string]string{},
			Role:              ort.DefaultRootRole,
			Namespace:         ort.DefaultNamespace,
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

	lg := s.lg
	resp = &pb.ListRoleResponse{}
	key := ort.DefaultRolePrefix

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
	header, role, err := s.getRole(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	resp = &pb.GetRoleResponse{
		Header: header,
		Role:   role,
	}
	return
}

func (s *authServer) getRole(ctx context.Context, name string) (*pb.ResponseHeader, *authv1.Role, error) {
	key := path.Join(ort.DefaultRolePrefix, name)
	role := &authv1.Role{}
	header, _, err := s.get(ctx, key, role, clientv3.WithSerializable())
	if err != nil {
		if errors.Is(err, rpctypes.ErrKeyNotFound) {
			err = rpctypes.ErrRoleNotFound
		}
		return nil, nil, err
	}
	return toHeader(header), role, nil
}

func (s *authServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (resp *pb.CreateRoleResponse, err error) {
	_, err = s.mustBeRoot(ctx)
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == nil {
		return nil, rpctypes.ErrEmptyValue
	}
	_, exists, err := s.getRole(ctx, role.Name)
	if !errors.Is(err, rpctypes.ErrRoleNotFound) {
		return nil, err
	}
	if exists != nil {
		return nil, rpctypes.ErrGRPCRoleAlreadyExist
	}

	role.CreationTimestamp = time.Now().Unix()
	role.UpdateTimestamp = role.CreationTimestamp
	key := path.Join(ort.DefaultRolePrefix, role.Name)
	header, err := s.put(ctx, key, role)
	if err != nil {
		return nil, err
	}
	resp = &pb.CreateRoleResponse{
		Header: toHeader(header),
		Role:   role,
	}
	return
}

func (s *authServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (resp *pb.UpdateRoleResponse, err error) {
	patcher := req.Patcher
	if patcher == nil {
		return nil, rpctypes.ErrEmptyValue
	}
	_, role, err := s.getRole(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	user, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if user.Role != role.Name && user.Role != ort.DefaultRootRole {
		return nil, rpctypes.ErrGRPCInvalidAuthMgmt
	}

	if patcher.Desc != "" {
		role.Desc = patcher.Desc
	}
	if patcher.Namespace != "" {
		role.Namespace = patcher.Namespace
	}
	if patcher.Metadata != nil {
		role.Metadata = patcher.Metadata
	}
	role.UpdateTimestamp = time.Now().Unix()
	key := path.Join(ort.DefaultRolePrefix, role.Name)
	header, err := s.put(ctx, key, role)
	if err != nil {
		return nil, err
	}
	resp = &pb.UpdateRoleResponse{
		Header: toHeader(header),
		Role:   role,
	}
	return
}

func (s *authServer) RemoveRole(ctx context.Context, req *pb.RemoveRoleRequest) (resp *pb.RemoveRoleResponse, err error) {
	name := req.Name
	if name == ort.DefaultRootRole {
		return nil, rpctypes.ErrGRPCInvalidAuthMgmt
	}

	if _, err = s.mustBeRoot(ctx); err != nil {
		return nil, err
	}

	_, role, err := s.getRole(ctx, name)
	if err != nil {
		return nil, err
	}
	key := path.Join(ort.DefaultRolePrefix, role.Name)
	header, err := s.del(ctx, key)
	if err != nil {
		return nil, err
	}
	resp = &pb.RemoveRoleResponse{
		Header: toHeader(header),
		Role:   role,
	}
	return
}

func (s *authServer) ListUser(ctx context.Context, req *pb.ListUserRequest) (resp *pb.ListUserResponse, err error) {

	lg := s.lg
	resp = &pb.ListUserResponse{}
	key := ort.DefaultUserPrefix

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
	header, user, err := s.getUser(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	resp = &pb.GetUserResponse{
		Header: header,
		User:   user,
	}
	return
}

func (s *authServer) getUser(ctx context.Context, name string) (*pb.ResponseHeader, *authv1.User, error) {
	key := path.Join(ort.DefaultUserPrefix, name)
	user := &authv1.User{}
	header, _, err := s.get(ctx, key, user, clientv3.WithSerializable())
	if err != nil {
		return nil, nil, err
	}
	return toHeader(header), user, nil
}

func (s *authServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (resp *pb.CreateUserResponse, err error) {
	user := req.User
	if user == nil {
		return nil, rpctypes.ErrEmptyValue
	}
	_, exists, err := s.getUser(ctx, user.Name)
	if !errors.Is(err, rpctypes.ErrRoleNotFound) {
		return nil, err
	}
	if exists != nil {
		return nil, rpctypes.ErrGRPCRoleAlreadyExist
	}

	user.CreationTimestamp = time.Now().Unix()
	user.UpdateTimestamp = user.CreationTimestamp
	key := path.Join(ort.DefaultUserPrefix, user.Name)
	header, err := s.put(ctx, key, user)
	if err != nil {
		return nil, err
	}
	resp = &pb.CreateUserResponse{
		Header: toHeader(header),
		User:   user,
	}
	return
}

func (s *authServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (resp *pb.UpdateUserResponse, err error) {
	patcher := req.Patcher
	if patcher == nil {
		return nil, rpctypes.ErrEmptyValue
	}
	_, user, err := s.getUser(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	current, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}

	if user.Role == ort.DefaultRootRole && current.Name != user.Name {
		return nil, rpctypes.ErrGRPCInvalidAuthMgmt
	}
	if user.Role != ort.DefaultRootRole &&
		current.Name != user.Name &&
		current.Role != ort.DefaultRootRole {
		return nil, rpctypes.ErrGRPCInvalidAuthMgmt
	}

	if patcher.Desc != "" {
		user.Desc = patcher.Desc
	}
	if patcher.Namespace != "" {
		user.Namespace = patcher.Namespace
	}
	if patcher.Metadata != nil {
		user.Metadata = patcher.Metadata
	}
	if patcher.Password != "" {
		passwd := crypto.NewSha256().Hash([]byte(patcher.Password))
		user.Password = passwd
	}
	user.UpdateTimestamp = time.Now().Unix()
	key := path.Join(ort.DefaultUserPrefix, user.Name)
	header, err := s.put(ctx, key, user)
	if err != nil {
		return nil, err
	}
	resp = &pb.UpdateUserResponse{
		Header: toHeader(header),
		User:   user,
	}
	return
}

func (s *authServer) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (resp *pb.RemoveUserResponse, err error) {
	username := req.Name
	if username == ort.DefaultRootRole || username == ort.DefaultSystemUser {
		return nil, rpctypes.ErrGRPCInvalidAuthMgmt
	}
	_, err = s.mustBeRoot(ctx)
	if err != nil {
		return nil, err
	}

	_, user, err := s.getUser(ctx, username)
	if err != nil {
		return nil, err
	}
	key := path.Join(ort.DefaultUserPrefix, user.Name)
	header, err := s.del(ctx, key)
	if err != nil {
		return nil, err
	}
	resp = &pb.RemoveUserResponse{
		Header: toHeader(header),
		User:   user,
	}
	return
}

func (s *authServer) Authenticate(ctx context.Context, req *pb.AuthenticateRequest) (resp *pb.AuthenticateResponse, err error) {
	options := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}
	key := path.Join(ort.DefaultUserPrefix, req.Name)
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
