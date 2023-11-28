// Copyright 2023 Lack (xingyys@gmail.com).
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
)

const (
	// maxLimit is a maximum page limit increase used when fetching objects from etcd.
	// This limit is used only for increasing page size by kube-apiserver. If request
	// specifies larger limit initially, it won't be changed.
	maxLimit = 10000
)

type definitionMeta struct {
	*pb.DefinitionMeta
	client *clientv3.Client
}

// Deploy deploys a new version definitions to storage (etcd)
func (dm *definitionMeta) Deploy(ctx context.Context, definition *pb.Definition) error {
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

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	if err = s.requestPrepare(ctx); err != nil {
		return
	}

	resp = &pb.DeployDefinitionResponse{}
	dm, err := s.definitionMeta(ctx, req.Id)
	if errors.Is(err, rpctypes.ErrGRPCKeyNotFound) {
		dm = &definitionMeta{client: dm.client}
		dm.Id = req.Id
	} else {
		return nil, err
	}

	if dm.Region == 0 {
		// select region
	}

	definition := &pb.Definition{
		Header:  &pb.OliveHeader{},
		Id:      req.Id,
		Name:    req.Name,
		Content: req.Content,
	}

	if err = dm.Deploy(ctx, definition); err != nil {
		return
	}

	resp.Version = dm.Version
	resp.Header = s.responseHeader()

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

			definition := &pb.Definition{}
			if err = definition.Unmarshal(rsp.Kvs[0].Value); err != nil {
				continue
			}

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
			// We got incomplete result due to field/label selector dropping the object.
			// Double page size to reduce total number of calls to etcd.
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
	if !s.notifier.IsLeader() {
		return
	}
	resp = &pb.GetDefinitionResponse{}

	version := req.Version
	if version == 0 {
		dm, err := s.definitionMeta(ctx, req.Id)
		if err != nil {
			return nil, err
		}
		version = dm.Version
	}

	options := []clientv3.OpOption{clientv3.WithSerializable()}
	key := path.Join(runtime.DefaultRunnerDefinitions, req.Id, fmt.Sprintf("%d", version))
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil || len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrGRPCKeyNotFound
	}
	definitions := &pb.Definition{}
	if err = definitions.Unmarshal(rsp.Kvs[0].Value); err != nil {
		return nil, err
	}
	resp.Header = s.responseHeader()
	resp.Definition = definitions

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
	if !s.notifier.IsLeader() {
		return
	}
	resp = &pb.ExecuteDefinitionResponse{}

	dm, err := s.definitionMeta(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	_ = dm

	return
}

func (s *Server) definitionMeta(ctx context.Context, id string) (*definitionMeta, error) {
	dm := &definitionMeta{
		DefinitionMeta: &pb.DefinitionMeta{},
		client:         s.v3cli,
	}

	key := path.Join(runtime.DefaultMetaDefinitionMeta, id)
	options := []clientv3.OpOption{clientv3.WithSerializable()}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrKeyNotFound
	}

	_ = dm.DefinitionMeta.Unmarshal(rsp.Kvs[0].Value)
	return dm, nil
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
