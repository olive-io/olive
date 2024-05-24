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

package endpoint

import (
	"context"
	"fmt"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	"github.com/olive-io/olive/apis"
	apidiscoveryv1 "github.com/olive-io/olive/apis/apidiscovery/v1"
	discoveryvalidation "github.com/olive-io/olive/apis/apidiscovery/validation"
)

// endpointStrategy implements verification logic for Endpoint.
type endpointStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Endpoint objects.
var Strategy = endpointStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for apidiscovery/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (endpointStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
	var groupVersion schema.GroupVersion
	if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
		groupVersion = schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
	}
	switch groupVersion {
	case apidiscoveryv1.SchemeGroupVersion:
		// for back compatibility
		return rest.OrphanDependents
	default:
		return rest.DeleteDependents
	}
}

// NamespaceScoped returns true because all endpoints need to be within a namespace.
func (endpointStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (endpointStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a endpoint before creation.
func (endpointStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	endpoint := obj.(*apidiscoveryv1.Endpoint)
	endpoint.Status = apidiscoveryv1.EndpointStatus{}

	endpoint.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (endpointStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newEndpoint := obj.(*apidiscoveryv1.Endpoint)
	oldEndpoint := old.(*apidiscoveryv1.Endpoint)
	newEndpoint.Status = oldEndpoint.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newEndpoint.Spec, oldEndpoint.Spec) {
		newEndpoint.Generation = oldEndpoint.Generation + 1
	}

}

// Validate validates a new endpoint.
func (endpointStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	endpoint := obj.(*apidiscoveryv1.Endpoint)
	return discoveryvalidation.ValidateEndpoint(endpoint)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (endpointStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newEndpoint := obj.(*apidiscoveryv1.Endpoint)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newEndpoint.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Endpoint names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (endpointStrategy) Canonicalize(obj runtime.Object) {
}

func (endpointStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for endpoints; this means a POST is needed to create one.
func (endpointStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (endpointStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	endpoint := obj.(*apidiscoveryv1.Endpoint)
	oldEndpoint := old.(*apidiscoveryv1.Endpoint)

	validationErrorList := discoveryvalidation.ValidateEndpoint(endpoint)
	updateErrorList := discoveryvalidation.ValidateEndpointUpdate(endpoint, oldEndpoint)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (endpointStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newEndpoint := obj.(*apidiscoveryv1.Endpoint)
	oldEndpoint := old.(*apidiscoveryv1.Endpoint)
	if newEndpoint.Generation != oldEndpoint.Generation {
	}
	return warnings
}

type endpointStatusStrategy struct {
	endpointStrategy
}

var StatusStrategy = endpointStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (endpointStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (endpointStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newEndpoint := obj.(*apidiscoveryv1.Endpoint)
	oldEndpoint := old.(*apidiscoveryv1.Endpoint)
	newEndpoint.Spec = oldEndpoint.Spec
}

func (endpointStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newEndpoint := obj.(*apidiscoveryv1.Endpoint)
	oldEndpoint := old.(*apidiscoveryv1.Endpoint)

	return discoveryvalidation.ValidateEndpointUpdateStatus(newEndpoint, oldEndpoint)
}

// WarningsOnUpdate returns warnings for the given update.
func (endpointStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// EndpointToSelectableFields returns a field set that represents the object for matching purposes.
func EndpointToSelectableFields(endpoint *apidiscoveryv1.Endpoint) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&endpoint.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	endpoint, ok := obj.(*apidiscoveryv1.Endpoint)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a endpoint.")
	}
	return labels.Set(endpoint.ObjectMeta.Labels), EndpointToSelectableFields(endpoint), nil
}

// MatchEndpoint is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchEndpoint(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
