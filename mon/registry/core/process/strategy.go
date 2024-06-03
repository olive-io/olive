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

package process

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

// processInstanceStrategy implements verification logic for Bpmn Process Instance.
type processInstanceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Bpmn Process Instance objects.
var Strategy = processInstanceStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for core/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (processInstanceStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
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

// NamespaceScoped returns true because all processInstances need to be within a namespace.
func (processInstanceStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (processInstanceStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a processInstance before creation.
func (processInstanceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	processInstance := obj.(*corev1.Process)
	processInstance.Status = corev1.ProcessStatus{}

	processInstance.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (processInstanceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	newProcess.Status = oldProcess.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newProcess.Spec, oldProcess.Spec) {
		newProcess.Generation = oldProcess.Generation + 1
	}

}

// Validate validates a new processInstance.
func (processInstanceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	processInstance := obj.(*corev1.Process)
	return corevalidation.ValidateProcess(processInstance)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (processInstanceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newProcess := obj.(*corev1.Process)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newProcess.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Process names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (processInstanceStrategy) Canonicalize(obj runtime.Object) {
}

func (processInstanceStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for processInstances; this means a POST is needed to create one.
func (processInstanceStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (processInstanceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	processInstance := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)

	validationErrorList := corevalidation.ValidateProcess(processInstance)
	updateErrorList := corevalidation.ValidateProcessUpdate(processInstance, oldProcess)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (processInstanceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	if newProcess.Generation != oldProcess.Generation {
	}
	return warnings
}

type processInstanceStatusStrategy struct {
	processInstanceStrategy
}

var StatusStrategy = processInstanceStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (processInstanceStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (processInstanceStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	newProcess.Spec = oldProcess.Spec
}

func (processInstanceStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)

	return corevalidation.ValidateProcessUpdateStatus(newProcess, oldProcess)
}

// WarningsOnUpdate returns warnings for the given update.
func (processInstanceStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// ProcessToSelectableFields returns a field set that represents the object for matching purposes.
func ProcessToSelectableFields(processInstance *corev1.Process) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&processInstance.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	processInstance, ok := obj.(*corev1.Process)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a process.")
	}
	return labels.Set(processInstance.ObjectMeta.Labels), ProcessToSelectableFields(processInstance), nil
}

// MatchProcess is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchProcess(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
