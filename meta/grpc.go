// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	defaultTimeout = time.Second * 15
)

type definitionMeta struct {
	pb.DefinitionMeta
	client *clientv3.Client
}

// Save saves a new version definitions to storage (etcd)
func (dm *definitionMeta) Save(ctx context.Context, definition *pb.Definition) error {
	newVersion := dm.Version + 1

	definition.Header.Region = dm.Region
	definition.Version = newVersion
	key := path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", newVersion))
	data, _ := definition.Marshal()
	rsp, err := dm.client.Put(ctx, key, string(data))
	if err != nil {
		return err
	}
	rev := rsp.Header.Revision
	definition.Header.Rev = rev
	if dm.StartRev == 0 {
		dm.StartRev = rev
	}

	dm.Version = newVersion
	dm.EndRev = rev

	data, _ = dm.Marshal()
	key = path.Join(runtime.DefaultMetaDefinitionMeta, dm.Id)
	_, err = dm.client.Put(ctx, key, string(data))
	return err
}

func (s *Server) GetMeta(ctx context.Context, req *pb.GetMetaRequest) (resp *pb.GetMetaResponse, err error) {
	cluster := s.etcd.Server.Cluster()
	meta := &pb.Meta{
		ClusterId: uint64(cluster.ID()),
		Leader:    uint64(s.etcd.Server.Leader()),
		Members:   make([]*pb.MetaMember, 0),
	}

	for _, member := range cluster.Members() {
		m := &pb.MetaMember{
			Id:         uint64(member.ID),
			ClientURLs: member.ClientURLs,
			PeerURLs:   member.PeerURLs,
		}
		meta.Members = append(meta.Members, m)
	}

	resp = &pb.GetMetaResponse{
		Meta: meta,
	}
	return resp, nil
}

func (s *Server) ListRunner(ctx context.Context, req *pb.ListRunnerRequest) (resp *pb.ListRunnerResponse, err error) {
	resp = &pb.ListRunnerResponse{}
	key := runtime.DefaultMetaRunnerRegistry
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
		if e1 := runner.Unmarshal(kv.Value); e1 == nil {
			runners = append(runners, runner)
		}
	}
	resp.Runners = runners
	return resp, nil
}

func (s *Server) GetRunner(ctx context.Context, req *pb.GetRunnerRequest) (resp *pb.GetRunnerResponse, err error) {
	resp = &pb.GetRunnerResponse{}
	resp.Runner, err = s.getRunner(ctx, req.Id)
	return
}

func (s *Server) getRunner(ctx context.Context, id uint64) (runner *pb.Runner, err error) {
	key := path.Join(runtime.DefaultMetaRunnerRegistry, fmt.Sprintf("%d", id))
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
	_ = runner.Unmarshal(rsp.Kvs[0].Value)
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
		if e1 := region.Unmarshal(kv.Value); e1 == nil {
			regions = append(regions, region)
		}
	}
	resp.Regions = regions
	return resp, nil
}

func (s *Server) GetRegion(ctx context.Context, req *pb.GetRegionRequest) (resp *pb.GetRegionResponse, err error) {
	resp = &pb.GetRegionResponse{}
	resp.Region, err = s.getRegion(ctx, req.Id)
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
	_ = region.Unmarshal(rsp.Kvs[0].Value)
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
		Header:  &pb.OliveHeader{},
		Id:      req.Id,
		Name:    req.Name,
		Content: req.Content,
	}

	if err = dm.Save(ctx, definition); err != nil {
		return
	}
	resp.Version = dm.Version
	resp.Header = s.responseHeader()

	if dm.Region == 0 {
		go func() {
			ok, _ := s.bindDefinition(ctx, &dm.DefinitionMeta)
			if ok {
				definition.Header.Region = dm.Region
				key := path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
				data, _ := definition.Marshal()
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
			id := path.Clean(string(lastKey))
			dm := &pb.DefinitionMeta{}
			if err = dm.Unmarshal(kv.Value); err != nil {
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
			if err = definition.Unmarshal(dkv.Value); err != nil {
				continue
			}
			definition.Header.Rev = dkv.ModRevision

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
	if err = definition.Unmarshal(kv.Value); err != nil {
		return nil, err
	}
	if definition.Header == nil {
		definition.Header = &pb.OliveHeader{}
	}
	definition.Header.Rev = kv.ModRevision
	resp.Header = s.responseHeader()
	resp.Definition = definition

	if dm.Region == 0 {
		go func() {
			ok, _ := s.bindDefinition(ctx, &dm.DefinitionMeta)
			if ok {
				definition.Header.Region = dm.Region
				key = path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
				data, _ := definition.Marshal()
				_, e1 := dm.client.Put(ctx, key, string(data))
				if e1 != nil {
					s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
				}
			}
		}()
	} else if dm.Region != definition.Header.Region {
		definition.Header.Region = dm.Region
		key = path.Join(runtime.DefaultRunnerDefinitions, definition.Id, fmt.Sprintf("%d", definition.Version))
		data, _ := definition.Marshal()
		_, e1 := dm.client.Put(ctx, key, string(data))
		if e1 != nil {
			s.lg.Error("update definition", zap.String("id", definition.Id), zap.Error(e1))
		}
	}

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

	if definition.Header.Region == 0 {
		return nil, rpctypes.ErrGRPCDefinitionNotReady
	}

	instance := &pb.ProcessInstance{
		OliveHeader:       &pb.OliveHeader{Region: definition.Header.Region},
		Id:                s.idReq.Next(),
		Name:              req.Name,
		DefinitionId:      definition.Id,
		DefinitionVersion: definition.Version,
		Headers:           req.Header,
		Properties:        req.Properties,
		RunningState:      &pb.ProcessRunningState{},
		FlowNodes:         make(map[string]*pb.FlowNodeStat),
		Status:            pb.ProcessInstance_Waiting,
	}
	key := path.Join(runtime.DefaultRunnerProcessInstance,
		definition.Id, fmt.Sprintf("%d", definition.Version), fmt.Sprintf("%d", instance.Id))
	data, _ := instance.Marshal()
	rsp, err := s.v3cli.Put(ctx, key, string(data))
	if err != nil {
		return nil, err
	}
	instance.OliveHeader.Rev = rsp.Header.Revision
	resp.Header = s.responseHeader()
	resp.Instance = instance

	return
}

func (s *Server) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (resp *pb.GetProcessInstanceResponse, err error) {
	resp = &pb.GetProcessInstanceResponse{}

	lg := s.lg

	key := path.Join(runtime.DefaultRunnerProcessInstance,
		req.DefinitionId, fmt.Sprintf("%d", req.DefinitionVersion), fmt.Sprintf("%d", req.Id))
	options := []clientv3.OpOption{clientv3.WithSerializable()}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrGRPCKeyNotFound
	}
	instance := new(pb.ProcessInstance)
	err = instance.Unmarshal(rsp.Kvs[0].Value)
	if err != nil {
		return nil, err
	}
	if instance.OliveHeader == nil {
		return
	}

	header := instance.OliveHeader
	if header.Region != 0 {
		region, _ := s.getRegion(ctx, header.Region)
		if region == nil {
			return
		}

		replica, ok := region.Replicas[region.Leader]
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

	_ = dm.DefinitionMeta.Unmarshal(rsp.Kvs[0].Value)
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
