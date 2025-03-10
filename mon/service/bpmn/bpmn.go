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
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/mon/scheduler"
)

type Service struct {
	ctx context.Context

	lg *zap.Logger

	v3cli *clientv3.Client
	idGen *idutil.Generator

	ds *DefinitionStorage
}

func New(ctx context.Context, lg *zap.Logger, v3cli *clientv3.Client, idGen *idutil.Generator) (*Service, error) {

	ds, err := NewDefinitionStorage(ctx, v3cli)
	if err != nil {
		return nil, err
	}

	s := &Service{
		ctx:   ctx,
		lg:    lg,
		v3cli: v3cli,
		idGen: idGen,
		ds:    ds,
	}

	return s, nil
}

func (s *Service) DeployDefinition(ctx context.Context, definition *types.Definition) (*types.Definition, error) {
	if definition.Id == 0 {
		definition.Id = int64(s.idGen.Next())
	}
	definition.Timestamp = time.Now().Unix()

	var err error
	err = s.ds.AddDefinition(ctx, definition)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) ListDefinitions(ctx context.Context, page, size int32) ([]*types.Definition, int64, error) {
	if page <= 0 {
		page = 1
	}
	definitions, total, err := s.ds.ListDefinitions(ctx, page, size)
	if err != nil {
		return nil, total, err
	}

	return definitions, total, nil
}

func (s *Service) GetDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {

	definition, err := s.ds.GetDefinition(ctx, id, version)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) RemoveDefinition(ctx context.Context, id int64, version uint64) (*types.Definition, error) {
	definition, err := s.ds.GetDefinition(ctx, id, version)
	if err != nil {
		return nil, err
	}

	err = s.ds.RemoveDefinition(ctx, id, version)
	if err != nil {
		return nil, err
	}

	return definition, nil
}

func (s *Service) ExecuteDefinition(ctx context.Context, process *types.ProcessInstance) (*types.ProcessInstance, error) {

	process.Id = int64(s.idGen.Next())
	process.Priority = 1
	process.Status = types.ProcessStatus_Prepare

	data, err := proto.Marshal(process)
	if err != nil {
		return nil, err
	}
	key := path.Join(api.ProcessPrefix, fmt.Sprintf("%d", process.Id))
	_, err = s.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return nil, err
	}
	scheduler.AddProcess(ctx, process)

	return process, nil
}
