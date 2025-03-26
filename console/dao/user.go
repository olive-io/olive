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

package dao

import (
	"context"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/console/model"
)

type UserDao struct{}

func NewUser() *UserDao {
	dao := &UserDao{}
	return dao
}

func (dao *UserDao) ListUsers(ctx context.Context, result *model.ListResult[types.User], username, email, mobile string) error {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	if len(username) != 0 {
		tx = tx.Where("username LIKE ?", "%"+username+"%")
	}
	if len(email) != 0 {
		tx = tx.Where("email LIKE ?", "%"+email+"%")
	}
	if len(mobile) != 0 {
		tx = tx.Where("mobile LIKE ?", "%"+mobile+"%")
	}

	if result.Page != -1 {
		offset, limit := result.Limit()
		tx = tx.Offset(offset).Limit(limit)
	}

	if err := tx.Order("id DESC").Find(&result.List).Error; err != nil {
		return parseErr(err)
	}
	return nil
}

func (dao *UserDao) GetBySecret(ctx context.Context, secret string) (*types.User, error) {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	user := &types.User{}
	err := tx.Where("username = ?", secret).
		Or("mobile = ?", secret).
		Or("email = ?", secret).
		First(user).Error
	if err != nil {
		return nil, parseErr(err)
	}
	return user, nil
}

func (dao *UserDao) GetById(ctx context.Context, id int64) (*types.User, error) {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	user := &types.User{}
	err := tx.Where("id = ?", id).First(user).Error
	if err != nil {
		return nil, parseErr(err)
	}
	return user, nil
}

func (dao *UserDao) Target() *types.User {
	return new(types.User)
}
