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

package rest

import (
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	definitionstore "github.com/olive-io/olive/plane/registry/core/definition"
	namespacestore "github.com/olive-io/olive/plane/registry/core/namespace"
	processstore "github.com/olive-io/olive/plane/registry/core/process"
	regionstore "github.com/olive-io/olive/plane/registry/core/region"
	runnerstore "github.com/olive-io/olive/plane/registry/core/runner"
)

type CoreRESTStorageProvider struct {
	v3cli  *clientv3.Client
	stopCh <-chan struct{}
}

func NewRESTStorageProvider(v3cli *clientv3.Client, stopCh <-chan struct{}) (*CoreRESTStorageProvider, error) {
	provider := &CoreRESTStorageProvider{
		v3cli:  v3cli,
		stopCh: stopCh,
	}

	return provider, nil
}

func (p *CoreRESTStorageProvider) NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(corev1.GroupName, apis.Scheme, apis.ParameterCodec, apis.Codecs)

	if storageMap, err := p.v1Storage(apiResourceConfigSource, restOptionsGetter); err != nil {
		return genericapiserver.APIGroupInfo{}, err
	} else if len(storageMap) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[corev1.SchemeGroupVersion.Version] = storageMap
	}

	return apiGroupInfo, nil
}

func (p *CoreRESTStorageProvider) v1Storage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (map[string]rest.Storage, error) {
	storage := map[string]rest.Storage{}

	// runner
	if resource := "runners"; apiResourceConfigSource.ResourceEnabled(corev1.SchemeGroupVersion.WithResource(resource)) {
		runnerStorage, err := runnerstore.NewStorage(p.v3cli, restOptionsGetter, p.stopCh)
		if err != nil {
			return storage, err
		}
		storage[resource] = runnerStorage.Runner
		storage[resource+"/status"] = runnerStorage.Status
	}

	// region
	if resource := "regions"; apiResourceConfigSource.ResourceEnabled(corev1.SchemeGroupVersion.WithResource(resource)) {
		regionsStorage, err := regionstore.NewStorage(p.v3cli, restOptionsGetter, p.stopCh)
		if err != nil {
			return storage, err
		}
		storage[resource] = regionsStorage.Region
		storage[resource+"/status"] = regionsStorage.Status
	}

	// namespace
	if resource := "namespaces"; apiResourceConfigSource.ResourceEnabled(corev1.SchemeGroupVersion.WithResource(resource)) {
		namespaceStorage, namespaceStatusStorage, finalizeStorage, err := namespacestore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = namespaceStorage
		storage[resource+"/status"] = namespaceStatusStorage
		storage[resource+"/finalize"] = finalizeStorage
	}

	// definition
	if resource := "definitions"; apiResourceConfigSource.ResourceEnabled(corev1.SchemeGroupVersion.WithResource(resource)) {
		defStorage, err := definitionstore.NewStorage(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = defStorage.Definition
		storage[resource+"/status"] = defStorage.Status
	}

	// processInstance
	if resource := "processes"; apiResourceConfigSource.ResourceEnabled(corev1.SchemeGroupVersion.WithResource(resource)) {
		processStorage, err := processstore.NewStorage(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = processStorage.Process
		storage[resource+"/status"] = processStorage.Status
	}

	return storage, nil
}

func (p *CoreRESTStorageProvider) GroupName() string {
	return core.GroupName
}
