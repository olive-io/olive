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

package system

import (
	"context"

	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
	"github.com/olive-io/olive/console/model"
)

type Service struct {
	ctx context.Context
	cfg *config.Config
	oct *client.Client

	lg *zap.Logger

	userDao *dao.UserDao
}

func NewSystem(ctx context.Context, cfg *config.Config, oct *client.Client) (*Service, error) {

	userDao := dao.NewUser()
	s := &Service{
		ctx: ctx,
		cfg: cfg,
		oct: oct,

		lg: cfg.GetLogger(),

		userDao: userDao,
	}

	return s, nil
}

func (s *Service) ListRunners(ctx context.Context) ([]*types.Runner, error) {
	runners, err := s.oct.ListRunners(ctx)
	if err != nil {
		return nil, err
	}
	return runners, nil
}

func (s *Service) GetRunner(ctx context.Context, id uint64) (*types.Runner, *types.RunnerStat, error) {
	runner, stat, err := s.oct.GetRunner(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return runner, stat, nil
}

func (s *Service) ListUsers(ctx context.Context, page, size int32, name, email, mobile string) (*model.ListResult[types.User], error) {
	result := model.NewListResult[types.User](page, size)

	if err := s.userDao.ListUsers(ctx, result, name, email, mobile); err != nil {
		return nil, err
	}

	return result, nil
}
