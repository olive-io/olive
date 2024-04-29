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

package meta

import (
	"context"
	"errors"
	"fmt"
	urlpkg "net/url"
	"path"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/meta/pagation"
	"github.com/olive-io/olive/pkg/runtime"
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
	key := path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", newVersion))
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
	key = path.Join(runtime.DefaultMetaDefinitionMeta, dm.Id)
	_, err = dm.client.Put(ctx, key, string(data))
	return err
}

func (s *Server) ListRunner(ctx context.Context, req *pb.ListRunnerRequest) (resp *pb.ListRunnerResponse, err error) {
	resp = &pb.ListRunnerResponse{}
	key := runtime.DefaultMetaRunnerRegistrar
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	runners := make([]*pb.Runner, 0, rsp.Count)
	for _, kv := range rsp.Kvs {
		runner := new(pb.Runner)
		if e1 := proto.Unmarshal(kv.Value, runner); e1 == nil {
			runners = append(runners, runner)
		}
	}
	resp.Header = s.responseHeader()
	resp.Runners = runners
	return resp, nil
}

func (s *Server) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (resp *pb.GetRunnerResponse, err error) {
	resp = &pb.GetRunnerResponse{}
	resp.Runner, err = s.getRunner(ctx, req.Id)
	resp.Header = s.responseHeader()
	return
}

func (s *Server) getRunner(ctx context.Context, id uint64) (runner *pb.Runner, err error) {
	key := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrKeyNotFound
	}
	runner = new(pb.Runner)
	_ = proto.Unmarshal(rsp.Kvs[0].Value, runner)
	return
}

func (s *Server) ListRegion(ctx context.Context, req *pb.ListRegionRequest) (resp *pb.ListRegionResponse, err error) {
	resp = &pb.ListRegionResponse{}
	key := runtime.DefaultRunnerRegion
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	regions := make([]*pb.Region, 0, rsp.Count)
	for _, kv := range rsp.Kvs {
		region := new(pb.Region)
		if e1 := proto.Unmarshal(kv.Value, region); e1 == nil {
			regions = append(regions, region)
		}
	}

	resp.Header = s.responseHeader()
	resp.Regions = regions
	return resp, nil
}

func (s *Server) GetRegion(ctx context.Context, req *pb.GetRegionRequest) (resp *pb.GetRegionResponse, err error) {
	resp = &pb.GetRegionResponse{}
	resp.Region, err = s.getRegion(ctx, req.Id)
	resp.Header = s.responseHeader()
	return
}

func (s *Server) getRegion(ctx context.Context, id uint64) (region *pb.Region, err error) {
	key := path.Join(runtime.DefaultRunnerRegion, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrKeyNotFound
	}
	region = new(pb.Region)
	_ = proto.Unmarshal(rsp.Kvs[0].Value, region)
	return
}

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	if err = s.reqPrepare(ctx); err != nil {
		return
	}

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

				key := path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
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

func (s *Server) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (resp *pb.ListDefinitionResponse, err error) {
	if err = s.reqPrepare(ctx); err != nil {
		return
	}

	resp = &pb.ListDefinitionResponse{}
	preparedKey := runtime.DefaultMetaDefinitionMeta
	keyPrefix := preparedKey
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	var paging bool
	// set the appropriate clientv3 options to filter the returned data set
	var limitOption *clientv3.OpOption
	limit := req.Limit
	if limit > 0 {
		paging = true
		options = append(options, clientv3.WithLimit(limit))
		limitOption = &options[len(options)-1]
	}

	var returnedRV, continueRV, withRev int64
	var continueKey string
	if len(req.Continue) > 0 {
		continueKey, continueRV, err = pagation.DecodeContinue(req.Continue, keyPrefix)
		if err != nil {
			return nil, fmt.Errorf("invalid continue token: %v", err)
		}

		rangeEnd := clientv3.GetPrefixRangeEnd(keyPrefix)
		options = append(options, clientv3.WithRange(rangeEnd))
		preparedKey = continueKey

		// If continueRV > 0, the LIST req needs a specific resource version.
		// continueRV==0 is invalid.
		// If continueRV < 0, the req is for the latest resource version.
		if continueRV > 0 {
			withRev = continueRV
			returnedRV = continueRV
		}
	}

	if withRev != 0 {
		options = append(options, clientv3.WithRev(withRev))
	}

	var lastKey []byte
	var hasMore bool
	v := make([]*pb.Definition, 0)
	for {
		getResp, err := s.v3cli.Get(ctx, preparedKey, options...)
		if err != nil {
			return nil, err
		}
		hasMore = getResp.More

		if len(getResp.Kvs) == 0 && getResp.More {
			return nil, fmt.Errorf("no results were found, but olive indicated there were more values remaining")
		}

		for _, kv := range getResp.Kvs {
			if paging && int64(len(v)) >= req.Limit {
				hasMore = true
				break
			}
			lastKey = kv.Key
			id := path.Base(string(lastKey))
			dm := &pb.DefinitionMeta{}
			if err = proto.Unmarshal(kv.Value, dm); err != nil {
				continue
			}
			version := dm.Version

			key := path.Join(runtime.DefaultRunnerDefinitions, id, fmt.Sprintf("%d", version))
			rsp, err := s.v3cli.Get(ctx, key, options...)
			if err != nil || len(rsp.Kvs) == 0 {
				continue
			}

			dkv := rsp.Kvs[0]
			definition := &pb.Definition{}
			if err = proto.Unmarshal(dkv.Value, definition); err != nil {
				continue
			}

			definition.Rev = dkv.ModRevision

			v = append(v, definition)
		}

		if !hasMore || paging {
			break
		}

		// indicate to the client which resource version was returned
		if returnedRV == 0 {
			returnedRV = getResp.Header.Revision
		}

		if int64(len(v)) >= req.Limit {
			break
		}

		if limit < maxLimit {
			limit *= 2
			if limit > maxLimit {
				limit = maxLimit
			}
			*limitOption = clientv3.WithLimit(limit)
		}
		preparedKey = string(lastKey) + "\x00"
		if withRev == 0 {
			withRev = returnedRV
			options = append(options, clientv3.WithRev(withRev))
		}
	}

	resp.Header = s.responseHeader()
	resp.Definitions = v
	if hasMore {
		// we want to start immediately after the last key
		next, err := pagation.EncodeContinue(string(lastKey)+"\x00", keyPrefix, returnedRV)
		if err != nil {
			return nil, err
		}
		resp.ContinueToken = next
	}

	return
}

