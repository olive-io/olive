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

package service

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
	apidiscoveryvalidation "github.com/olive-io/olive/apis/apidiscovery/validation"
)

// serviceStrategy implements verification logic for Service.
type serviceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Service objects.
var Strategy = serviceStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for apidiscovery/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (serviceStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
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

// NamespaceScoped returns true because all services need to be within a namespace.
func (serviceStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (serviceStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a service before creation.
func (serviceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	service := obj.(*apidiscoveryv1.Service)
	service.Status = apidiscoveryv1.ServiceStatus{}

	service.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (serviceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newService := obj.(*apidiscoveryv1.Service)
	oldService := old.(*apidiscoveryv1.Service)
	newService.Status = oldService.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newService.Spec, oldService.Spec) {
		newService.Generation = oldService.Generation + 1
	}

}

// Validate validates a new service.
func (serviceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	service := obj.(*apidiscoveryv1.Service)
	return apidiscoveryvalidation.ValidateService(service)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (serviceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newService := obj.(*apidiscoveryv1.Service)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newService.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Service names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (serviceStrategy) Canonicalize(obj runtime.Object) {
}

func (serviceStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for services; this means a POST is needed to create one.
func (serviceStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (serviceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	service := obj.(*apidiscoveryv1.Service)
	oldService := old.(*apidiscoveryv1.Service)

	validationErrorList := apidiscoveryvalidation.ValidateService(service)
	updateErrorList := apidiscoveryvalidation.ValidateServiceUpdate(service, oldService)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (serviceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newService := obj.(*apidiscoveryv1.Service)
	oldService := old.(*apidiscoveryv1.Service)
	if newService.Generation != oldService.Generation {
	}
	return warnings
}

type serviceStatusStrategy struct {
	serviceStrategy
}

var StatusStrategy = serviceStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (serviceStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (serviceStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newService := obj.(*apidiscoveryv1.Service)
	oldService := old.(*apidiscoveryv1.Service)
	newService.Spec = oldService.Spec
}

func (serviceStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newService := obj.(*apidiscoveryv1.Service)
	oldService := old.(*apidiscoveryv1.Service)

	return apidiscoveryvalidation.ValidateServiceUpdateStatus(newService, oldService)
}

// WarningsOnUpdate returns warnings for the given update.
func (serviceStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// ServiceSelectableFields returns a field set that represents the object for matching purposes.
func ServiceToSelectableFields(service *apidiscoveryv1.Service) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&service.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	service, ok := obj.(*apidiscoveryv1.Service)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a service.")
	}
	return labels.Set(service.ObjectMeta.Labels), ServiceToSelectableFields(service), nil
}

// MatchService is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchService(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
