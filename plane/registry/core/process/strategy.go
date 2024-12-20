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

// processStrategy implements verification logic for Bpmn Process Instance.
type processStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Bpmn Process Instance objects.
var Strategy = processStrategy{apis.Scheme, names.SimpleNameGenerator}

// DefaultGarbageCollectionPolicy returns OrphanDependents for core/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (processStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
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

// NamespaceScoped returns true because all processs need to be within a namespace.
func (processStrategy) NamespaceScoped() bool {
	return true
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (processStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a process before creation.
func (processStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	process := obj.(*corev1.Process)
	process.Status = corev1.ProcessStatus{}

	process.Generation = 1
	process.Status.Phase = corev1.ProcessPending
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (processStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	newProcess.Status = oldProcess.Status

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newProcess.Spec, oldProcess.Spec) {
		newProcess.Generation = oldProcess.Generation + 1
	}

	if newProcess.Status.Phase == "" {
		newProcess.Status.Phase = corev1.ProcessPending
	}
}

// Validate validates a new process.
func (processStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	process := obj.(*corev1.Process)
	return corevalidation.ValidateProcess(process)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (processStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	newProcess := obj.(*corev1.Process)
	var warnings []string
	if msgs := utilvalidation.IsDNS1123Label(newProcess.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Process names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (processStrategy) Canonicalize(obj runtime.Object) {
}

func (processStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for processs; this means a POST is needed to create one.
func (processStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (processStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	process := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)

	validationErrorList := corevalidation.ValidateProcess(process)
	updateErrorList := corevalidation.ValidateProcessUpdate(process, oldProcess)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (processStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	if newProcess.Generation != oldProcess.Generation {
	}
	return warnings
}

type processStatusStrategy struct {
	processStrategy
}

var StatusStrategy = processStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (processStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (processStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)
	newProcess.Spec = oldProcess.Spec
}

func (processStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newProcess := obj.(*corev1.Process)
	oldProcess := old.(*corev1.Process)

	return corevalidation.ValidateProcessUpdateStatus(newProcess, oldProcess)
}

// WarningsOnUpdate returns warnings for the given update.
func (processStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// ProcessToSelectableFields returns a field set that represents the object for matching purposes.
func ProcessToSelectableFields(process *corev1.Process) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&process.ObjectMeta, true)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	process, ok := obj.(*corev1.Process)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a process.")
	}
	return labels.Set(process.ObjectMeta.Labels), ProcessToSelectableFields(process), nil
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
