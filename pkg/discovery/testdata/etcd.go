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
