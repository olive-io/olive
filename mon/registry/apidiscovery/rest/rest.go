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
	"github.com/olive-io/olive/apis/apidiscovery"
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	edgestore "github.com/olive-io/olive/mon/registry/apidiscovery/edge"
	endpointstore "github.com/olive-io/olive/mon/registry/apidiscovery/endpoint"
	servicestore "github.com/olive-io/olive/mon/registry/apidiscovery/service"
)

type APIRESTStorageProvider struct{}

func NewRESTStorageProvider() (*APIRESTStorageProvider, error) {
	provider := &APIRESTStorageProvider{}
	return provider, nil
}

func (p *APIRESTStorageProvider) NewRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(apidiscoveryv1.GroupName, apis.Scheme, apis.ParameterCodec, apis.Codecs)

	if storageMap, err := p.v1Storage(apiResourceConfigSource, restOptionsGetter); err != nil {
		return genericapiserver.APIGroupInfo{}, err
	} else if len(storageMap) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[apidiscoveryv1.SchemeGroupVersion.Version] = storageMap
	}

	return apiGroupInfo, nil
}

func (p *APIRESTStorageProvider) v1Storage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (map[string]rest.Storage, error) {
	storage := map[string]rest.Storage{}

	// endpoint
	if resource := "endpoints"; apiResourceConfigSource.ResourceEnabled(apidiscoveryv1.SchemeGroupVersion.WithResource(resource)) {
		endpointsStorage, endpointsStatusStorage, err := endpointstore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = endpointsStorage
		storage[resource+"/status"] = endpointsStatusStorage
	}

	// service
	if resource := "pluginservices"; apiResourceConfigSource.ResourceEnabled(apidiscoveryv1.SchemeGroupVersion.WithResource(resource)) {
		servicesStorage, servicesStatusStorage, err := servicestore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = servicesStorage
		storage[resource+"/status"] = servicesStatusStorage
	}

	// edge
	if resource := "edges"; apiResourceConfigSource.ResourceEnabled(apidiscoveryv1.SchemeGroupVersion.WithResource(resource)) {
		edgesStorage, edgesStatusStorage, err := edgestore.NewREST(restOptionsGetter)
		if err != nil {
			return storage, err
		}
		storage[resource] = edgesStorage
		storage[resource+"/status"] = edgesStatusStorage
	}

	return storage, nil
}

func (p *APIRESTStorageProvider) GroupName() string {
	return apidiscovery.GroupName
}