func (s *Server) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	if err = s.reqPrepare(ctx); err != nil {
		return
	}
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
	key := path.Join(runtime.DefaultRunnerDefinitions, req.Id, fmt.Sprintf("%d", version))
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil || len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrGRPCKeyNotFound
	}
	definition := &pb.Definition{}
	kv := rsp.Kvs[0]
	if err = proto.Unmarshal(kv.Value, definition); err != nil {
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
				key = path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
				data, _ := proto.Marshal(definition)
				_, e1 := dm.client.Put(ctx, key, string(data))
				if e1 != nil {
					s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
				}
			}
		}()
	} else if dm.Region != definition.Region {
		definition.Region = dm.Region
		key = path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
		data, _ := proto.Marshal(definition)
		_, e1 := dm.client.Put(ctx, key, string(data))
		if e1 != nil {
			s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
		}
	}

	resp.Header = s.responseHeader()

	return
}

func (s *Server) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (resp *pb.RemoveDefinitionResponse, err error) {
	if !s.notifier.IsLeader() {
		return
	}
	resp = &pb.RemoveDefinitionResponse{}

	dm, err := s.definitionMeta(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	_ = dm

	return
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (resp *pb.ExecuteDefinitionResponse, err error) {
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

	key := path.Join(runtime.DefaultRunnerProcessInstance,
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

func (s *Server) ListProcessInstances(ctx context.Context, req *pb.ListProcessInstancesRequest) (resp *pb.ListProcessInstancesResponse, err error) {
	resp = &pb.ListProcessInstancesResponse{}

	lg := s.lg

	key := path.Join(runtime.DefaultRunnerProcessInstance, req.DefinitionId)
	if req.DefinitionVersion != 0 {
		key = path.Join(key, fmt.Sprintf("%d", req.DefinitionVersion))
	}
	if req.Id != "" {
		key = path.Join(key, req.Id)
	}
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	instances := make([]*pb.ProcessInstance, 0, len(rsp.Kvs))
	for _, kv := range rsp.Kvs {
		instance := new(pb.ProcessInstance)
		err = proto.Unmarshal(kv.Value, instance)
		if err != nil {
			lg.Error("unmarshal process instance", zap.String("key", string(kv.Key)), zap.Error(err))
			continue
		}
		instances = append(instances, instance)
	}
	resp.Header = s.responseHeader()
	resp.Instances = instances

	return
}

func (s *Server) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (resp *pb.GetProcessInstanceResponse, err error) {
	resp = &pb.GetProcessInstanceResponse{}

	lg := s.lg

	listResp, err := s.ListProcessInstances(ctx, &pb.ListProcessInstancesRequest{
		DefinitionId:      req.DefinitionId,
		DefinitionVersion: req.DefinitionVersion,
		Id:                req.Id,
	})
	if err != nil {
		return nil, err
	}
	if len(listResp.Instances) == 0 {
		return nil, rpctypes.ErrGRPCKeyNotFound
	}
	instance := listResp.Instances[0]

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

func (s *Server) definitionMeta(ctx context.Context, id string) (*definitionMeta, error) {
	dm := &definitionMeta{
		client: s.v3cli,
	}

	key := path.Join(runtime.DefaultMetaDefinitionMeta, id)
	options := []clientv3.OpOption{clientv3.WithSerializable()}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrGRPCKeyNotFound
	}

	_ = proto.Unmarshal(rsp.Kvs[0].Value, &dm.DefinitionMeta)
	return dm, nil
}

func (s *Server) bindDefinition(ctx context.Context, dm *pb.DefinitionMeta) (bool, error) {
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

func (s *Server) reqPrepare(ctx context.Context) error {
	if !s.notifier.IsLeader() {
		return rpctypes.ErrGRPCNotLeader
	}
	if s.etcd.Server.Leader() == 0 {
		return rpctypes.ErrGRPCNoLeader
	}

	return nil
}

func (s *Server) responseHeader() *pb.ResponseHeader {
	es := s.etcd.Server
	header := &pb.ResponseHeader{
		ClusterId: uint64(es.Cluster().ID()),
		MemberId:  uint64(es.ID()),
		RaftTerm:  es.Term(),
	}
	return header
}

func (s *Server) buildGRPCConn(ctx context.Context, targetURL string) (*grpc.ClientConn, error) {
	url, err := urlpkg.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	host := url.Host

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	return grpc.DialContext(ctx, host, options...)
}
