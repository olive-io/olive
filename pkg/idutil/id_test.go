package idutil

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/olive-io/olive/pkg/runtime"
	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
)

func newEtcd() (*embed.Etcd, func()) {
	cfg := embed.NewConfig()
	cfg.Dir = "testdata"
	etcd, err := embed.StartEtcd(cfg)
	if err != nil {
		panic("start etcd: " + err.Error())
	}
	<-etcd.Server.ReadyNotify()

	cancel := func() {
		etcd.Close()
		<-etcd.Server.StopNotify()
		_ = os.RemoveAll(cfg.Dir)
	}

	return etcd, cancel
}

func TestNewGenerator(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := path.Join(runtime.DefaultOlivePrefix, "runner", "id")

	gen, err := NewGenerator(context.Background(), key, v3client.New(etcd.Server))
	if err != nil {
		t.Fatal(err)
	}

	id := gen.Next()
	id2 := gen.Next()
	if id == id2 {
		t.Errorf("generate the same id %x", id)
	}
}

func BenchmarkNext(b *testing.B) {
	etcd, cancel := newEtcd()
	defer cancel()

	key := path.Join(runtime.DefaultOlivePrefix, "runner", "id")

	gen, err := NewGenerator(context.Background(), key, v3client.New(etcd.Server))
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Next()
	}
}
