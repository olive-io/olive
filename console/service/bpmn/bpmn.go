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

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/console/dao"
	"github.com/olive-io/olive/console/model"
)

type Service struct {
	ctx context.Context
	cfg *config.Config
	oct *client.Client

	definitionDao *dao.DefinitionDao
	processDao    *dao.ProcessDao
	watchDao      *dao.WatchDao
}

func NewBpmn(ctx context.Context, cfg *config.Config, oct *client.Client) (*Service, error) {

	definitionDao := dao.NewDefinition()
	processDao := dao.NewProcess()
	watchDao := dao.NewWatch()

	s := &Service{
		ctx: ctx,
		cfg: cfg,
		oct: oct,

		definitionDao: definitionDao,
		processDao:    processDao,
		watchDao:      watchDao,
	}

	go s.process()
	return s, nil
}

func (s *Service) ListDefinitions(ctx context.Context, page, size int32) (*model.ListResult[types.Definition], error) {
	mr := model.NewListResult[model.Definition](page, size)
	if err := s.definitionDao.ListDefinitions(ctx, mr); err != nil {
		return nil, err
	}

	result := model.NewListResult[types.Definition](page, size)
	result.Total = mr.Total

	for _, item := range mr.List {
		definition, err := s.oct.GetDefinition(ctx, item.DefinitionID, item.Version)
		if err != nil {
			continue
		}
		result.List = append(result.List, definition)
	}

	return result, nil
}

func (s *Service) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	md, err := s.definitionDao.GetDefinition(ctx, id, version)
	if err != nil {
		return nil, err
	}

	definition, err := s.oct.GetDefinition(ctx, md.DefinitionID, md.Version)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) DeployDefinition(ctx context.Context, definition *types.Definition) (*types.Definition, error) {
	var err error

	definition, err = s.oct.DeployDefinition(ctx, definition)
	if err != nil {
		return nil, err
	}

	err = s.definitionDao.AddDefinition(ctx, definition)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) DeleteDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	definition, err := s.GetDefinition(ctx, id, version)
	if err != nil {
		return nil, err
	}

	if err := s.oct.RemoveDefinition(ctx, id, version); err != nil {
		return nil, err
	}

	if err := s.definitionDao.DeleteDefinition(ctx, id, version); err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) ListProcesses(ctx context.Context, page, size int32, definitionId int64, version uint64, status types.ProcessStatus) (*model.ListResult[types.Process], error) {
	mr := model.NewListResult[model.Process](page, size)
	if err := s.processDao.ListProcess(ctx, definitionId, version, status, mr); err != nil {
		return nil, err
	}

	result := model.NewListResult[types.Process](page, size)
	result.Total = mr.Total

	for _, item := range mr.List {
		process, err := s.oct.GetProcess(ctx, item.ProcessID)
		if err != nil {
			continue
		}
		result.List = append(result.List, process)
	}

	return result, nil
}

func (s *Service) GetProcess(ctx context.Context, id int64) (*types.Process, error) {
	mp, err := s.processDao.GetProcess(ctx, id)
	if err != nil {
		return nil, err
	}

	process, err := s.oct.GetProcess(ctx, mp.ProcessID)
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (s *Service) DeleteProcess(ctx context.Context, id int64) (*types.Process, error) {
	process, err := s.GetProcess(ctx, id)
	if err != nil {
		return nil, err
	}

	process, err = s.oct.RemoveProcess(ctx, process.Id)
	if err != nil {
		return nil, err
	}

	err = s.processDao.DeleteProcess(ctx, id)
	if err != nil {
		return nil, err
	}

	return process, nil
}

func (s *Service) process() {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	rev := s.watchDao.GetRev(ctx)

	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithMinModRev(rev + 1),
		clientv3.WithSerializable(),
	}

	currentRev := int64(0)
	resp, _ := s.oct.Get(ctx, api.DefinitionPrefix, opts...)
	if resp != nil {
		for _, kv := range resp.Kvs {
			definition, err := types.DefinitionFromKV(kv)
			if err != nil {
				continue
			}
			_ = s.definitionDao.AddDefinition(ctx, definition)
		}
	}

	resp, _ = s.oct.Get(ctx, api.ProcessPrefix, opts...)
	if resp != nil {
		for _, kv := range resp.Kvs {
			process, err := types.ProcessFromKV(kv)
			if err != nil {
				continue
			}
			currentRev = kv.ModRevision
			_ = s.processDao.AddProcess(ctx, process)
		}
	}

	_ = s.watchDao.SetRev(ctx, currentRev)

	for {

		rev = s.watchDao.GetRev(ctx)
		wopts := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithPrevKV(),
		}
		if rev != 0 {
			wopts = append(wopts, clientv3.WithRev(rev+1))
		}

		dw := s.oct.Watch(ctx, api.DefinitionPrefix, wopts...)

		pw := s.oct.Watch(ctx, api.ProcessPrefix, wopts...)

	LOOP:
		for {
			select {
			case <-ctx.Done():
				return
			case wch, ok := <-dw:
				if !ok {
					break LOOP
				}
				if werr := wch.Err(); werr != nil {
					break LOOP
				}

				for _, event := range wch.Events {
					if modRev := event.Kv.ModRevision; modRev > rev {
						s.watchDao.SetRev(ctx, modRev)
					}

					switch {
					case event.Type == mvccpb.PUT:
						kv := event.Kv
						_ = s.watchDao.SetRev(ctx, kv.ModRevision)
						definition, e1 := types.DefinitionFromKV(kv)
						if e1 != nil {
							continue
						}

						if event.IsCreate() {
							_ = s.definitionDao.AddDefinition(ctx, definition)
						} else {
							_ = s.definitionDao.UpdateDefinition(ctx, definition)
						}
					case event.Type == mvccpb.DELETE:
						kv := event.PrevKv
						_ = s.watchDao.SetRev(ctx, kv.ModRevision)
						definition, e1 := types.DefinitionFromKV(kv)
						if e1 != nil {
							continue
						}
						_ = s.definitionDao.DeleteDefinition(ctx, definition.Id, definition.Version)
					}
				}

			case wch, ok := <-pw:
				if !ok {
					break LOOP
				}
				if werr := wch.Err(); werr != nil {
					break LOOP
				}

				for _, event := range wch.Events {
					if modRev := event.Kv.ModRevision; modRev > rev {
						s.watchDao.SetRev(ctx, modRev)
					}

					switch {
					case event.Type == mvccpb.PUT:
						kv := event.Kv
						_ = s.watchDao.SetRev(ctx, kv.ModRevision)
						process, e1 := types.ProcessFromKV(kv)
						if e1 != nil {
							continue
						}

						if event.IsCreate() {
							_ = s.processDao.AddProcess(ctx, process)
						} else {
							_ = s.processDao.UpdateProcess(ctx, process)
						}

					case event.Type == mvccpb.DELETE:
						kv := event.PrevKv
						_ = s.watchDao.SetRev(ctx, kv.ModRevision)
						process, e1 := types.ProcessFromKV(kv)
						if e1 != nil {
							continue
						}
						_ = s.processDao.DeleteProcess(ctx, process.Id)
					}
				}
			}
		}
	}
}
