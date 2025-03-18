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

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/console/model"
)

type DefinitionDao struct{}

func NewDefinition() *DefinitionDao {
	dao := &DefinitionDao{}
	return dao
}

func (dao *DefinitionDao) ListDefinitions(ctx context.Context, result *model.ListResult[model.Definition]) error {
	tx1 := GetSession().WithContext(ctx).Model(dao.Target())
	tx2 := GetSession().WithContext(ctx).Model(dao.Target())

	if err := tx1.Count(&result.Total).Error; err != nil {
		return err
	}

	if result.Page != -1 {
		offset, limit := result.Limit()
		tx2 = tx2.Offset(offset).Limit(limit)
	}
	if err := tx2.Find(&result.List).Error; err != nil {
		return err
	}

	return nil
}

func (dao *DefinitionDao) GetDefinition(ctx context.Context, definitionID int64, version uint64) (*model.Definition, error) {
	md := &model.Definition{}

	tx := GetSession().WithContext(ctx).Model(dao.Target())

	tx = tx.Where("definition_id = ?", definitionID)
	if version != 0 {
		tx = tx.Where("version = ?", version)
	}
	if err := tx.Order("version DESC").First(md).Error; err != nil {
		return nil, err
	}

	return md, nil
}

func (dao *DefinitionDao) AddDefinition(ctx context.Context, definition *types.Definition) error {
	md := &model.Definition{
		DefinitionID: definition.Id,
		Version:      definition.Version,
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())
	if err := tx.Create(md).Error; err != nil {
		return err
	}
	return nil
}

func (dao *DefinitionDao) DeleteDefinition(ctx context.Context, definitionID int64, version uint64) error {

	result := model.NewListResult[model.Process](-1, 10)
	if err := dao.ListProcess(ctx, definitionID, version, -1, result); err != nil {
		return err
	}

	if result.Total != 0 {
		return fmt.Errorf("definition with relational Process")
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())
	err := tx.Delete("definition_id = ? AND version = ?", definitionID, version).Error
	if err != nil {
		return err
	}
	return nil
}

func (dao *DefinitionDao) ListProcess(ctx context.Context, definitionId int64, version uint64, status types.ProcessStatus, result *model.ListResult[model.Process]) error {
	tx1 := GetSession().WithContext(ctx).Model(&model.Process{})
	tx2 := GetSession().WithContext(ctx).Model(&model.Process{})

	tx1 = tx1.Where("definition_id = ?", definitionId)
	tx2 = tx2.Where("definition_id = ?", definitionId)

	if version != 0 {
		tx1 = tx1.Where("version = ?", version)
		tx2 = tx2.Where("version = ?", version)
	}
	if status > 0 {
		tx1 = tx1.Where("status = ?", status)
		tx2 = tx2.Where("status = ?", status)
	}

	if err := tx1.Count(&result.Total).Error; err != nil {
		return err
	}

	processes := make([]model.Process, 0)
	if result.Page != -1 {
		offset, limit := result.Limit()
		tx2 = tx2.Offset(offset).Limit(limit)
	}
	if err := tx2.Order("definition_id DESC").Find(&processes).Error; err != nil {
		return err
	}

	return nil
}

func (dao *DefinitionDao) Target() *model.Definition {
	return new(model.Definition)
}
