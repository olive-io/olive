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
	corev1 "github.com/olive-io/olive/apis/core/v1"
	corevalidation "github.com/olive-io/olive/apis/core/validation"
	"github.com/olive-io/olive/pkg/idutil"
)

// regionStrategy implements verification logic for Region.
type regionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator

	ring *idutil.Ring
}

func createStrategy(ring *idutil.Ring) *regionStrategy {
	strategy := &regionStrategy{
		ObjectTyper:   apis.Scheme,
		NameGenerator: names.SimpleNameGenerator,
		ring:          ring,
	}

	return strategy
}

// DefaultGarbageCollectionPolicy returns OrphanDependents for mon/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (rs *regionStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
	var groupVersion schema.GroupVersion
	if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
		groupVersion = schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
	}
	switch groupVersion {
	case corev1.SchemeGroupVersion:
		// for back compatibility
		return rest.OrphanDependents
	default:
		return rest.DeleteDependents
	}
}

// NamespaceScoped returns true because all regions need to be within a namespace.
func (rs *regionStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (rs *regionStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a region before creation.
func (rs *regionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	region := obj.(*corev1.Region)
	region.Status = corev1.RegionStatus{}

	nextId := rs.ring.Next(ctx)
	region.Name = fmt.Sprintf("rn%d", nextId)
	region.Spec.Id = int64(nextId)
	region.Status = corev1.RegionStatus{
		Phase: corev1.RegionPending,
	}

	region.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (rs *regionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRegion := obj.(*corev1.Region)
	oldRegion := old.(*corev1.Region)
	newRegion.Status = oldRegion.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newRegion.Spec, oldRegion.Spec) {
		newRegion.Generation = oldRegion.Generation + 1
	}

}

func (rs *regionStrategy) PrepareForDelete(ctx context.Context, obj runtime.Object) error {
	region := obj.(*corev1.Region)
	rs.ring.Recycle(ctx, uint64(region.Spec.Id))
	return nil
}

// Validate validates a new region.
func (rs *regionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	region := obj.(*corev1.Region)
	return corevalidation.ValidateRegion(region)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (rs *regionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newRegion := obj.(*corev1.Region)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newRegion.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Region names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (rs *regionStrategy) Canonicalize(obj runtime.Object) {
}

func (rs *regionStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for regions; this means a POST is needed to create one.
func (rs *regionStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (rs *regionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	region := obj.(*corev1.Region)
	oldRegion := old.(*corev1.Region)

	validationErrorList := corevalidation.ValidateRegion(region)
	updateErrorList := corevalidation.ValidateRegionUpdate(region, oldRegion)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (rs *regionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newRegion := obj.(*corev1.Region)
	oldRegion := old.(*corev1.Region)
	if newRegion.Generation != oldRegion.Generation {
	}
	return warnings
}

type regionStatusStrategy struct {
	*regionStrategy
}

func createStatusStrategy(strategy *regionStrategy) *regionStatusStrategy {
	return &regionStatusStrategy{strategy}
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (regionStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (regionStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRegion := obj.(*corev1.Region)
	oldRegion := old.(*corev1.Region)
	newRegion.Spec = oldRegion.Spec
}

func (regionStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newRegion := obj.(*corev1.Region)
	oldRegion := old.(*corev1.Region)

	return corevalidation.ValidateRegionUpdateStatus(newRegion, oldRegion)
}

// WarningsOnUpdate returns warnings for the given update.
func (regionStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// RegionToSelectableFields returns a field set that represents the object for matching purposes.
func RegionToSelectableFields(region *corev1.Region) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&region.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	region, ok := obj.(*corev1.Region)
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
