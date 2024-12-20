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

package testdata

import (
	"os"
	"testing"

	"go.etcd.io/etcd/server/v3/embed"
	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	"go.uber.org/zap"

	dsy "github.com/olive-io/olive/pkg/discovery"
)

func TestDiscovery(t *testing.T) (dsy.IDiscovery, func()) {
	cfg := embed.NewConfig()
	cfg.Dir = ".testdata"

	etcd, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}
	<-etcd.Server.ReadyNotify()

	cancel := func() {
		etcd.Server.HardStop()
		<-etcd.Server.StopNotify()
		_ = os.RemoveAll(cfg.Dir)
	}

	lg := zap.NewExample()
	discovery, err := dsy.NewDiscovery(v3client.New(etcd.Server), dsy.SetLogger(lg))
	if err != nil {
		cancel()
		t.Fatal(err)
	}

	return discovery, cancel
}
