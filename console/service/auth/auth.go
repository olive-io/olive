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

	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
)

type Service struct {
	ctx context.Context
	cfg *config.Config

	lg  *zap.Logger
	oct *client.Client

	userDao *dao.UserDao
}

func NewAuth(ctx context.Context, cfg *config.Config, oct *client.Client) (*Service, error) {

	userDao := dao.NewUser()
	s := &Service{
		ctx:     ctx,
		cfg:     cfg,
		lg:      cfg.GetLogger(),
		oct:     oct,
		userDao: userDao,
	}

	return s, nil
}

func (s *Service) Login(ctx context.Context, username string, password string) (*types.Token, error) {
	token := &types.Token{}
	return token, nil
}

func (s *Service) Register(ctx context.Context, username, password, email, phone string) (*types.User, error) {
	user := &types.User{}
	return user, nil
}
