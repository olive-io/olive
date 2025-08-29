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
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/olive-io/olive/api/types"
)

type ListProcessOptions struct {
	DefinitionId       int64
	definitionsVersion uint64
	ProcessStatus      types.Process_ProcessStatus
	ProcessStage       types.Process_ProcessStage
}

func NewListProcessOptions(id int64, version uint64) *ListProcessOptions {
	return &ListProcessOptions{
		DefinitionId:       id,
		definitionsVersion: version,
		ProcessStage:       -1,
		ProcessStatus:      -1,
	}
}

type ProcessDao struct {
	db *gorm.DB
}

func NewProcessDao(db *gorm.DB) (*ProcessDao, error) {
	err := db.AutoMigrate(
		&types.Process{},
		&types.FlowNode{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate process models: %w", err)
	}

	dao := &ProcessDao{
		db: db,
	}
	return dao, nil
}

func (dao *ProcessDao) ListProcesses(ctx context.Context, page, size int32, options *ListProcessOptions) ([]*types.Process, int64, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Process{})

	countTx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Process{})
	if options != nil {
		if defId := options.DefinitionId; defId > 0 {
			tx = tx.Where("definitions_id = ?", defId)
			countTx = countTx.Where("definitions_version = ?", defId)
		}
		if defVersion := options.definitionsVersion; defVersion > 0 {
			tx = tx.Where("definitions_version = ?", defVersion)
			countTx = countTx.Where("definitions_version = ?", defVersion)
		}
		if status := options.ProcessStatus; status != -1 {
			tx = tx.Where("status = ?", status)
			countTx = countTx.Where("status = ?", status)
		}
		if stage := options.ProcessStage; stage != -1 {
			tx = tx.Where("process_stage = ?", stage)
		}
	}

	total := int64(0)
	if err := countTx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * size)
	limit := int(size)
	processes := make([]*types.Process, 0)

	if offset > 0 {
		tx.Offset(offset)
	}
	if limit > 0 {
		tx.Limit(limit)
	}

	tx = tx.Order("id desc")
	err := tx.Find(&processes).Error
	if err != nil {
		return nil, 0, err
	}

	return processes, total, nil
}

func (dao *ProcessDao) GetProcess(ctx context.Context, id int64) (*types.Process, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Process{})

	var process types.Process
	err := tx.Where("id = ?", id).First(&process).Error
	if err != nil {
		return nil, err
	}

	return &process, nil
}

func (dao *ProcessDao) ListFlowNodes(ctx context.Context, pid int64) ([]*types.FlowNode, error) {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.FlowNode{})

	nodes := make([]*types.FlowNode, 0)
	err := tx.Where("process_id = ?", pid).Find(&nodes).Error
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

func (dao *ProcessDao) CreateProcess(ctx context.Context, process *types.Process) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Process{})
	if err := tx.Create(process).Error; err != nil {
		return err
	}
	return nil
}

func (dao *ProcessDao) UpdateProcess(ctx context.Context, process *types.Process) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.Process{})
	if err := tx.Where("id = ?", process.Id).Updates(process).Error; err != nil {
		return err
	}
	return nil
}

func (dao *ProcessDao) SaveFlowNode(ctx context.Context, node *types.FlowNode) error {
	tx := dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.FlowNode{})

	if node.FlowId != "" {
		tx = tx.Where("flow_id = ?", node.FlowId)
	}
	if node.ProcessId != 0 {
		tx = tx.Where("process_id = ?", node.ProcessId)
	}
	var target types.FlowNode
	if err := tx.First(&target).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		tx = dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.FlowNode{})
		if err = tx.Create(node).Error; err != nil {
			return err
		}
		return nil
	}

	tx = dao.db.Session(&gorm.Session{}).WithContext(ctx).Model(&types.FlowNode{})
	if node.FlowId != "" {
		tx = tx.Where("flow_id = ?", node.FlowId)
	}
	if node.ProcessId != 0 {
		tx = tx.Where("process_id = ?", node.ProcessId)
	}
	if err := tx.Updates(node).Error; err != nil {
		return err
	}
	return nil
}
