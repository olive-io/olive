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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/cel/openapi/resolver"
	quota "k8s.io/apiserver/pkg/quota/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/component-base/featuregate"

	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
)

// WantsExternalKubeClientSet defines a function which sets external ClientSet for admission plugins that need it
type WantsExternalKubeClientSet interface {
	SetExternalKubeClientSet(clientset.Interface)
	admission.InitializationValidator
}

// WantsExternalKubeInformerFactory defines a function which sets InformerFactory for admission plugins that need it
type WantsExternalKubeInformerFactory interface {
	SetExternalKubeInformerFactory(informers.SharedInformerFactory)
	admission.InitializationValidator
}

// WantsAuthorizer defines a function which sets Authorizer for admission plugins that need it.
type WantsAuthorizer interface {
	SetAuthorizer(authorizer.Authorizer)
	admission.InitializationValidator
}

// WantsQuotaConfiguration defines a function which sets quota configuration for admission plugins that need it.
type WantsQuotaConfiguration interface {
	SetQuotaConfiguration(quota.Configuration)
	admission.InitializationValidator
}

// WantsDrainedNotification defines a function which sets the notification of where the apiserver
// has already been drained for admission plugins that need it.
// After receiving that notification, Admit/Validate calls won't be called anymore.
type WantsDrainedNotification interface {
	SetDrainedNotification(<-chan struct{})
	admission.InitializationValidator
}

// WantsFeatureGate defines a function which passes the featureGates for inspection by an admission plugin.
// Admission plugins should not hold a reference to the featureGates.  Instead, they should query a particular one
// and assign it to a simple bool in the admission plugin struct.
//
//	func (a *admissionPlugin) InspectFeatureGates(features featuregate.FeatureGate){
//	    a.myFeatureIsOn = features.Enabled("my-feature")
//	}
type WantsFeatures interface {
	InspectFeatureGates(featuregate.FeatureGate)
	admission.InitializationValidator
}

type WantsDynamicClient interface {
	SetDynamicClient(dynamic.Interface)
	admission.InitializationValidator
}

// WantsRESTMapper defines a function which sets RESTMapper for admission plugins that need it.
type WantsRESTMapper interface {
	SetRESTMapper(meta.RESTMapper)
	admission.InitializationValidator
}

// WantsSchemaResolver defines a function which sets the SchemaResolver for
// an admission plugin that needs it.
type WantsSchemaResolver interface {
	SetSchemaResolver(resolver resolver.SchemaResolver)
	admission.InitializationValidator
}

// WantsExcludedAdmissionResources defines a function which sets the ExcludedAdmissionResources
// for an admission plugin that needs it.
type WantsExcludedAdmissionResources interface {
	SetExcludedAdmissionResources(excludedAdmissionResources []schema.GroupResource)
	admission.InitializationValidator
}
