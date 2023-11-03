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
	"path"

	"github.com/olive-io/olive/api"
	errs "github.com/olive-io/olive/pkg/errors"
)

const (
	definitionPrefix = "/definitions"
)

var noPrefixEnd = []byte{0}

func (s *Server) DeployDefinition(ctx context.Context, req *api.DeployDefinitionRequest) (resp *api.DeployDefinitionResponse, err error) {
	resp = &api.DeployDefinitionResponse{}
	shardID := s.getShardID()

	definitions := &api.Definition{
		Id:      req.Id,
		Name:    req.Name,
		Content: req.Content,
	}

	data, _ := definitions.Marshal()

	key := path.Join(definitionPrefix, req.Id)
	putReq := &api.PutRequest{
		Key:         []byte(key),
		Value:       data,
		PrevKv:      false,
		IgnoreValue: false,
	}

	var rsp *api.PutResponse
	rsp, err = s.kvs.Put(ctx, shardID, putReq)
	if err != nil {
		return
	}

	resp.Header = rsp.Header
	if rsp.Header != nil {
		resp.Version = rsp.Header.Revision
	}

	return
}

func (s *Server) ListDefinition(ctx context.Context, req *api.ListDefinitionRequest) (resp *api.ListDefinitionResponse, err error) {
	resp = &api.ListDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix)
	rangeReq := &api.RangeRequest{
		Key:          []byte(key),
		RangeEnd:     getPrefix([]byte(key)),
		Serializable: true,
		KeysOnly:     true,
	}

	var rsp *api.RangeResponse
	rsp, err = s.kvs.Range(ctx, shardID, rangeReq)
	if err != nil {
		return
	}

	resp.Definitions = make([]*api.Definition, 0)
	resp.Header = rsp.Header
	for _, kv := range rsp.Kvs {
		definitions := &api.Definition{}
		if err = definitions.Unmarshal(kv.Value); err == nil {
			resp.Definitions = append(resp.Definitions, definitions)
		}
	}

	return
}

func (s *Server) GetDefinition(ctx context.Context, req *api.GetDefinitionRequest) (resp *api.GetDefinitionResponse, err error) {
	resp = &api.GetDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix, req.Id)
	rangeReq := &api.RangeRequest{
		Key:          []byte(key),
		Serializable: true,
		Limit:        1,
		Revision:     req.Version,
		MultiVersion: true,
	}

	var rsp *api.RangeResponse
	rsp, err = s.kvs.Range(ctx, shardID, rangeReq)
	if err != nil {
		return
	}

	if len(rsp.Kvs) == 0 {
		return nil, errs.ErrKeyNotFound
	}

	resp.Header = rsp.Header
	resp.Definition = &api.Definition{}
	resp.Definition.Versions = rsp.Versions
	err = resp.Definition.Unmarshal(rsp.Kvs[0].Value)

	return
}

func (s *Server) RemoveDefinition(ctx context.Context, req *api.RemoveDefinitionRequest) (resp *api.RemoveDefinitionResponse, err error) {
	resp = &api.RemoveDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix, req.Id)
	deleteReq := &api.DeleteRangeRequest{
		Key: []byte(key),
	}

	var rsp *api.DeleteRangeResponse
	rsp, err = s.kvs.DeleteRange(ctx, shardID, deleteReq)
	if err != nil {
		return
	}

	resp.Header = rsp.Header

	return
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *api.ExecuteDefinitionRequest) (resp *api.ExecuteDefinitionResponse, err error) {
	// TODO: select a runner
	return
}

func getPrefix(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return end
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to WithFromKey policy
	return noPrefixEnd
}
