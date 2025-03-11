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
	"sync"

	"github.com/emirpasic/gods/trees/btree"
	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/leader"
)

type definitionV struct {
	id int64

	vrw      sync.RWMutex
	versions *rbt.Tree
}

func newDef(id int64) *definitionV {
	versions := rbt.NewWith(utils.UInt64Comparator)
	return &definitionV{
		id:       id,
		versions: versions,
	}
}

func (d *definitionV) ID() int64 {
	return d.id
}

func (d *definitionV) AddVersion() uint64 {
	d.vrw.Lock()
	defer d.vrw.Unlock()

	var current uint64
	if d.versions.Empty() {
		current = 1
	} else {
		maxVersion := d.versions.Right().Key.(uint64)
		current = maxVersion + 1
	}

	d.versions.Put(current, struct{}{})
	return current
}

func (d *definitionV) PushVersion(id uint64) {
	d.vrw.Lock()
	defer d.vrw.Unlock()

	d.versions.Put(id, struct{}{})
}

func (d *definitionV) AllVersions() []uint64 {
	versions := make([]uint64, 0, d.versions.Size())

	d.vrw.RLock()
	iter := d.versions.Iterator()
	for iter.Next() {
		versions = append(versions, iter.Key().(uint64))
	}
	d.vrw.RUnlock()
	return versions
}

func (d *definitionV) RemoveVersion(version uint64) {
	d.vrw.Lock()
	defer d.vrw.Unlock()
	d.versions.Remove(version)
}

func (d *definitionV) FindVersion(version uint64) bool {
	d.vrw.RLock()
	_, exists := d.versions.Get(version)
	d.vrw.RUnlock()
	return exists
}

func (d *definitionV) LastVersion() uint64 {
	d.vrw.RLock()
	defer d.vrw.RUnlock()

	if d.versions.Empty() {
		return 0
	}
	version := d.versions.Right().Key.(uint64)
	return version
}

func (d *definitionV) Clear() {
	d.id = 0
	d.vrw.Lock()
	d.versions.Clear()
	d.versions = nil
	d.vrw.Unlock()
}

type DefinitionStorage struct {
	ctx   context.Context
	v3cli *clientv3.Client

	bmu sync.RWMutex
	bt  *btree.Tree
}

func NewDefinitionStorage(ctx context.Context, v3cli *clientv3.Client) (*DefinitionStorage, error) {

	bt := btree.NewWith(3, utils.Int64Comparator)
	ds := &DefinitionStorage{
		ctx:   ctx,
		v3cli: v3cli,
		bt:    bt,
	}

	go ds.watching(ctx)

	return ds, nil
}

func (ds *DefinitionStorage) AddDefinition(ctx context.Context, definition *types.Definition) error {
	dv, ok := ds.getDefV(definition.Id)
	if !ok {
		dv = newDef(definition.Id)
	}
	currVersion := dv.AddVersion()
	definition.Version = currVersion

	ds.putDefV(dv)

	data, err := proto.Marshal(definition)
	if err != nil {
		return err
	}

	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", definition.Id), fmt.Sprintf("%d", currVersion))
	_, err = ds.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return err
	}
	return nil
}

