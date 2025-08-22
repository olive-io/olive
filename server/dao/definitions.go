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

type DefinitionsDao struct {
	db *gorm.DB
}

func NewDefinitionsDao(db *gorm.DB) (*DefinitionsDao, error) {
	err := db.AutoMigrate(
		&types.Definitions{},
		&types.DefinitionsSnapshot{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate definitions schemas: %w", err)
	}

	dao := &DefinitionsDao{
		db: db,
	}

	return dao, nil
}

func (dao *DefinitionsDao) ListDefinitions(ctx context.Context, page, size int32) ([]*types.Definitions, int64, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Definitions{})

	total := int64(0)
	err := tx.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * size)
	limit := int(size)
	definitionsList := make([]*types.Definitions, 0)
	tx = dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Definitions{})
	if offset > 0 {
		tx = tx.Offset(offset)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err = tx.Order("id desc").Find(&definitionsList).Error
	if err != nil {
		return nil, 0, err
	}

	return definitionsList, total, nil
}

func (dao *DefinitionsDao) GetDefinitions(ctx context.Context, id int64, uid string) (*types.Definitions, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Definitions{})

	var definitions types.Definitions

	if id != 0 {
		tx = tx.Where("id = ?", id)
	}
	if uid != "" {
		tx = tx.Where("uid = ?", uid)
	}

	err := tx.First(&definitions).Error
	if err != nil {
		return nil, err
	}
	return &definitions, nil
}

func (dao *DefinitionsDao) CreateDefinitions(ctx context.Context, definitions *types.Definitions) (int64, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Definitions{})

	if definitions.Version == 0 {
		definitions.Version = 1
	}
	err := tx.Create(&definitions).Error
	if err != nil {
		return 0, err
	}

	return tx.RowsAffected, nil
}

func (dao *DefinitionsDao) AddSnapshots(ctx context.Context, definitions *types.Definitions) error {
	snapshot := &types.DefinitionsSnapshot{
		Uid:         definitions.Uid,
		Description: definitions.Description,
		Metadata:    definitions.Metadata,
		Content:     definitions.Content,
		Version:     definitions.Version,
	}

	tx := dao.db.WithContext(ctx).Begin()

	err := tx.Create(snapshot).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	definitions.Version += 1
	err = tx.Updates(definitions).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (dao *DefinitionsDao) GetDefinitionsSnapshots(ctx context.Context, id int64, page, size int32) ([]*types.DefinitionsSnapshot, int64, error) {
	definitions, err := dao.GetDefinitions(ctx, id, "")
	if err != nil {
		return nil, 0, err
	}

	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.DefinitionsSnapshot{})

	total := int64(0)
	err = tx.Where("uid = ?", definitions.Uid).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * size)
	limit := int(size)
	snapshots := make([]*types.DefinitionsSnapshot, 0)
	tx = dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.DefinitionsSnapshot{})

	if offset > 0 {
		tx = tx.Offset(offset)
	}
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	err = tx.Where("uid = ?", definitions.Uid).Order("id desc").Find(&snapshots).Error
	if err != nil {
		return nil, 0, err
	}
	return snapshots, total, nil
}

func (dao *DefinitionsDao) GetSnapshot(ctx context.Context, uid string, version uint64) (*types.DefinitionsSnapshot, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.DefinitionsSnapshot{})

	var snapshot types.DefinitionsSnapshot
	err := tx.Where("uid = ? AND version = ?", uid, version).First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}
