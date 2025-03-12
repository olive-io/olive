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

package bpmn

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/glebarez/sqlite"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/leader"
)

// VersionedDefinition saves Definition id and version
type VersionedDefinition struct {
	Id int64 `gorm:"primary_key"`

	// definition id
	Definition int64
	// definition version
	Version uint64
}

// NextDefinition saves Definition id and next version
type NextDefinition struct {
	Id      int64 `gorm:"primary_key"`
	Current uint64
}

// Process links *types.Definitions with *types.Process
type Process struct {
	Id         int64 `gorm:"primary_key"`
	Definition int64
	Version    uint64
	ProcessId  int64
}

type DefinitionStorage struct {
	ctx   context.Context
	v3cli *clientv3.Client

	db *gorm.DB
}

func NewDefinitionStorage(ctx context.Context, v3cli *clientv3.Client, workdir string) (*DefinitionStorage, error) {

	dsn := sqlite.Open(filepath.Join(workdir, "mon.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"))
	db, err := gorm.Open(dsn, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, errors.Wrap(err, "open database")
	}

	if err := db.AutoMigrate(
		&VersionedDefinition{},
		&NextDefinition{},
		&Process{},
	); err != nil {
		return nil, errors.Wrap(err, "auto migrate")
	}

	ds := &DefinitionStorage{
		ctx:   ctx,
		v3cli: v3cli,
		db:    db,
	}

	go ds.watching(ctx)

	return ds, nil
}

func (ds *DefinitionStorage) AddDefinition(ctx context.Context, definition *types.Definition) error {
	nextVersion := ds.definitionNextVersion(ctx, definition.Id)
	definition.Version = nextVersion

	data, err := proto.Marshal(definition)
	if err != nil {
		return err
	}

	if err = ds.addDefinitionVersion(ctx, definition.Id, nextVersion); err != nil {
		return err
	}

	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", definition.Id), fmt.Sprintf("%d", nextVersion))
	_, err = ds.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return err
	}

	return nil
}

func (ds *DefinitionStorage) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {

	if version == 0 {
		version = ds.getDefinitionLastVersion(ctx, id)
	} else {
		if ok := ds.isExistsDefinitionVersion(ctx, id, version); !ok {
			return nil, fmt.Errorf("definition not found")
		}
	}
	if version == 0 {
		return nil, fmt.Errorf("definition not found")
	}

	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", id), fmt.Sprintf("%d", version))
	opts := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}
	resp, err := ds.v3cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	definition, err := parseDefinition(resp.Kvs[0])
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (ds *DefinitionStorage) ListDefinitions(ctx context.Context, page, size int32) ([]*types.Definition, int64, error) {

	items, total := ds.listDefinitions(ctx, page, size)

	definitions := make([]*types.Definition, 0, size)

	for _, item := range items {
		key := path.Join(api.DefinitionPrefix,
			fmt.Sprintf("%d", item.Definition),
			fmt.Sprintf("%d", item.Version))
		resp, err := ds.v3cli.Get(ctx, key, clientv3.WithSerializable())
		if err != nil {
			continue
		}

		definition, err := parseDefinition(resp.Kvs[0])
		if err != nil {
			continue
		}

		definitions = append(definitions, definition)
	}

	return definitions, total, nil
}

func (ds *DefinitionStorage) RemoveDefinition(ctx context.Context, id int64, version uint64) error {
	ok := ds.isExistsDefinitionVersion(ctx, id, version)
	if !ok {
		return fmt.Errorf("definition not found")
	}

	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", id), fmt.Sprintf("%d", version))
	_, err := ds.v3cli.Delete(ctx, key)
	if err != nil {
		return err
	}

	_ = ds.removeDefinitionVersion(ctx, id, version)
	return nil
}

func (ds *DefinitionStorage) ListProcess(ctx context.Context, definition int64, version uint64, page, size int32) ([]*types.Process, int64, error) {
	items, total := ds.listProcess(ctx, definition, version, page, size)

	processes := make([]*types.Process, 0, size)

	for _, item := range items {
		key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", item.ProcessId))
		resp, err := ds.v3cli.Get(ctx, key, clientv3.WithSerializable())
		if err != nil {
			continue
		}

		process, err := parseProcess(resp.Kvs[0])
		if err != nil {
			continue
		}

		processes = append(processes, process)
	}

	return processes, total, nil
}

