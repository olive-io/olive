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

package definition

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
)

// definitionStrategy implements verification logic for Bpmn Definitions.
type definitionStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Bpmn Definitions objects.
var Strategy = definitionStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for core/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (definitionStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
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

// NamespaceScoped returns true because all definitions need to be within a namespace.
func (definitionStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (definitionStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a definition before creation.
func (s definitionStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	definition := obj.(*corev1.Definition)
	definition.Status = corev1.DefinitionStatus{}

	definition.Generation = 1
	definition.Spec.Version = 1

	definition.Status.Phase = corev1.DefPending
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (definitionStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newDefinition := obj.(*corev1.Definition)
	oldDefinition := old.(*corev1.Definition)
	newDefinition.Status = oldDefinition.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newDefinition.Spec, oldDefinition.Spec) {
		newDefinition.Generation = oldDefinition.Generation + 1
	}

	if oldDefinition.Spec.Content != newDefinition.Spec.Content {
		newDefinition.Spec.Version += 1
	}
}

// Validate validates a new definition.
func (definitionStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	definition := obj.(*corev1.Definition)
	return corevalidation.ValidateDefinition(definition)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (definitionStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newDefinition := obj.(*corev1.Definition)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newDefinition.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Definition names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (definitionStrategy) Canonicalize(obj runtime.Object) {
}

func (definitionStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for definitions; this means a POST is needed to create one.
func (definitionStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (definitionStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	definition := obj.(*corev1.Definition)
	oldDefinition := old.(*corev1.Definition)

	validationErrorList := corevalidation.ValidateDefinition(definition)
	updateErrorList := corevalidation.ValidateDefinitionUpdate(definition, oldDefinition)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (definitionStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newDefinition := obj.(*corev1.Definition)
	oldDefinition := old.(*corev1.Definition)
	if newDefinition.Generation != oldDefinition.Generation {
	}
	return warnings
}

type definitionStatusStrategy struct {
	definitionStrategy
}

var StatusStrategy = definitionStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (definitionStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (definitionStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newDefinition := obj.(*corev1.Definition)
	oldDefinition := old.(*corev1.Definition)
	newDefinition.Spec = oldDefinition.Spec
}

func (definitionStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newDefinition := obj.(*corev1.Definition)
	oldDefinition := old.(*corev1.Definition)

	return corevalidation.ValidateDefinitionUpdateStatus(newDefinition, oldDefinition)
}

// WarningsOnUpdate returns warnings for the given update.
func (definitionStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// DefinitionToSelectableFields returns a field set that represents the object for matching purposes.
func DefinitionToSelectableFields(definition *corev1.Definition) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&definition.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	definition, ok := obj.(*corev1.Definition)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a definition.")
	}
	return labels.Set(definition.ObjectMeta.Labels), DefinitionToSelectableFields(definition), nil
}

// MatchDefinition is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchDefinition(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
