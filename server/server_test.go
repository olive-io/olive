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

package server_test

import (
	"context"
	"os"
	"testing"

	"github.com/olive-io/olive/api"
	"github.com/olive-io/olive/server"
	"github.com/olive-io/olive/server/config"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/client/pkg/v3/types"
)

const (
	testAddress1 = "localhost:60001"
	testAddress2 = "localhost:60002"
	testAddress3 = "localhost:60003"
)

func newServer(t *testing.T, dirname, address string) *server.KVServer {
	dir, err := os.MkdirTemp(t.TempDir(), dirname)
	if err != nil {
		panic(err)
	}

	lgCfg := config.NewLoggerConfig()
	lgCfg.Pkgs = `{"transport": "debug","rsm": "debug", "raft": "debug", "grpc": "debug"}`
	if err = lgCfg.Apply(); err != nil {
		panic(err)
	}

	cfg := config.NewServerConfig(dir, address)
	if err = cfg.Apply(); err != nil {
		panic(err)
	}

	s, err := server.NewServer(lgCfg.GetLogger(), cfg)
	if !assert.NoError(t, err) {
		return nil
	}
	s.Start()

	return s
}

func TestNewServer(t *testing.T) {
	s := newServer(t, "olive_kv_server_test", testAddress1)

	scfg := config.ShardConfig{
		Name:       "r0",
		NewCluster: false,
	}
	scfg.PeerURLs, _ = types.NewURLsMap("test=http://localhost:60001")
	_, err := s.StartReplica(scfg)
	if !assert.NoError(t, err) {
		return
	}

	shard := server.GenHash([]byte(scfg.Name))
	ctx := context.Background()
	req := &api.PutRequest{Key: []byte("foo"), Value: []byte("bar")}
	_, err = s.Put(ctx, shard, req)
	if !assert.NoError(t, err) {
		return
	}

	rsp, err := s.Range(ctx, shard, &api.RangeRequest{Key: []byte("foo"), Limit: 1})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, rsp.Kvs[0].Value, []byte("bar"))
}

func TestNewCluster(t *testing.T) {
	s1 := newServer(t, "olive_kv_server_test1", testAddress1)
	s2 := newServer(t, "olive_kv_server_test2", testAddress2)
	s3 := newServer(t, "olive_kv_server_test3", testAddress3)

	scfg := config.ShardConfig{
		Name:       "r0",
		NewCluster: false,
	}
	scfg.PeerURLs, _ = types.NewURLsMap("test1=http://localhost:60001,test2=http://localhost:60002,test3=http://localhost:60003")
	go func() {
		_, err := s1.StartReplica(scfg)
		if !assert.NoError(t, err) {
			return
		}
	}()

	go func() {
		_, err := s2.StartReplica(scfg)
		if !assert.NoError(t, err) {
			return
		}
	}()

	_, err := s3.StartReplica(scfg)
	if !assert.NoError(t, err) {
		return
	}

	shard := server.GenHash([]byte(scfg.Name))
	ctx := context.Background()
	req := &api.PutRequest{Key: []byte("foo"), Value: []byte("bar")}
	_, err = s1.Put(ctx, shard, req)
	if !assert.NoError(t, err) {
		return
	}

	rsp, err := s2.Range(ctx, shard, &api.RangeRequest{Key: []byte("foo"), Limit: 1, Serializable: true})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, rsp.Kvs[0].Value, []byte("bar"))
}
