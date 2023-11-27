package leader

import (
	"os"
	"testing"

	"go.etcd.io/etcd/server/v3/embed"
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

func Test_Notify(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

	etcd.Server.MoveLeader()
	nr := NewNotify(etcd.Server)
	select {
	case <-nr.ChangeNotify():
		t.Log("changed")
	default:

	}
	<-nr.ReadyNotify()
	if !nr.IsLeader() {
		t.Fatal("must be leader")
	}
}
