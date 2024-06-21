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

package validation

import (
	"strings"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	"github.com/olive-io/olive/apis/core/helper"
	apivalidation "github.com/olive-io/olive/apis/core/validation"
)

// ValidateNameFunc validates that the provided name is valid for a given resource type.
// Not all resources have the same validation rules for names. Prefix is true
// if the name will have a value appended to it.  If the name is not valid,
// this returns a list of descriptions of individual characteristics of the
// value that were not valid.  Otherwise this returns an empty list or nil.
type ValidateNameFunc apimachineryvalidation.ValidateNameFunc

// ValidatePluginServiceName can be used to check whether the given service name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidatePluginServiceName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateObjectMeta validates an object's metadata on creation. It expects that name generation has already
// been performed.
// It doesn't return an error for rootscoped resources with namespace, because namespace should already be cleared before.
func ValidateObjectMeta(meta *metav1.ObjectMeta, requiresNamespace bool, nameFn ValidateNameFunc, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateObjectMeta(meta, requiresNamespace, apimachineryvalidation.ValidateNameFunc(nameFn), fldPath)
	// run additional checks for the finalizer name
	for i := range meta.Finalizers {
		allErrs = append(allErrs, validateKubeFinalizerName(string(meta.Finalizers[i]), fldPath.Child("finalizers").Index(i))...)
	}
	return allErrs
}

// ValidatePluginService tests if required fields are set.
func ValidatePluginService(service *apidiscoveryv1.PluginService) field.ErrorList {
	allErrs := ValidateObjectMeta(&service.ObjectMeta, true, ValidatePluginServiceName, field.NewPath("metadata"))
	return allErrs
}

// ValidatePluginServiceUpdate validates an update to a PluginService and returns an ErrorList with any errors.
func ValidatePluginServiceUpdate(service, oldPluginService *apidiscoveryv1.PluginService) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&service.ObjectMeta, &oldPluginService.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidatePluginServiceSpecUpdate(service.Spec, oldPluginService.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidatePluginServiceSpecUpdate validates an update to a PluginServiceSpec and returns an ErrorList with any errors.
func ValidatePluginServiceSpecUpdate(spec, oldSpec apidiscoveryv1.PluginServiceSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidatePluginServiceUpdateStatus validates an update to the status of a PluginService and returns an ErrorList with any errors.
func ValidatePluginServiceUpdateStatus(service, oldPluginService *apidiscoveryv1.PluginService) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&service.ObjectMeta, &oldPluginService.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidatePluginServiceStatusUpdate(service, oldPluginService)...)
	return allErrs
}

// ValidatePluginServiceStatusUpdate validates an update to a PluginServiceStatus and returns an ErrorList with any errors.
func ValidatePluginServiceStatusUpdate(service, oldPluginService *apidiscoveryv1.PluginService) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validatePluginServiceStatus(service, statusFld)...)

	return allErrs
}

// validatePluginServiceStatus validates a PluginServiceStatus and returns an ErrorList with any errors.
func validatePluginServiceStatus(service *apidiscoveryv1.PluginService, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateEndpointName can be used to check whether the given endpoint name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateEndpointName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateEndpoint tests if required fields are set.
func ValidateEndpoint(endpoint *apidiscoveryv1.Endpoint) field.ErrorList {
	allErrs := ValidateObjectMeta(&endpoint.ObjectMeta, true, ValidateEndpointName, field.NewPath("metadata"))
	return allErrs
}

// ValidateEndpointUpdate validates an update to a Endpoint and returns an ErrorList with any errors.
func ValidateEndpointUpdate(endpoint, oldEndpoint *apidiscoveryv1.Endpoint) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&endpoint.ObjectMeta, &oldEndpoint.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateEndpointSpecUpdate(endpoint.Spec, oldEndpoint.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateEndpointSpecUpdate validates an update to a EndpointSpec and returns an ErrorList with any errors.
func ValidateEndpointSpecUpdate(spec, oldSpec apidiscoveryv1.EndpointSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateEndpointUpdateStatus validates an update to the status of a Endpoint and returns an ErrorList with any errors.
func ValidateEndpointUpdateStatus(endpoint, oldEndpoint *apidiscoveryv1.Endpoint) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&endpoint.ObjectMeta, &oldEndpoint.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateEndpointStatusUpdate(endpoint, oldEndpoint)...)
	return allErrs
}

// ValidateEndpointStatusUpdate validates an update to a EndpointStatus and returns an ErrorList with any errors.
func ValidateEndpointStatusUpdate(endpoint, oldEndpoint *apidiscoveryv1.Endpoint) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateEndpointStatus(endpoint, statusFld)...)

	return allErrs
}

// validateEndpointStatus validates a EndpointStatus and returns an ErrorList with any errors.
func validateEndpointStatus(endpoint *apidiscoveryv1.Endpoint, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateEdgeName can be used to check whether the given edge name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateEdgeName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateEdge tests if required fields are set.
func ValidateEdge(edge *apidiscoveryv1.Edge) field.ErrorList {
	allErrs := ValidateObjectMeta(&edge.ObjectMeta, true, ValidateEdgeName, field.NewPath("metadata"))
	return allErrs
}

// ValidateEdgeUpdate validates an update to a Edge and returns an ErrorList with any errors.
func ValidateEdgeUpdate(edge, oldEdge *apidiscoveryv1.Edge) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&edge.ObjectMeta, &oldEdge.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateEdgeSpecUpdate(edge.Spec, oldEdge.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateEdgeSpecUpdate validates an update to a EdgeSpec and returns an ErrorList with any errors.
func ValidateEdgeSpecUpdate(spec, oldSpec apidiscoveryv1.EdgeSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateEdgeUpdateStatus validates an update to the status of a Edge and returns an ErrorList with any errors.
func ValidateEdgeUpdateStatus(edge, oldEdge *apidiscoveryv1.Edge) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&edge.ObjectMeta, &oldEdge.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateEdgeStatusUpdate(edge, oldEdge)...)
	return allErrs
}

// ValidateEdgeStatusUpdate validates an update to a EdgeStatus and returns an ErrorList with any errors.
func ValidateEdgeStatusUpdate(edge, oldEdge *apidiscoveryv1.Edge) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateEdgeStatus(edge, statusFld)...)

	return allErrs
}

// validateEdgeStatus validates a EdgeStatus and returns an ErrorList with any errors.
func validateEdgeStatus(edge *apidiscoveryv1.Edge, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// Validate finalizer names
func validateFinalizerName(stringValue string, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateFinalizerName(stringValue, fldPath)
	allErrs = append(allErrs, validateKubeFinalizerName(stringValue, fldPath)...)
	return allErrs
}

// validateKubeFinalizerName checks for "standard" names of legacy finalizer
func validateKubeFinalizerName(stringValue string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(strings.Split(stringValue, "/")) == 1 {
		if !helper.IsStandardFinalizerName(stringValue) {
			return append(allErrs, field.Invalid(fldPath, stringValue, "name is neither a standard finalizer name nor is it fully qualified"))
		}
	}

	return allErrs
}
