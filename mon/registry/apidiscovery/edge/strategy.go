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

package edge

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

// edgeStrategy implements verification logic for Edge.
type edgeStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Edge objects.
var Strategy = edgeStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for apidiscovery/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (edgeStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
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

// NamespaceScoped returns true because all edges need to be within a namespace.
func (edgeStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (edgeStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a edge before creation.
func (edgeStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	edge := obj.(*apidiscoveryv1.Edge)
	edge.Status = apidiscoveryv1.EdgeStatus{}

	edge.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (edgeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newEdge := obj.(*apidiscoveryv1.Edge)
	oldEdge := old.(*apidiscoveryv1.Edge)
	newEdge.Status = oldEdge.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newEdge.Spec, oldEdge.Spec) {
		newEdge.Generation = oldEdge.Generation + 1
	}

}

// Validate validates a new edge.
func (edgeStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	edge := obj.(*apidiscoveryv1.Edge)
	return apidiscoveryvalidation.ValidateEdge(edge)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (edgeStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newEdge := obj.(*apidiscoveryv1.Edge)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newEdge.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Edge names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (edgeStrategy) Canonicalize(obj runtime.Object) {
}

func (edgeStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for edges; this means a POST is needed to create one.
func (edgeStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (edgeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	edge := obj.(*apidiscoveryv1.Edge)
	oldEdge := old.(*apidiscoveryv1.Edge)

	validationErrorList := apidiscoveryvalidation.ValidateEdge(edge)
	updateErrorList := apidiscoveryvalidation.ValidateEdgeUpdate(edge, oldEdge)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (edgeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newEdge := obj.(*apidiscoveryv1.Edge)
	oldEdge := old.(*apidiscoveryv1.Edge)
	if newEdge.Generation != oldEdge.Generation {
	}
	return warnings
}

type edgeStatusStrategy struct {
	edgeStrategy
}

var StatusStrategy = edgeStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (edgeStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"apidiscovery/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (edgeStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newEdge := obj.(*apidiscoveryv1.Edge)
	oldEdge := old.(*apidiscoveryv1.Edge)
	newEdge.Spec = oldEdge.Spec
}

func (edgeStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newEdge := obj.(*apidiscoveryv1.Edge)
	oldEdge := old.(*apidiscoveryv1.Edge)

	return apidiscoveryvalidation.ValidateEdgeUpdateStatus(newEdge, oldEdge)
}

// WarningsOnUpdate returns warnings for the given update.
func (edgeStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// EdgeSelectableFields returns a field set that represents the object for matching purposes.
func EdgeToSelectableFields(edge *apidiscoveryv1.Edge) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&edge.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	edge, ok := obj.(*apidiscoveryv1.Edge)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a edge.")
	}
	return labels.Set(edge.ObjectMeta.Labels), EdgeToSelectableFields(edge), nil
}

// MatchEdge is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchEdge(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
