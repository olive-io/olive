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

func (dao *UserDao) ListUsers(ctx context.Context, result *model.ListResult[types.User], name, email, mobile string) error {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	if len(name) != 0 {
		tx = tx.Where("name LIKE ?", "%"+name+"%")
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
		return err
	}
	return nil
}

func (dao *UserDao) Target() *types.User {
	return new(types.User)
}
