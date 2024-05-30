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

package initializer

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/client-go/dynamic"
	"k8s.io/component-base/featuregate"

	clientset "github.com/olive-io/olive/client/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client/generated/informers/externalversions"
)

type pluginInitializer struct {
	externalClient    clientset.Interface
	dynamicClient     dynamic.Interface
	externalInformers informers.SharedInformerFactory
	authorizer        authorizer.Authorizer
	featureGates      featuregate.FeatureGate
	stopCh            <-chan struct{}
	restMapper        meta.RESTMapper
}

// New creates an instance of admission plugins initializer.
// This constructor is public with a long param list so that callers immediately know that new information can be expected
// during compilation when they update a level.
func New(
	extClientset clientset.Interface,
	dynamicClient dynamic.Interface,
	extInformers informers.SharedInformerFactory,
	authz authorizer.Authorizer,
	featureGates featuregate.FeatureGate,
	stopCh <-chan struct{},
	restMapper meta.RESTMapper,
) pluginInitializer {
	return pluginInitializer{
		externalClient:    extClientset,
		dynamicClient:     dynamicClient,
		externalInformers: extInformers,
		authorizer:        authz,
		featureGates:      featureGates,
		stopCh:            stopCh,
		restMapper:        restMapper,
	}
}

// Initialize checks the initialization interfaces implemented by a plugin
// and provide the appropriate initialization data
func (i pluginInitializer) Initialize(plugin admission.Interface) {
	// First tell the plugin about drained notification, so it can pass it to further initializations.
	if wants, ok := plugin.(WantsDrainedNotification); ok {
		wants.SetDrainedNotification(i.stopCh)
	}

	// Second tell the plugin about enabled features, so it can decide whether to start informers or not
	if wants, ok := plugin.(WantsFeatures); ok {
		wants.InspectFeatureGates(i.featureGates)
	}

	if wants, ok := plugin.(WantsExternalKubeClientSet); ok {
		wants.SetExternalKubeClientSet(i.externalClient)
	}

	if wants, ok := plugin.(WantsDynamicClient); ok {
		wants.SetDynamicClient(i.dynamicClient)
	}

	if wants, ok := plugin.(WantsExternalKubeInformerFactory); ok {
		wants.SetExternalKubeInformerFactory(i.externalInformers)
	}

	if wants, ok := plugin.(WantsAuthorizer); ok {
		wants.SetAuthorizer(i.authorizer)
	}
	if wants, ok := plugin.(WantsRESTMapper); ok {
		wants.SetRESTMapper(i.restMapper)
	}
}

var _ admission.PluginInitializer = pluginInitializer{}
