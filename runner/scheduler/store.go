/*
Copyright 2024 The olive Authors

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

package scheduler

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/runner/storage"
)

func (s *Scheduler) ListDefinition(ctx context.Context, id int64) ([]*types.Definition, error) {

	definitions := make([]*types.Definition, 0)
	key := api.DefinitionPrefix
	if id != 0 {
		key = path.Join(key, strconv.FormatInt(id, 10))
	}

	opts := []storage.GetOption{
		storage.GetPrefix(),
	}

	resp, err := s.bs.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}
	err = storage.NewUnmarshaler[*types.Definition](resp).SliceTo(&definitions)
	if err != nil {
		return nil, err
	}

	return definitions, nil
}

func (s *Scheduler) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {

	definition := new(types.Definition)
	key := path.Join(api.DefinitionPrefix, strconv.FormatInt(id, 10))
	if version != 0 {
		key = path.Join(key, fmt.Sprintf("%d", version))
	}
	resp, err := s.bs.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if err = resp.Unmarshal(definition); err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Scheduler) saveDefinition(ctx context.Context, definition *types.Definition) error {
	if definition.Version != 0 {
		definition.Version = 1
	}

	key := path.Join(api.DefinitionPrefix,
		strconv.FormatInt(definition.Id, 10),
		fmt.Sprintf("%d", definition.Version))
	err := s.bs.Put(ctx, key, definition)
	if err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) undoneTasks(ctx context.Context) []*ProcessItem {
	items := make([]*ProcessItem, 0)

	key := api.ProcessPrefix
	opts := []storage.GetOption{
		storage.GetPrefix(),
	}
	resp, err := s.bs.Get(ctx, key, opts...)
	if err != nil {
		return items
	}

	for _, kv := range resp.Kvs {
		var instance types.ProcessInstance
		if err := json.Unmarshal(kv.Value, &instance); err != nil {
			continue
		}

		if instance.Finished() {
			continue
		}
		items = append(items, &ProcessItem{&instance})
	}

	return items
}

func (s *Scheduler) ListProcess(ctx context.Context, definition int64, version uint64) ([]*types.ProcessInstance, error) {

	instances := make([]*types.ProcessInstance, 0)
	key := api.ProcessPrefix
	if definition != 0 {
		key = path.Join(key, fmt.Sprintf("%d", definition))
		if version != 0 {
			key = path.Join(key, fmt.Sprintf("%d", version))
		}
	}
	opts := []storage.GetOption{
		storage.GetPrefix(),
	}
	resp, err := s.bs.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}
	err = storage.NewUnmarshaler[*types.ProcessInstance](resp).SliceTo(&instances)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

func (s *Scheduler) GetProcess(ctx context.Context, definition int64, version uint64, id int64) (*types.ProcessInstance, error) {
	identify := fmt.Sprintf("%d/%d/%d", definition, version, id)

	key := api.ProcessPrefix
	key = path.Join(key, identify)
	resp, err := s.bs.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	pi := new(types.ProcessInstance)
	err = resp.Unmarshal(pi)
	if err != nil {
		return nil, err
	}

	return pi, nil
}

func (s *Scheduler) saveProcess(ctx context.Context, instance *types.ProcessInstance) error {
	if instance.StartAt == 0 {
		instance.StartAt = time.Now().UnixNano()
	}

	if instance.Id == 0 {
		instance.Id = int64(s.idGen.Next())
	}

	identify := fmt.Sprintf("%d/%d/%d", instance.DefinitionsId, instance.DefinitionsVersion, instance.Id)
	key := path.Join(api.ProcessPrefix, identify)

	if err := s.bs.Put(ctx, key, instance); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) finishProcess(ctx context.Context, instance *types.ProcessInstance, err error) error {
	instance.EndAt = time.Now().UnixNano()
	if err != nil {
		instance.Status = types.ProcessStatus_Failed
		instance.Message = err.Error()
	} else {
		instance.Status = types.ProcessStatus_Ok
	}
	return s.saveProcess(ctx, instance)
}
