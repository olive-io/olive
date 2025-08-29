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
	"fmt"

	"gorm.io/gorm"

	"github.com/olive-io/olive/api/types"
)

type RunnerDao struct {
	db *gorm.DB
}

func NewRunnerDao(db *gorm.DB) (*RunnerDao, error) {
	err := db.AutoMigrate(
		&types.Runner{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate runner models: %w", err)
	}

	dao := &RunnerDao{
		db: db,
	}

	return dao, nil
}

func (dao *RunnerDao) ListRunners(ctx context.Context, page, size int32) ([]*types.Runner, int64, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})

	total := int64(0)
	err := tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * size)
	limit := int(size)
	runners := make([]*types.Runner, 0)
	tx = dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})
	if offset > 0 {
		tx = tx.Offset(offset)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err = tx.Order("id desc").Find(&runners).Error
	if err != nil {
		return nil, 0, err
	}

	return runners, total, nil
}

func (dao *RunnerDao) GetRunner(ctx context.Context, id uint64, uid string) (*types.Runner, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})

	if id != 0 {
		tx = tx.Where("id = ?", id)
	}
	if uid != "" {
		tx = tx.Where("uid = ?", uid)
	}

	var runner types.Runner
	err := tx.First(&runner).Error
	if err != nil {
		return nil, err
	}
	return &runner, nil
}

func (dao *RunnerDao) CreateRunner(ctx context.Context, runner *types.Runner) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})

	err := tx.Create(runner).Error
	if err != nil {
		return err
	}
	runner.Id = uint64(tx.RowsAffected)

	return nil
}

func (dao *RunnerDao) UpdateRunner(ctx context.Context, runner *types.Runner) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})

	err := tx.Where("uid = ?", runner.Uid).Updates(runner).Error
	if err != nil {
		return err
	}
	return nil
}

func (dao *RunnerDao) RemoveRunner(ctx context.Context, id uint64, uid string) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Runner{})

	if id != 0 {
		tx = tx.Where("id = ?", id)
	}
	if uid != "" {
		tx = tx.Where("uid = ?", uid)
	}

	err := tx.Delete(&types.Runner{}).Error
	if err != nil {
		return err
	}
	return nil
}
