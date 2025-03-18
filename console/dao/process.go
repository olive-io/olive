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

type ProcessDao struct{}

func NewProcess() *ProcessDao {
	dao := &ProcessDao{}
	return dao
}

func (dao *ProcessDao) ListProcess(ctx context.Context, definitionId int64, version uint64, status types.ProcessStatus, result *model.ListResult[model.Process]) error {

	tx1 := GetSession().WithContext(ctx).Model(dao.Target())
	tx2 := GetSession().WithContext(ctx).Model(dao.Target())

	if definitionId != 0 {
		tx1 = tx1.Where("definition_id = ?", definitionId)
		tx2 = tx2.Where("definition_id = ?", definitionId)
	}
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

	if result.Page != -1 {
		offset, limit := result.Limit()
		tx2 = tx2.Offset(offset).Limit(limit)
	}

	if err := tx2.Order("id DESC").Find(&result.List).Error; err != nil {
		return err
	}

	return nil
}

func (dao *ProcessDao) GetProcess(ctx context.Context, id int64) (*model.Process, error) {
	tx := GetSession().WithContext(ctx).Model(dao.Target())

	mp := &model.Process{}
	if err := tx.Where("process_id = ?", id).First(mp).Error; err != nil {
		return nil, err
	}

	return mp, nil
}

func (dao *ProcessDao) UpdateProcess(ctx context.Context, process *types.Process) error {
	mp := &model.Process{
		DefinitionID: process.DefinitionsId,
		Version:      process.DefinitionsVersion,
		Status:       process.Status,
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())
	if err := tx.Where("process_id = ?", process.Id).Updates(mp).Error; err != nil {
		return err
	}
	return nil
}

func (dao *ProcessDao) AddProcess(ctx context.Context, process *types.Process) error {
	mp := &model.Process{
		ProcessID:    process.Id,
		DefinitionID: process.DefinitionsId,
		Version:      process.DefinitionsVersion,
		Status:       process.Status,
	}

	tx := GetSession().WithContext(ctx).Model(dao.Target())
	if err := tx.Create(mp).Error; err != nil {
		return err
	}
	return nil
}

func (dao *ProcessDao) DeleteProcess(ctx context.Context, id int64) error {
	tx := GetSession().WithContext(ctx).Model(dao.Target())
	if err := tx.Delete("process_id = ?", id).Error; err != nil {
		return err
	}

	return nil
}

func (dao *ProcessDao) Target() *model.Process {
	return new(model.Process)
}
