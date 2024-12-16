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
	"errors"
	"fmt"
	"path"
	"time"

	json "github.com/bytedance/sonic"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
	"github.com/olive-io/olive/runner/storage/backend"
)

const (
	definitionPrefix  = "/de"
	processPrefix     = "/pr"
	processMetaPrefix = "/pm"
)

func (s *Scheduler) ListDefinition(ctx context.Context, id string) ([]*corev1.Definition, error) {

	key := path.Join(definitionPrefix, id)
	resp, err := s.db.Get(ctx, key, backend.GetPrefix())
	if err != nil {
		return nil, err
	}

	ds := make([]*corev1.Definition, 0)
	for _, kv := range resp.Kvs {
		d := &corev1.Definition{}
		if err = json.Unmarshal(kv.Value, d); err != nil {
			return nil, err
		}
		ds = append(ds, d)
	}

	return ds, nil
}

func (s *Scheduler) GetDefinition(ctx context.Context, id string, version int64) (*corev1.Definition, error) {

	key := path.Join(definitionPrefix, fmt.Sprintf("%s/%010d", id, version))
	resp, err := s.db.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	definition := corev1.Definition{}
	if err = resp.Unmarshal(&definition); err != nil {
		return nil, err
	}

	return &definition, nil
}

func (s *Scheduler) saveDefinition(ctx context.Context, id, name string, version int64, content string) error {
	definition := &corev1.Definition{
		ObjectMeta: metav1.ObjectMeta{
			UID:  id,
			Name: name,
		},
		Content: content,
		Version: version,
	}

	key := path.Join(definitionPrefix, fmt.Sprintf("%s/%010d", id, version))
	_, err := s.db.Get(ctx, key)
	if err == nil {
		return nil
	}

	if !errors.Is(err, backend.ErrNotFound) {
		return err
	}

	if err = s.db.Put(ctx, key, definition); err != nil {
		return err
	}

	return nil
}

type ProcessMeta struct {
	DefinitionsId      string `json:"id"`
	DefinitionsVersion int64  `json:"version"`

	Instance string               `json:"instance"`
	Status   corev1.ProcessStatus `json:"status"`
}

func (pm *ProcessMeta) finished() bool {
	switch pm.Status {
	case corev1.ProcessOk,
		corev1.ProcessFail:
		return true
	default:
		return false
	}
}

func (s *Scheduler) undoneTasks(ctx context.Context) []*ProcessItem {
	items := make([]*ProcessItem, 0)

	metaKey := path.Join(processMetaPrefix)
	resp, err := s.db.Get(ctx, metaKey, backend.GetPrefix())
	if err != nil {
		return items
	}
	for _, kv := range resp.Kvs {
		pm := ProcessMeta{}
		if err = json.Unmarshal(kv.Value, &pm); err != nil {
			continue
		}

		if pm.finished() {
			continue
		}

		identify := fmt.Sprintf("%s/%d/%s", pm.DefinitionsId, pm.DefinitionsVersion, pm.Instance)
		instanceKey := path.Join(processPrefix, identify)

		resp1, err := s.db.Get(ctx, instanceKey)
		if err != nil {
			continue
		}
		instance := corev1.ProcessInstance{}
		if err = resp1.Unmarshal(&instance); err != nil {
			continue
		}

		switch instance.Status {
		case corev1.ProcessOk,
			corev1.ProcessFail:
			continue
		default:
		}
		items = append(items, &ProcessItem{&instance})
	}

	return items
}

func (s *Scheduler) ListProcess(ctx context.Context, definition string, version int64) ([]*corev1.ProcessInstance, error) {

	prefix := path.Join(processMetaPrefix)
	if definition != "" {
		prefix = path.Join(prefix, definition)
	}
	if version != 0 {
		prefix = path.Join(prefix, fmt.Sprintf("%d", version))
	}

	resp, err := s.db.Get(ctx, prefix, backend.GetPrefix())
	if err != nil {
		return nil, err
	}

	instances := make([]*corev1.ProcessInstance, 0)
	for _, kv := range resp.Kvs {
		pm := ProcessMeta{}
		if err = json.Unmarshal(kv.Value, &pm); err != nil {
			continue
		}

		identify := fmt.Sprintf("%s/%d/%s", pm.DefinitionsId, pm.DefinitionsVersion, pm.Instance)
		instanceKey := path.Join(processPrefix, identify)

		resp1, err := s.db.Get(ctx, instanceKey)
		if err != nil {
			continue
		}
		instance := corev1.ProcessInstance{}
		if err = resp1.Unmarshal(&instance); err != nil {
			continue
		}

		instances = append(instances, &instance)
	}

	return instances, nil
}

func (s *Scheduler) GetProcess(ctx context.Context, definition string, version int64, id string) (*corev1.ProcessInstance, error) {
	identify := fmt.Sprintf("%s/%d/%s", definition, version, id)

	instanceKey := path.Join(processPrefix, identify)
	resp, err := s.db.Get(ctx, instanceKey)
	if err != nil {
		return nil, err
	}

	instance := corev1.ProcessInstance{}
	if err = resp.Unmarshal(&instance); err != nil {
		return nil, err
	}

	return &instance, nil
}

func (s *Scheduler) saveProcess(ctx context.Context, instance *corev1.ProcessInstance) error {
	if instance.CreationTimestamp == 0 {
		instance.CreationTimestamp = time.Now().UnixNano()
	}

	pm := &ProcessMeta{
		DefinitionsId:      instance.DefinitionsId,
		DefinitionsVersion: instance.DefinitionsVersion,
		Instance:           instance.UID,
		Status:             instance.Status,
	}

	pmKey := fmt.Sprintf("%s/%d/%020d", instance.DefinitionsId, instance.DefinitionsVersion, instance.CreationTimestamp)
	metaKey := path.Join(processMetaPrefix, pmKey)

	if err := s.db.Put(ctx, metaKey, pm); err != nil {
		return err
	}

	identify := fmt.Sprintf("%s/%d/%s", instance.DefinitionsId, instance.DefinitionsVersion, instance.UID)
	instanceKey := path.Join(processPrefix, identify)
	if err := s.db.Put(ctx, instanceKey, instance); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) finishProcess(ctx context.Context, instance *corev1.ProcessInstance, err error) error {
	instance.EndTimestamp = time.Now().UnixNano()
	if err != nil {
		instance.Status = corev1.ProcessFail
		instance.Message = err.Error()
	} else {
		instance.Status = corev1.ProcessOk
	}
	return s.saveProcess(ctx, instance)
}
