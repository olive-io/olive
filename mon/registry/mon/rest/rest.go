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
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	serverstorage "k8s.io/apiserver/pkg/server/storage"

	"github.com/olive-io/olive/apis"
	"github.com/olive-io/olive/apis/mon"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	regionstore "github.com/olive-io/olive/mon/registry/mon/region/storage"
	runnerstore "github.com/olive-io/olive/mon/registry/mon/runner/storage"
)

type RESTStorageProvider struct{}

func (p *RESTStorageProvider) NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(monv1.GroupName, apis.Scheme, apis.ParameterCodec, apis.Codecs)

	if storageMap, err := p.v1Storage(apiResourceConfigSource, restOptionsGetter); err != nil {
		return genericapiserver.APIGroupInfo{}, err
	} else if len(storageMap) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[monv1.SchemeGroupVersion.Version] = storageMap
	}

	return apiGroupInfo, nil
}

func (p *RESTStorageProvider) v1Storage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (map[string]rest.Storage, error) {
	storage := map[string]rest.Storage{}

	// runner
	if resource := "runner"; apiResourceConfigSource.ResourceEnabled(monv1.SchemeGroupVersion.WithResource(resource)) {
		runnersStorage, runnersStatusStorage, err := runnerstore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = runnersStorage
		storage[resource+"/status"] = runnersStatusStorage
	}

	// region
	if resource := "region"; apiResourceConfigSource.ResourceEnabled(monv1.SchemeGroupVersion.WithResource(resource)) {
		regionsStorage, regionsStatusStorage, err := regionstore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = regionsStorage
		storage[resource+"/status"] = regionsStatusStorage
	}

	return storage, nil
}

func (p *RESTStorageProvider) GroupName() string {
	return mon.GroupName
}
