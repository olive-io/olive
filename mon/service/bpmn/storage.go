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
	"encoding/binary"
	"fmt"
	"path"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
)

const (
	definitionIds = "/olive.io/did"
)

type DefinitionStorage struct {
	ctx   context.Context
	v3cli *clientv3.Client
}

func NewDefinitionStorage(ctx context.Context, v3cli *clientv3.Client, workdir string) (*DefinitionStorage, error) {

	ds := &DefinitionStorage{
		ctx:   ctx,
		v3cli: v3cli,
	}

	return ds, nil
}

func (ds *DefinitionStorage) AddDefinition(ctx context.Context, definition *types.Definition) error {
	nextVersion := ds.definitionNextVersion(ctx, definition.Id)
	definition.Version = nextVersion

	data, err := proto.Marshal(definition)
	if err != nil {
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

	definition, err := types.DefinitionFromKV(resp.Kvs[0])
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (ds *DefinitionStorage) ListDefinitions(ctx context.Context) ([]*types.Definition, error) {

	key := api.DefinitionPrefix
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := ds.v3cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	definitions := make([]*types.Definition, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		definition, err := types.DefinitionFromKV(kv)
		if err != nil {
			continue
		}
		definitions = append(definitions, definition)
	}

	return definitions, nil
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

	return nil
}

func (ds *DefinitionStorage) ListProcess(ctx context.Context) ([]*types.Process, error) {

	key := api.ProcessPrefix
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	resp, err := ds.v3cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	processes := make([]*types.Process, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		process, err := types.ProcessFromKV(kv)
		if err != nil {
			continue
		}
		processes = append(processes, process)
	}

	return processes, nil
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

	return nil
}

func (ds *DefinitionStorage) RemoveProcess(ctx context.Context, process *types.Process) error {
	key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", process.Id))
	_, err := ds.v3cli.Delete(ctx, key)
	if err != nil {
		return err
	}

	return nil
}

func (ds *DefinitionStorage) isExistsDefinitionVersion(ctx context.Context, id int64, version uint64) bool {
	key := path.Join(api.DefinitionPrefix, fmt.Sprintf("%d", id), fmt.Sprintf("%d", version))
	resp, err := ds.v3cli.Get(ctx, key)
	if err != nil {
		return false
	}
	return len(resp.Kvs) > 0
}

func (ds *DefinitionStorage) definitionNextVersion(ctx context.Context, id int64) uint64 {

	var version uint64
	key := path.Join(definitionIds, fmt.Sprintf("%d", id))
	resp, err := ds.v3cli.Get(ctx, key)
	if err != nil || resp == nil || len(resp.Kvs) == 0 {
		version = 0
	} else {
		kv := resp.Kvs[0]
		version = binary.LittleEndian.Uint64(kv.Value[:8])
	}

	version += 1

	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(version))
	_, _ = ds.v3cli.Put(ctx, key, string(value))

	return version
}

func (ds *DefinitionStorage) getDefinitionLastVersion(ctx context.Context, id int64) uint64 {
	key := path.Join(definitionIds, fmt.Sprintf("%d", id))
	resp, err := ds.v3cli.Get(ctx, key)
	if err != nil || resp == nil || len(resp.Kvs) == 0 {
		return 0
	}
	kv := resp.Kvs[0]
	return binary.LittleEndian.Uint64(kv.Value[:8])
}
