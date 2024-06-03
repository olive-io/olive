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

package runner

import (
	"context"
	"fmt"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilvalidation "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

// runnerStrategy implements verification logic for Runner.
type runnerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator

	ring *idutil.Ring
}

func createStrategy(ring *idutil.Ring) *runnerStrategy {
	strategy := &runnerStrategy{
		ObjectTyper:   apis.Scheme,
		NameGenerator: names.SimpleNameGenerator,
		ring:          ring,
	}

	return strategy
}

// DefaultGarbageCollectionPolicy returns OrphanDependents for mon/v1 for backwards compatibility,
// and DeleteDependents for all other versions.
func (rs *runnerStrategy) DefaultGarbageCollectionPolicy(ctx context.Context) rest.GarbageCollectionPolicy {
	//var groupVersion schema.GroupVersion
	//if requestInfo, found := genericapirequest.RequestInfoFrom(ctx); found {
	//	groupVersion = schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
	//}
	//switch groupVersion {
	//case corev1.SchemeGroupVersion:
	//	// for back compatibility
	//	return rest.OrphanDependents
	//default:
	//	return rest.DeleteDependents
	//}
	return rest.DeleteDependents
}

// NamespaceScoped returns true because all runners need to be within a namespace.
func (rs *runnerStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (rs *runnerStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a runner before creation.
func (rs *runnerStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	runner, ok := obj.(*corev1.Runner)
	if !ok {
		return
	}

	nextId := rs.ring.Next(ctx)
	runner.Name = fmt.Sprintf("runner%d", nextId)
	runner.Spec.ID = int64(nextId)
	runner.Status = corev1.RunnerStatus{
		Phase: corev1.RunnerPending,
	}

	runner.Generation = 1
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (rs *runnerStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRunner := obj.(*corev1.Runner)
	oldRunner := old.(*corev1.Runner)

	newRunner.Spec.DeepCopyInto(&oldRunner.Spec)

	// See metav1.ObjectMeta description for more information on Generation.
	if !apiequality.Semantic.DeepEqual(newRunner.Spec, oldRunner.Spec) {
		newRunner.Generation = oldRunner.Generation + 1
	}
}

func (rs *runnerStrategy) PrepareForDelete(ctx context.Context, obj runtime.Object) error {
	runner := obj.(*corev1.Runner)
	rs.ring.Recycle(ctx, uint64(runner.Spec.ID))
	return nil
}

// Validate validates a new runner.
func (rs *runnerStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	runner, ok := obj.(*corev1.Runner)
	if !ok {
		return field.ErrorList{}
	}
	return corevalidation.ValidateRunner(runner)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (rs *runnerStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	var warnings []string
	newRunner := obj.(*corev1.Runner)

	if msgs := utilvalidation.IsDNS1123Label(newRunner.Name); len(msgs) != 0 {
		warnings = append(warnings, fmt.Sprintf("metadata.name: this is used in Runner names and hostnames, which can result in surprising behavior; a DNS label is recommended: %v", msgs))
	}
	return warnings
}

// Canonicalize normalizes the object after validation.
func (rs *runnerStrategy) Canonicalize(obj runtime.Object) {
}

func (rs *runnerStrategy) AllowUnconditionalUpdate() bool {
	return true
}

// AllowCreateOnUpdate is false for runners; this means a POST is needed to create one.
func (rs *runnerStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (rs *runnerStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	runner := obj.(*corev1.Runner)
	oldRunner := old.(*corev1.Runner)

	validationErrorList := corevalidation.ValidateRunner(runner)
	updateErrorList := corevalidation.ValidateRunnerUpdate(runner, oldRunner)
	return append(validationErrorList, updateErrorList...)
}

// WarningsOnUpdate returns warnings for the given update.
func (rs *runnerStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	var warnings []string
	newRunner := obj.(*corev1.Runner)
	oldRunner := old.(*corev1.Runner)

	if newRunner.Generation != oldRunner.Generation {
	}
	return warnings
}

type runnerStatusStrategy struct {
	*runnerStrategy
}

func createStatusStrategy(strategy *runnerStrategy) *runnerStatusStrategy {
	return &runnerStatusStrategy{strategy}
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (rs *runnerStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"core/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}
}

func (rs *runnerStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newRunner := obj.(*corev1.Runner)
	oldRunner := old.(*corev1.Runner)
	newRunner.Status.DeepCopyInto(&oldRunner.Status)
}

func (rs *runnerStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	newRunner := obj.(*corev1.Runner)
	oldRunner := old.(*corev1.Runner)
	return corevalidation.ValidateRunnerUpdateStatus(newRunner, oldRunner)
}

// WarningsOnUpdate returns warnings for the given update.
func (rs *runnerStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// ProcessToSelectableFields returns a field set that represents the object for matching purposes.
func ProcessToSelectableFields(runner *corev1.Runner) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&runner.ObjectMeta, false)
	specificFieldsSet := fields.Set{}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	runner, ok := obj.(*corev1.Runner)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a runner.")
	}
	return labels.Set(runner.ObjectMeta.Labels), ProcessToSelectableFields(runner), nil
}

// MatchRunner is the filter used by the generic etcd backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func MatchRunner(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}
