package server_test

import (
	"context"
	"os"
	"testing"

	pb "github.com/olive-io/olive/api/serverpb"
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

func newServer(t *testing.T, dirname, address string) *server.OliveServer {
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
	defer s.Stop()

	shard := uint64(128)
	scfg := config.ShardConfig{
		Name:       "r0",
		ShardID:    shard,
		NewCluster: false,
	}
	scfg.PeerURLs, _ = types.NewURLsMap("test=http://localhost:60001")
	err := s.StartReplica(scfg)
	if !assert.NoError(t, err) {
		return
	}

	s.ShardView(shard)

	ra, _ := s.GetReplica(shard)

	ctx := context.Background()
	req := &pb.PutRequest{Key: []byte("foo"), Value: []byte("bar")}
	_, err = ra.Put(ctx, req)
	if !assert.NoError(t, err) {
		return
	}

	rsp, err := ra.Range(ctx, &pb.RangeRequest{Key: []byte("foo"), Limit: 1})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, rsp.Kvs[0].Value, []byte("bar"))
}

func TestNewCluster(t *testing.T) {
	s1 := newServer(t, "olive_kv_server_test1", testAddress1)
	s2 := newServer(t, "olive_kv_server_test2", testAddress2)
	s3 := newServer(t, "olive_kv_server_test3", testAddress3)

	shard := uint64(128)
	scfg := config.ShardConfig{
		Name:       "r0",
		ShardID:    shard,
		NewCluster: false,
	}
	scfg.PeerURLs, _ = types.NewURLsMap("test1=http://localhost:60001,test2=http://localhost:60002,test3=http://localhost:60003")
	go func() {
		err := s1.StartReplica(scfg)
		if !assert.NoError(t, err) {
			return
		}
	}()

	go func() {
		err := s2.StartReplica(scfg)
		if !assert.NoError(t, err) {
			return
		}
	}()

	err := s3.StartReplica(scfg)
	if !assert.NoError(t, err) {
		return
	}

	s3.ShardView(shard)

	ra1, _ := s1.GetReplica(shard)
	ra2, _ := s3.GetReplica(shard)

	ctx := context.Background()
	req := &pb.PutRequest{Key: []byte("foo"), Value: []byte("bar")}
	_, err = ra1.Put(ctx, req)
	if !assert.NoError(t, err) {
		return
	}

	rsp, err := ra2.Range(ctx, &pb.RangeRequest{Key: []byte("foo"), Limit: 1, Serializable: true})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, rsp.Kvs[0].Value, []byte("bar"))
}
