/*
Copyright 2023 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
