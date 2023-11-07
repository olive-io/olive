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
	"encoding/xml"
	"path"

	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/olive/api/rpctypes"
	pb "github.com/olive-io/olive/api/serverpb"
	errs "github.com/olive-io/olive/pkg/errors"
)

const (
	definitionPrefix = "/dt"
)

var noPrefixEnd = []byte{0}

func (s *Server) DeployDefinition(ctx context.Context, req *pb.DeployDefinitionRequest) (resp *pb.DeployDefinitionResponse, err error) {
	resp = &pb.DeployDefinitionResponse{}
	shardID := s.getShardID()

	var sd schema.Definitions
	err = xml.Unmarshal(req.Content, &sd)
	if err != nil {
		err = rpctypes.ErrGRPCBadDefinition
		return
	}

	definitions := &pb.Definition{
		Id:      req.Id,
		Name:    req.Name,
		Content: req.Content,
	}

	data, _ := definitions.Marshal()

	key := path.Join(definitionPrefix, req.Id)
	putReq := &pb.PutRequest{
		Key:         []byte(key),
		Value:       data,
		PrevKv:      false,
		IgnoreValue: false,
	}

	var rsp *pb.PutResponse
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

func (s *Server) ListDefinition(ctx context.Context, req *pb.ListDefinitionRequest) (resp *pb.ListDefinitionResponse, err error) {
	resp = &pb.ListDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix)
	rangeReq := &pb.RangeRequest{
		Key:          []byte(key),
		RangeEnd:     getPrefix([]byte(key)),
		Serializable: true,
		KeysOnly:     true,
	}

	var rsp *pb.RangeResponse
	rsp, err = s.kvs.Range(ctx, shardID, rangeReq)
	if err != nil {
		return
	}

	resp.Definitions = make([]*pb.Definition, 0)
	resp.Header = rsp.Header
	for _, kv := range rsp.Kvs {
		definitions := &pb.Definition{}
		if err = definitions.Unmarshal(kv.Value); err == nil {
			resp.Definitions = append(resp.Definitions, definitions)
		}
	}

	return
}

func (s *Server) GetDefinition(ctx context.Context, req *pb.GetDefinitionRequest) (resp *pb.GetDefinitionResponse, err error) {
	resp = &pb.GetDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix, req.Id)
	rangeReq := &pb.RangeRequest{
		Key:          []byte(key),
		Serializable: true,
		Limit:        1,
		Revision:     req.Version,
		MultiVersion: true,
	}

	var rsp *pb.RangeResponse
	rsp, err = s.kvs.Range(ctx, shardID, rangeReq)
	if err != nil {
		return
	}

	if len(rsp.Kvs) == 0 {
		return nil, errs.ErrKeyNotFound
	}

	resp.Header = rsp.Header
	resp.Definition = &pb.Definition{}
	resp.Definition.Versions = rsp.Versions
	err = resp.Definition.Unmarshal(rsp.Kvs[0].Value)

	return
}

func (s *Server) RemoveDefinition(ctx context.Context, req *pb.RemoveDefinitionRequest) (resp *pb.RemoveDefinitionResponse, err error) {
	resp = &pb.RemoveDefinitionResponse{}

	shardID := s.getShardID()

	key := path.Join(definitionPrefix, req.Id)
	deleteReq := &pb.DeleteRangeRequest{
		Key: []byte(key),
	}

	var rsp *pb.DeleteRangeResponse
	rsp, err = s.kvs.DeleteRange(ctx, shardID, deleteReq)
	if err != nil {
		return
	}

	resp.Header = rsp.Header

	return
}

func (s *Server) ExecuteDefinition(ctx context.Context, req *pb.ExecuteDefinitionRequest) (resp *pb.ExecuteDefinitionResponse, err error) {
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