func (ds *DefinitionStorage) AddProcess(ctx context.Context, process *types.Process) error {

	key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", process.Id))
	data, err := proto.Marshal(process)
	if err != nil {
		return err
	}

	_, err = ds.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return err
	}

	if ok := ds.isExistsProcess(ctx, process); !ok {
		_ = ds.addProcess(ctx, process)
	}

	return nil
}

func (ds *DefinitionStorage) RemoveProcess(ctx context.Context, process *types.Process) error {
	key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", process.Id))
	_, err := ds.v3cli.Delete(ctx, key)
	if err != nil {
		return err
	}

	_ = ds.removeProcess(ctx, process)
	return nil
}

func (ds *DefinitionStorage) watching(ctx context.Context) {

	select {
	case <-ds.ctx.Done():
		return
	case <-leader.WaitReadyc():
	}

	ds.fetchAllDefinitions(ctx)
	ds.fetchAllProcesses(ctx)

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithPrevKV(),
	}

	for {

		w := ds.v3cli.Watch(ctx, api.DefinitionPrefix, opts...)
		pw := ds.v3cli.Watch(ctx, api.ProcessPrefix, opts...)
	LOOP:
		for {
			select {
			case <-ctx.Done():
				return

			case wch, ok := <-w:
				if !ok {
					break LOOP
				}
				if wch.Err() != nil {
					break LOOP
				}

				for _, ev := range wch.Events {
					switch {
					case ev.Type == mvccpb.PUT && ev.IsCreate():
						definition, err := parseDefinition(ev.Kv)
						if err != nil {
							continue
						}

						_ = ds.addDefinitionVersion(ctx, definition.Id, definition.Version)

					case ev.Type == mvccpb.DELETE:
						definition, err := parseDefinition(ev.PrevKv)
						if err != nil {
							continue
						}
						_ = ds.removeDefinitionVersion(ctx, definition.Id, definition.Version)
					}
				}
			case wch, ok := <-pw:
				if !ok {
					break LOOP
				}
				if wch.Err() != nil {
					break LOOP
				}

				for _, ev := range wch.Events {
					switch {
					case ev.Type == mvccpb.PUT && ev.IsCreate():
						process, err := parseProcess(ev.Kv)
						if err != nil {
							continue
						}

						_ = ds.addProcess(ctx, process)

					case ev.Type == mvccpb.DELETE:
						process, err := parseProcess(ev.PrevKv)
						if err != nil {
							continue
						}
						_ = ds.removeProcess(ctx, process)
					}
				}
			}

		}
	}
}

func (ds *DefinitionStorage) fetchAllDefinitions(ctx context.Context) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := ds.v3cli.Get(ctx, api.DefinitionPrefix, opts...)
	if err != nil {
		return
	}

	for _, kv := range resp.Kvs {
		definition, err := parseDefinition(kv)
		if err != nil {
			continue
		}

		if ok := ds.isExistsDefinitionVersion(ctx, definition.Id, definition.Version); !ok {
			_ = ds.addDefinitionVersion(ctx, definition.Id, definition.Version)
		}
	}
}

func (ds *DefinitionStorage) fetchAllProcesses(ctx context.Context) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := ds.v3cli.Get(ctx, api.ProcessPrefix, opts...)
	if err != nil {
		return
	}

	for _, kv := range resp.Kvs {
		process, err := parseProcess(kv)
		if err != nil {
			continue
		}

		if ok := ds.isExistsProcess(ctx, process); !ok {
			_ = ds.addProcess(ctx, process)
		}
	}
}

func (ds *DefinitionStorage) isExistsDefinitionVersion(ctx context.Context, id int64, version uint64) bool {
	var count int64
	ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Where("definition = ?", id).
		Where("version = ?", version).
		Count(&count)
	return count > 0
}

func (ds *DefinitionStorage) getDefinitionLastVersion(ctx context.Context, id int64) uint64 {

	var version uint64
	ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Where("definition = ?", id).
		Order("version desc").
		First(&version)

	return version
}

func (ds *DefinitionStorage) listDefinitions(ctx context.Context, page, size int32) ([]*VersionedDefinition, int64) {

	var total int64
	ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Select("definition").
		Group("definition").
		Count(&total)

	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * size
	limit := size
	definitions := make([]*VersionedDefinition, 0)
	ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Select("definition, max(version) as version").
		Group("definition").
		Offset(int(offset)).
		Limit(int(limit)).
		Find(&definitions)

	return definitions, total
}

