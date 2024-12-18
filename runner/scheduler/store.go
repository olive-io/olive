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
	"time"

	"github.com/google/uuid"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
)

func (s *Scheduler) ListDefinition(ctx context.Context, id string) (*corev1.DefinitionList, error) {

	list := new(corev1.DefinitionList)
	key := s.bs.GenerateKey(list)
	if id != "" {
		key = path.Join(key, id)
	}

	if err := s.bs.GetList(ctx, key, list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Scheduler) GetDefinition(ctx context.Context, id string, version int64) (*corev1.Definition, error) {

	definition := new(corev1.Definition)
	key := s.bs.GenerateKey(definition)
	key = path.Join(key, fmt.Sprintf("%s/%d", id, version))
	err := s.bs.Get(ctx, key, definition)
	if err != nil {
		return nil, err
	}

	return definition, nil
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

	key := s.bs.GenerateKey(definition)
	s.bs.Default(definition)

	if err := s.bs.Create(ctx, key, definition, 0); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) undoneTasks(ctx context.Context) []*ProcessItem {
	items := make([]*ProcessItem, 0)

	list := new(corev1.ProcessInstanceList)
	key := s.bs.GenerateKey(list)
	if err := s.bs.GetList(ctx, key, list); err != nil {
		return items
	}

	for _, item := range list.Items {
		if processFinished(item.Status) {
			continue
		}
		items = append(items, &ProcessItem{&item})
	}

	return items
}

func (s *Scheduler) ListProcess(ctx context.Context, definition string, version int64) (*corev1.ProcessInstanceList, error) {

	list := new(corev1.ProcessInstanceList)
	key := s.bs.GenerateKey(list)
	if definition != "" {
		key = path.Join(key, definition)
		if version != 0 {
			key = path.Join(key, fmt.Sprintf("%d", version))
		}
	}
	if err := s.bs.GetList(ctx, key, list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *Scheduler) GetProcess(ctx context.Context, definition string, version int64, id string) (*corev1.ProcessInstance, error) {
	identify := fmt.Sprintf("%s/%d/%s", definition, version, id)

	instance := new(corev1.ProcessInstance)
	key := s.bs.GenerateKey(instance)
	key = path.Join(key, identify)
	if err := s.bs.Get(ctx, key, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *Scheduler) saveProcess(ctx context.Context, instance *corev1.ProcessInstance) error {
	if instance.CreationTimestamp == 0 {
		instance.CreationTimestamp = time.Now().UnixNano()
	}

	if instance.UID == "" {
		instance.SetUID(uuid.New().String())
	}

	key := s.bs.GenerateKey(instance)
	identify := fmt.Sprintf("%s/%d/%s", instance.DefinitionsId, instance.DefinitionsVersion, instance.UID)
	key = path.Join(key, identify)

	if err := s.bs.Create(ctx, key, instance, 0); err != nil {
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
