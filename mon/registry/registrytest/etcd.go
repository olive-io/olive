/*
Copyright 2024 The olive Authors

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

package registrytest

import (
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	etcd3testing "k8s.io/apiserver/pkg/storage/etcd3/testing"
	"k8s.io/apiserver/pkg/storage/storagebackend"

	"github.com/olive-io/olive/mon/embed"
	"github.com/olive-io/olive/mon/options"
)

func StartTestEmbedEtcd() (*embed.Etcd, func()) {
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

// NewEtcdStorage is for testing.  It configures the etcd storage for a bogus resource; the test must not care.
func NewEtcdStorage(t *testing.T, group string) (*storagebackend.ConfigForResource, *etcd3testing.EtcdTestServer) {
	return NewEtcdStorageForResource(t, schema.GroupResource{Group: group, Resource: "any"})
}

func NewEtcdStorageForResource(t *testing.T, resource schema.GroupResource) (*storagebackend.ConfigForResource, *etcd3testing.EtcdTestServer) {
	t.Helper()

	server, config := etcd3testing.NewUnsecuredEtcd3TestClientServer(t)

	opts := options.NewStorageEtcdOptions(config)
	completedConfig := options.NewStorageFactoryConfig().Complete(opts)
	completedConfig.APIResourceConfig = serverstorage.NewResourceConfig()
	factory, err := completedConfig.New()
	if err != nil {
		t.Fatalf("Error while making storage factory: %v", err)
	}
	resourceConfig, err := factory.NewConfig(resource)
	if err != nil {
		t.Fatalf("Error while finding storage destination: %v", err)
	}
	return resourceConfig, server
}