func (ds *DefinitionStorage) listDefinitionVersion(ctx context.Context, id int64) []uint64 {
	versions := make([]uint64, 0)

	ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Select("version").
		Where("definition = ?", id).
		Order("version desc").
		Find(&versions)

	return versions
}

func (ds *DefinitionStorage) addDefinitionVersion(ctx context.Context, id int64, version uint64) error {
	vd := &VersionedDefinition{
		Definition: id,
		Version:    version,
	}
	err := ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Where("definition = ?", id).
		Where("version = ?", version).
		FirstOrCreate(&vd).Error

	ds.setDefinitionVersion(ctx, id, version)
	return err
}

func (ds *DefinitionStorage) removeDefinitionVersion(ctx context.Context, id int64, version uint64) error {
	vd := &VersionedDefinition{
		Definition: id,
		Version:    version,
	}
	err := ds.db.WithContext(ctx).Model(new(VersionedDefinition)).
		Session(&gorm.Session{}).
		Delete(&vd).Error

	return err
}

func (ds *DefinitionStorage) definitionNextVersion(ctx context.Context, id int64) uint64 {
	next := NextDefinition{}

	tx := ds.db.WithContext(ctx).Model(new(NextDefinition))
	if err := tx.Session(&gorm.Session{}).Where("id = ?", id).First(&next).Error; err != nil {
		next = NextDefinition{Id: id, Current: 0}
		tx.Session(&gorm.Session{}).Save(&next)
	}

	return next.Current + 1
}

func (ds *DefinitionStorage) setDefinitionVersion(ctx context.Context, id int64, version uint64) {
	next := NextDefinition{Id: id, Current: version}

	tx := ds.db.WithContext(ctx).Model(new(NextDefinition))
	tx.Session(&gorm.Session{}).Where("id = ?", id).Save(&next)
}

func (ds *DefinitionStorage) listProcess(ctx context.Context, definition int64, version uint64, page, size int32) ([]*Process, int64) {
	if page <= 0 {
		page = 1
	}

	tx1 := ds.db.WithContext(ctx).Model(new(Process)).Session(&gorm.Session{})
	tx2 := ds.db.WithContext(ctx).Model(new(Process)).Session(&gorm.Session{})

	if definition != 0 {
		tx1 = tx1.Where("definition = ?", definition)
		tx2 = tx2.Where("definition = ?", version)
	}
	if version != 0 {
		tx1 = tx1.Where("version = ?", version)
		tx2 = tx2.Where("version = ?", version)
	}

	var total int64
	tx1.Count(&total)

	offset := (page - 1) * size
	limit := size
	ps := make([]*Process, 0)
	tx2.Offset(int(offset)).
		Limit(int(limit)).
		Find(&ps)

	return ps, total
}

func (ds *DefinitionStorage) isExistsProcess(ctx context.Context, process *types.Process) bool {
	var count int64
	ds.db.WithContext(ctx).Model(new(Process)).
		Session(&gorm.Session{}).
		Where("definition = ?", process.DefinitionsId).
		Where("version = ?", process.DefinitionsVersion).
		Where("process_id = ?", process.Id).
		Count(&count)

	return count > 0
}

func (ds *DefinitionStorage) addProcess(ctx context.Context, process *types.Process) error {
	p := &Process{
		Definition: process.DefinitionsId,
		Version:    process.DefinitionsVersion,
		ProcessId:  process.Id,
	}
	err := ds.db.WithContext(ctx).Model(new(Process)).
		Session(&gorm.Session{}).
		Where("definition = ?", process.DefinitionsId).
		Where("version = ?", process.DefinitionsVersion).
		Where("process_id = ?", process.Id).
		FirstOrCreate(&p).Error

	return err
}

func (ds *DefinitionStorage) removeProcess(ctx context.Context, process *types.Process) error {
	p := &Process{}
	err := ds.db.WithContext(ctx).Model(new(Process)).
		Session(&gorm.Session{}).
		Where("definition = ?", process.DefinitionsId).
		Where("version = ?", process.DefinitionsVersion).
		Where("process_id = ?", process.Id).
		Delete(&p).Error

	return err
}

func parseDefinition(kv *mvccpb.KeyValue) (*types.Definition, error) {
	var definition types.Definition
	if err := proto.Unmarshal(kv.Value, &definition); err != nil {
		return nil, err
	}
	return &definition, nil
}

func parseProcess(kv *mvccpb.KeyValue) (*types.Process, error) {
	var process types.Process
	if err := proto.Unmarshal(kv.Value, &process); err != nil {
		return nil, err
	}
	return &process, nil
}
