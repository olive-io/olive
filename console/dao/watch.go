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
	"sync/atomic"

	"github.com/olive-io/olive/console/model"
)

type WatchDao struct {
	currRev atomic.Int64
}

func NewWatch() *WatchDao {
	dao := &WatchDao{
		currRev: atomic.Int64{},
	}
	return dao
}

func (dao *WatchDao) GetRev(ctx context.Context) int64 {
	current := dao.currRev.Load()
	if current != 0 {
		return current
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())

	rev := &model.WatchRev{}
	if err := tx.Where("id = ?", 1).First(&rev).Error; err != nil {
		tx := GetSession().WithContext(ctx).Model(dao.Target())
		rev = &model.WatchRev{ID: 1, Revision: 0}
		tx.Create(rev)
	}
	dao.currRev.Store(current)
	return rev.Revision
}

func (dao *WatchDao) SetRev(ctx context.Context, rev int64) error {
	if rev < dao.currRev.Load() {
		return nil
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())

	if err := tx.Where("id = ?", 1).Updates(&model.WatchRev{Revision: rev}).Error; err != nil {
		return err
	}

	dao.currRev.Store(rev)
	return nil
}

func (dao *WatchDao) Clear(ctx context.Context) error {
	return dao.SetRev(ctx, 0)
}

func (dao *WatchDao) Target() *model.WatchRev {
	return new(model.WatchRev)
}
