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

package auth

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	apiErr "github.com/olive-io/olive/api/errors"
	pb "github.com/olive-io/olive/api/rpc/consolepb"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
	"github.com/olive-io/olive/pkg/jwtutil"
)

type Service struct {
	ctx context.Context
	cfg *config.Config

	lg  *zap.Logger
	oct *client.Client

	userDao *dao.UserDao
	roleDao *dao.RoleDao
}

func NewAuth(ctx context.Context, cfg *config.Config, oct *client.Client) (*Service, grpc.UnaryServerInterceptor, error) {

	userDao := dao.NewUser()
	roleDao := dao.NewRole()
	s := &Service{
		ctx:     ctx,
		cfg:     cfg,
		lg:      cfg.GetLogger(),
		oct:     oct,
		userDao: userDao,
		roleDao: roleDao,
	}

	return s, s.Check, nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (*types.Token, error) {

	user, err := s.userDao.GetBySecret(ctx, username)
	if err != nil {
		return nil, err
	}

	if !user.VerifyPassword(password) {
		return nil, apiErr.NewBadRequest("invalid password")
	}

	token, err := jwtutil.GenerateToken(uint64(user.Id), user.Username)
	if err != nil {
		return nil, apiErr.NewInternal(err.Error())
	}

	return token, nil
}

func (s *Service) Register(ctx context.Context, username, password, email, phone string) (*types.User, error) {
	user := &types.User{}
	return user, nil
}

func (s *Service) Check(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	fullMethod := info.FullMethod
	switch fullMethod {
	case pb.AuthRPC_Login_FullMethodName,
		pb.AuthRPC_Register_FullMethodName:
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, apiErr.NewBadRequest("token is required").ToStatus().Err()
	}

	authorization := md["authorization"]
	if len(authorization) < 1 {
		return nil, apiErr.NewBadRequest("token is required").ToStatus().Err()
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")

	claims, err := jwtutil.ParseToken(token)
	if err != nil {
		return nil, apiErr.NewUnauthorized(err.Error()).ToStatus().Err()
	}

	user, err := s.userDao.GetById(ctx, int64(claims.Identified))
	if err != nil {
		return nil, apiErr.NewUnauthorized("invalid token").ToStatus().Err()
	}

	role, err := s.roleDao.GetById(ctx, user.RoleId)
	if err != nil {
		return nil, apiErr.NewForbidden("User with invalid role").ToStatus().Err()
	}

	tokenCtx := &jwtutil.TokenCtx{
		User: user,
		Role: role,
	}

	ctx = jwtutil.SetTokenCtx(ctx, tokenCtx)
	return handler(ctx, req)
}
