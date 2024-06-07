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

package region

import (
	"testing"

	"go.etcd.io/etcd/server/v3/etcdserver/api/v3client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/generic"
	etcd3testing "k8s.io/apiserver/pkg/storage/etcd3/testing"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/registry/registrytest"
	genericdaemon "github.com/olive-io/olive/pkg/daemon"
)

func newStorage(t *testing.T) (*RegionStorage, *etcd3testing.EtcdTestServer) {
	etcdStorage, server := registrytest.NewEtcdStorage(t, corev1.GroupName)
	restOptions := generic.RESTOptions{
		StorageConfig:           etcdStorage,
		Decorator:               generic.UndecoratedStorage,
		DeleteCollectionWorkers: 1,
		ResourcePrefix:          "runners",
	}

	etcd, cancel := registrytest.StartTestEmbedEtcd()
	defer cancel()
	v3cli := v3client.New(etcd.Server)
	stopCh := genericdaemon.SetupSignalHandler()

	regionStorage, err := NewStorage(v3cli, restOptions, stopCh)
	if err != nil {
		t.Fatalf("unexpected error from REST storage: %v", err)
	}
	return &regionStorage, server
}

func validNewRegion() *corev1.Region {
	return &corev1.Region{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		},
		Spec: corev1.RegionSpec{},
	}
}
