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
	"path"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/meta/pagation"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

const (
	// maxLimit is a maximum page limit increase used when fetching objects from etcd.
	// This limit is used only for increasing page size by olive. If request
	// specifies larger limit initially, it won't be changed.
	maxLimit = 10000
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

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	if err = s.requestPrepare(ctx); err != nil {
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
	if err = s.requestPrepare(ctx); err != nil {
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

		// If continueRV > 0, the LIST request needs a specific resource version.
		// continueRV==0 is invalid.
		// If continueRV < 0, the request is for the latest resource version.
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
	if err = s.requestPrepare(ctx); err != nil {
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
		Header:            &pb.OliveHeader{Region: definition.Header.Region},
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
	instance.Header.Rev = rsp.Header.Revision
	resp.Header = s.responseHeader()
	resp.Instance = instance

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

func (s *Server) requestPrepare(ctx context.Context) error {
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
