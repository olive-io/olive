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
)

const (
	definitionPrefix = "/definitions"
)

func (s *Server) DeployDefinition(ctx context.Context, req *api.DeployDefinitionRequest) (resp *api.DeployDefinitionResponse, err error) {
	definitions := &api.Definition{
		Id:      req.Id,
		Name:    req.Name,
		Version: 0,
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
	rsp, err = s.kvs.Put(ctx, 0, putReq)
	if err != nil {
		return
	}

	resp.Version = rsp.Header.Revision
	return
}

func (s *Server) ListDefinition(ctx context.Context, req *api.ListDefinitionRequest) (resp *api.ListDefinitionResponse, err error) {
	key := path.Join(definitionPrefix)
	rangeReq := &api.RangeRequest{
		Key:          []byte(key),
		Serializable: true,
	}

	var rsp *api.RangeResponse
	rsp, err = s.kvs.Range(ctx, 0, rangeReq)
	if err != nil {
		return
	}

	resp.Definition = make([]*api.Definition, 0)
	for _, kv := range rsp.Kvs {
		definitions := &api.Definition{}
		if err = definitions.Unmarshal(kv.Value); err == nil {
			resp.Definition = append(resp.Definition, definitions)
		}
	}

	return
}

func (s *Server) GetDefinition(ctx context.Context, req *api.GetDefinitionRequest) (resp *api.GetDefinitionResponse, err error) {
	key := path.Join(definitionPrefix)
	rangeReq := &api.RangeRequest{
		Key:          []byte(key),
		Serializable: true,
		Limit:        1,
	}

	var rsp *api.RangeResponse
	rsp, err = s.kvs.Range(ctx, 0, rangeReq)
	if err != nil {
		return
	}

	resp.Definition = &api.Definition{}
	err = resp.Definition.Unmarshal(rsp.Kvs[0].Value)

	return
}

func (s *Server) RemoveDefinition(ctx context.Context, req *api.RemoveDefinitionRequest) (resp *api.RemoveDefinitionResponse, err error) {
	key := path.Join(definitionPrefix, req.Id)
	deleteReq := &api.DeleteRangeRequest{
		Key: []byte(key),
	}

	var rsp *api.DeleteRangeResponse
	rsp, err = s.kvs.DeleteRange(ctx, 0, deleteReq)
	if err != nil {
		return
	}

	_ = rsp
	return
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *api.ExecuteDefinitionRequest) (resp *api.ExecuteDefinitionResponse, err error) {
	// TODO: select a runner
	return
}