func (ds *DefinitionStorage) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	dv, ok := ds.getDefV(id)
	if !ok {
		return nil, fmt.Errorf("definition not found")
	}

	if version == 0 {
		version = dv.LastVersion()
	} else {
		if ok = dv.FindVersion(version); !ok {
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

	total := ds.Counts()
	if size <= 0 {
		return nil, total, nil
	}

	offset := (page - 1) * size
	iter := ds.bt.Iterator()
	for i := 0; i < int(offset); i++ {
		if !iter.Next() {
			break
		}
	}

	definitions := make([]*types.Definition, 0, size)

	pos := int32(0)
	for pos < size {
		if !iter.Next() {
			break
		}

		id := iter.Key().(int64)
		version := iter.Value().(*definitionV).LastVersion()

		key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", id), fmt.Sprintf("%d", version))
		resp, err := ds.v3cli.Get(ctx, key, clientv3.WithSerializable())
		if err != nil {
			continue
		}

		definition, err := parseDefinition(resp.Kvs[0])
		if err != nil {
			continue
		}

		definitions = append(definitions, definition)

		pos += 1
	}

	return definitions, total, nil
}

func (ds *DefinitionStorage) RemoveDefinition(ctx context.Context, id int64, version uint64) error {
	dv, ok := ds.getDefV(id)
	if !ok {
		return fmt.Errorf("definition not found")
	}
	ok = dv.FindVersion(version)
	if !ok {
		return fmt.Errorf("definition not found")
	}

	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", id), fmt.Sprintf("%d", version))
	_, err := ds.v3cli.Delete(ctx, key)
	if err != nil {
		return err
	}

	dv.RemoveVersion(version)
	ds.putDefV(dv)
	return nil
}

func (ds *DefinitionStorage) Counts() int64 {
	ds.bmu.RLock()
	defer ds.bmu.RUnlock()
	return int64(ds.bt.Size())
}

func (ds *DefinitionStorage) watching(ctx context.Context) {

	select {
	case <-ds.ctx.Done():
		return
	case <-leader.WaitReadyc():
	}

	ds.fetchAllDefinitions(ctx)

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithPrevKV(),
	}

	for {

		w := ds.v3cli.Watch(ctx, api.DefinitionPrefix, opts...)
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

						dv, ok := ds.getDefV(definition.Id)
						if !ok {
							dv = newDef(definition.Id)
						}
						dv.PushVersion(definition.Version)
						ds.putDefV(dv)

					case ev.Type == mvccpb.DELETE:
						definition, err := parseDefinition(ev.Kv)
						if err != nil {
							continue
						}
						dv, ok := ds.getDefV(definition.Id)
						if !ok {
							continue
						}

						dv.RemoveVersion(definition.Version)
						if len(dv.AllVersions()) == 0 {
							ds.removeDefV(definition.Id)
						} else {
							ds.putDefV(dv)
						}
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

	dvs := map[int64]*definitionV{}
	for _, kv := range resp.Kvs {
		definition, err := parseDefinition(kv)
		if err != nil {
			continue
		}

		dfv, ok := dvs[definition.Id]
		if !ok {
			dfv = newDef(definition.Id)
		}
		dfv.PushVersion(definition.Version)
		dvs[definition.Id] = dfv
	}

	for _, dv := range dvs {
		ds.putDefV(dv)
	}
}

func (ds *DefinitionStorage) putDefV(dv *definitionV) {
	ds.bmu.Lock()
	defer ds.bmu.Unlock()
	ds.bt.Put(dv.ID(), dv)
}

func (ds *DefinitionStorage) getDefV(id int64) (*definitionV, bool) {
	ds.bmu.RLock()
	defer ds.bmu.RUnlock()

	v, ok := ds.bt.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*definitionV), true
}

func (ds *DefinitionStorage) safeUpdateDefV(id int64, update func(dv *definitionV, exists bool)) {
	ds.bmu.Lock()
	defer ds.bmu.Unlock()

	var dv *definitionV
	v, ok := ds.bt.Get(id)
	if ok {
		dv, ok = v.(*definitionV)
	}
	update(dv, ok)

	if ok {
		ds.bt.Put(id, dv)
	}
}

func (ds *DefinitionStorage) removeDefV(id int64) {
	ds.bmu.Lock()
	defer ds.bmu.Unlock()

	v, ok := ds.bt.Get(id)
	if !ok {
		return
	}

	dv := v.(*definitionV)
	dv.Clear()
	ds.bt.Remove(id)
}

func parseDefinition(kv *mvccpb.KeyValue) (*types.Definition, error) {
	var definition types.Definition
	if err := proto.Unmarshal(kv.Value, &definition); err != nil {
		return nil, err
	}
	return &definition, nil
}
