/*
   Copyright 2023 The olive Authors

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

package mon

import (
	"context"
	"errors"
	"fmt"
	"path"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/apis/rpctypes"

	ort "github.com/olive-io/olive/pkg/runtime"
)

const (
	// maxLimit is a maximum page limit increase used when fetching objects from etcd.
	// This limit is used only for increasing page size by olive. If req
	// specifies larger limit initially, it won't be changed.
	maxLimit = 10000

	defaultTimeout = time.Second * 30
)

type definitionMeta struct {
	pb.DefinitionMeta
	client *clientv3.Client
}

// Save saves a new version definitions to storage (etcd)
func (dm *definitionMeta) Save(ctx context.Context, definition *pb.Definition) error {
	newVersion := dm.Version + 1

	definition.Region = dm.Region
	definition.Version = newVersion
	key := path.Join(ort.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", newVersion))
	data, _ := proto.Marshal(definition)
	rsp, err := dm.client.Put(ctx, key, string(data))
	if err != nil {
		return err
	}
	rev := rsp.Header.Revision

	if dm.StartRev == 0 {
		dm.StartRev = rev
	}

	dm.Version = newVersion
	dm.EndRev = rev

	definition.Rev = rev

	data, _ = proto.Marshal(dm)
	key = path.Join(ort.DefaultMetaDefinitionMeta, dm.Id)
	_, err = dm.client.Put(ctx, key, string(data))
	return err
}

type bpmnServer struct {
	pb.UnsafeBpmnRPCServer

	*Server
}

func newBpmnServer(s *Server) (*bpmnServer, error) {
	bs := &bpmnServer{Server: s}
	return bs, nil
}

func (s *bpmnServer) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {

	resp = &pb.DeployDefinitionResponse{}
	dm, err := s.definitionMeta(ctx, req.Id)
	if err != nil {
		if !errors.Is(err, rpctypes.ErrGRPCKeyNotFound) {
			return nil, err
		}
		dm = &definitionMeta{client: s.v3cli}
		dm.Id = req.Id
	}

	definition := &pb.Definition{
		Id:      req.Id,
		Name:    req.Name,
		Content: string(req.Content),
	}

	if err = dm.Save(ctx, definition); err != nil {
		return
	}
	resp.Header = s.responseHeader()
	resp.Definition = definition

	if dm.Region == 0 {
		go func() {
			ok, _ := s.bindDefinition(ctx, &dm.DefinitionMeta)
			if ok {
				definition.Region = dm.Region

				key := path.Join(ort.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
				data, _ := proto.Marshal(definition)
				_, e1 := dm.client.Put(ctx, key, string(data))
				if e1 != nil {
					s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
				}
			}
		}()
	}

	return
}

func (s *bpmnServer) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (resp *pb.ListDefinitionResponse, err error) {

	resp = &pb.ListDefinitionResponse{}
	preparedKey := ort.DefaultMetaDefinitionMeta

	options := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}

	var lastKey []byte
	v := make([]*pb.Definition, 0)
	var continueToken string
	continueToken, err = s.pageList(ctx, preparedKey, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
		lastKey = kv.Key
		id := path.Base(string(lastKey))
		dm := &pb.DefinitionMeta{}
		if e1 := proto.Unmarshal(kv.Value, dm); e1 != nil {
			return e1
		}
		version := dm.Version

		key := path.Join(ort.DefaultRunnerDefinitions, id, fmt.Sprintf("%d", version))
		definition := &pb.Definition{}

		_, kv, err := s.get(ctx, key, definition, options...)
		if err != nil {
			return err
		}
		definition.Rev = kv.ModRevision

		v = append(v, definition)

		return nil
	})
	if err != nil {
		return
	}

	resp.Header = s.responseHeader()
	resp.Definitions = v
	resp.ContinueToken = continueToken

	return
}

func (s *bpmnServer) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {

	resp = &pb.GetDefinitionResponse{}

	version := req.Version
	dm, err := s.definitionMeta(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if version == 0 {
		version = dm.Version
	}

	options := []clientv3.OpOption{clientv3.WithSerializable()}
	key := path.Join(ort.DefaultRunnerDefinitions, req.Id, fmt.Sprintf("%d", version))
	definition := &pb.Definition{}
	_, kv, err := s.get(ctx, key, definition, options...)
	if err != nil {
		return nil, err
	}

	definition.Rev = kv.ModRevision
	resp.Header = s.responseHeader()
	resp.Definition = definition

	if dm.Region == 0 {
		go func() {
			ok, _ := s.bindDefinition(ctx, &dm.DefinitionMeta)
			if ok {
				definition.Region = dm.Region
				key = path.Join(ort.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
				data, _ := proto.Marshal(definition)
				_, e1 := dm.client.Put(ctx, key, string(data))
				if e1 != nil {
					s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
				}
			}
		}()
	} else if dm.Region != definition.Region {
		definition.Region = dm.Region
		key = path.Join(ort.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
		data, _ := proto.Marshal(definition)
		_, e1 := dm.client.Put(ctx, key, string(data))
		if e1 != nil {
			s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
		}
	}

	resp.Header = s.responseHeader()

	return
}

func (s *bpmnServer) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (resp *pb.RemoveDefinitionResponse, err error) {

	resp = &pb.RemoveDefinitionResponse{}

	dm, err := s.definitionMeta(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	_ = dm

	return
}

func (s *bpmnServer) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (resp *pb.ExecuteDefinitionResponse, err error) {

	resp = &pb.ExecuteDefinitionResponse{}

	in := &pb.GetDefinitionRequest{
		Id:      req.DefinitionId,
		Version: req.DefinitionVersion,
	}
	out, err := s.GetDefinition(ctx, in)
	if err != nil {
		return nil, err
	}
	definition := out.Definition

	if definition.Region == 0 {
		return nil, rpctypes.ErrGRPCDefinitionNotReady
	}

	id := fmt.Sprintf("%d", s.idGen.Next())
	instance := &pb.ProcessInstance{
		Id:                 id,
		Name:               req.Name,
		DefinitionsId:      definition.Id,
		DefinitionsVersion: definition.Version,
		Headers:            req.Header,
		Properties:         req.Properties,
		RunningState:       &pb.ProcessRunningState{},
		FlowNodes:          make(map[string]*pb.FlowNodeStat),
		Status:             pb.ProcessInstance_Waiting,
		CreationTime:       time.Now().UnixNano(),
	}
	instance.Region = definition.Region

	key := path.Join(ort.DefaultRunnerProcessInstance,
		definition.Id, fmt.Sprintf("%d", definition.Version), instance.Id)
	data, _ := proto.Marshal(instance)
	rsp, err := s.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return nil, err
	}

	instance.Rev = rsp.Header.Revision
	resp.Header = s.responseHeader()
	resp.Instance = instance

	return
}

func (s *bpmnServer) ListProcessInstances(ctx context.Context, req *pb.ListProcessInstancesRequest) (resp *pb.ListProcessInstancesResponse, err error) {

	lg := s.lg
	resp = &pb.ListProcessInstancesResponse{}
	key := path.Join(ort.DefaultRunnerProcessInstance, req.DefinitionId)
	if req.DefinitionVersion != 0 {
		key = path.Join(key, fmt.Sprintf("%d", req.DefinitionVersion))
	}

	instances := make([]*pb.ProcessInstance, 0)
	var continueToken string
	continueToken, err = s.pageList(ctx, key, req.Limit, req.Continue, func(kv *mvccpb.KeyValue) error {
		instance := new(pb.ProcessInstance)
		if e1 := proto.Unmarshal(kv.Value, instance); e1 != nil {
			lg.Error("unmarshal process instance", zap.String("key", string(kv.Key)), zap.Error(err))
			return e1
		}
		instances = append(instances, instance)

		return nil
	})
	if err != nil {
		return
	}

	resp.Header = s.responseHeader()
	resp.Instances = instances
	resp.ContinueToken = continueToken

	return
}

func (s *bpmnServer) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (resp *pb.GetProcessInstanceResponse, err error) {

	resp = &pb.GetProcessInstanceResponse{}

	lg := s.lg

	options := []clientv3.OpOption{clientv3.WithSerializable()}

	key := path.Join(ort.DefaultRunnerProcessInstance, req.DefinitionId,
		fmt.Sprintf("%d", req.DefinitionVersion),
		req.Id)

	instance := new(pb.ProcessInstance)
	_, _, err = s.get(ctx, key, instance, options...)
	if err != nil {
		return nil, err
	}

	if instance.Region != 0 {
		region, _ := s.getRegion(ctx, instance.Region)
		if region == nil {
			return
		}

		replica, ok := region.GetLeaderMember()
		if !ok {
			if len(region.Replicas) == 0 {
				return
			}

			for id := range region.Replicas {
				replica = region.Replicas[id]
				break
			}
		}

		runner, _ := s.getRunner(ctx, replica.Runner)
		if runner == nil {
			return
		}

		var conn *grpc.ClientConn
		conn, err = s.buildGRPCConn(ctx, runner.ListenClientURL)
		if err != nil {
			lg.Error("build grpc connection",
				zap.String("target", runner.ListenClientURL),
				zap.Error(err))
			return
		}

		req.Region = region.Id
		runnerRsp, err := pb.NewRunnerRPCClient(conn).GetProcessInstance(ctx, req)
		if err != nil {
			return nil, err
		}
		resp.Instance = runnerRsp.Instance
	}
	resp.Header = s.responseHeader()

	return
}

func (s *bpmnServer) definitionMeta(ctx context.Context, id string) (*definitionMeta, error) {
	dm := &definitionMeta{
		client: s.v3cli,
	}

	key := path.Join(ort.DefaultMetaDefinitionMeta, id)
	options := []clientv3.OpOption{clientv3.WithSerializable()}
	_, _, err := s.get(ctx, key, &dm.DefinitionMeta, options...)
	if err != nil {
		return nil, err
	}
	return dm, nil
}

func (s *bpmnServer) bindDefinition(ctx context.Context, dm *pb.DefinitionMeta) (bool, error) {
	_, ok, err := s.scheduler.BindRegion(ctx, dm)
	if err != nil {
		s.lg.Error("binding region",
			zap.String("definition", dm.Id),
			zap.Error(err))
	}

	if ok && dm.Region > 0 {
		s.lg.Info("binding definition",
			zap.String("definition", dm.Id),
			zap.Uint64("region", dm.Region))
	}

	return ok, err
}
