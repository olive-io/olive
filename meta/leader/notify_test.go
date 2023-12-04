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

func TestEtcdCluster(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

	<-etcd.Server.ReadyNotify()

	leader := etcd.Server.Leader()
	members := etcd.Server.Cluster().Members()
	t.Logf("leader=%d, members=%+v\n", leader, members)
}

func Test_Notify(t *testing.T) {
	etcd, cancel := newEtcd()
	defer cancel()

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
