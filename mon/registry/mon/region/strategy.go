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
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	monvalidation "github.com/olive-io/olive/apis/mon/validation"
)

// regionStrategy implements verification logic for Region.
type regionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Region objects.
var Strategy = regionStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for mon/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (regionStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
	var groupVersion schema.GroupVersion
	if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
		groupVersion = schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
	}
	switch groupVersion {
	case monv1.SchemeGroupVersion:
		// for back compatibility
		return rest.OrphanDependents
	default:
		return rest.DeleteDependents
	}
}

// NamespaceScoped returns true because all regions need to be within a namespace.
func (regionStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (regionStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"mon/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a region before creation.
func (regionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	region := obj.(*monv1.Region)
	region.Status = monv1.RegionStatus{}

	region.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (regionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRegion := obj.(*monv1.Region)
	oldRegion := old.(*monv1.Region)
	newRegion.Status = oldRegion.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newRegion.Spec, oldRegion.Spec) {
		newRegion.Generation = oldRegion.Generation + 1
	}

}

// Validate validates a new region.
func (regionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	region := obj.(*monv1.Region)
	return monvalidation.ValidateRegion(region)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (regionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newRegion := obj.(*monv1.Region)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newRegion.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Region names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (regionStrategy) Canonicalize(obj runtime.Object) {
}

func (regionStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for regions; this means a POST is needed to create one.
func (regionStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (regionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	region := obj.(*monv1.Region)
	oldRegion := old.(*monv1.Region)

	validationErrorList := monvalidation.ValidateRegion(region)
	updateErrorList := monvalidation.ValidateRegionUpdate(region, oldRegion)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (regionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newRegion := obj.(*monv1.Region)
	oldRegion := old.(*monv1.Region)
	if newRegion.Generation != oldRegion.Generation {
	}
	return warnings
}

type regionStatusStrategy struct {
	regionStrategy
}

var StatusStrategy = regionStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (regionStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"mon/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (regionStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRegion := obj.(*monv1.Region)
	oldRegion := old.(*monv1.Region)
	newRegion.Spec = oldRegion.Spec
}

func (regionStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newRegion := obj.(*monv1.Region)
	oldRegion := old.(*monv1.Region)

	return monvalidation.ValidateRegionUpdateStatus(newRegion, oldRegion)
}

// WarningsOnUpdate returns warnings for the given update.
func (regionStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// RegionToSelectableFields returns a field set that represents the object for matching purposes.
func RegionToSelectableFields(region *monv1.Region) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&region.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	region, ok := obj.(*monv1.Region)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a region.")
	}
	return labels.Set(region.ObjectMeta.Labels), RegionToSelectableFields(region), nil
}

// MatchRegion is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchRegion(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
