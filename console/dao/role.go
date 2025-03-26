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

type RoleDao struct{}

func NewRole() *RoleDao {
	dao := &RoleDao{}
	return dao
}

func (dao *RoleDao) ListRoles(ctx context.Context, result *model.ListResult[types.Role], name string) error {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	if len(name) != 0 {
		tx = tx.Where("name LIKE ?", "%"+name+"%")
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

func (dao *RoleDao) GetById(ctx context.Context, id int64) (*types.Role, error) {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	user := &types.Role{}
	err := tx.Where("id = ?", id).First(user).Error
	if err != nil {
		return nil, parseErr(err)
	}
	return user, nil
}

func (dao *RoleDao) Target() *types.Role {
	return new(types.Role)
}
