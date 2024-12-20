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

package namespace

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apistorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	"github.com/olive-io/olive/apis"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	corevalidation "github.com/olive-io/olive/apis/core/validation"
)

// namespaceStrategy implements behavior for Namespaces
type namespaceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Namespace
// objects via the REST API.
var Strategy = namespaceStrategy{apis.Scheme, names.SimpleNameGenerator}

// NamespaceScoped is false for namespaces.
func (namespaceStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (namespaceStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (namespaceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	// on create, status is active
	namespace := obj.(*corev1.Namespace)
	namespace.Status = corev1.NamespaceStatus{
		Phase: corev1.NamespaceActive,
	}
	// on create, we require the kubernetes value
	// we cannot use this in defaults conversion because we let it get removed over life of object
	hasKubeFinalizer := false
	for i := range namespace.Spec.Finalizers {
		if namespace.Spec.Finalizers[i] == corev1.FinalizerOlive {
			hasKubeFinalizer = true
			break
		}
	}
	if !hasKubeFinalizer {
		if len(namespace.Spec.Finalizers) == 0 {
			namespace.Spec.Finalizers = []corev1.FinalizerName{corev1.FinalizerOlive}
		} else {
			namespace.Spec.Finalizers = append(namespace.Spec.Finalizers, corev1.FinalizerOlive)
		}
	}
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (namespaceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNamespace := obj.(*corev1.Namespace)
	oldNamespace := old.(*corev1.Namespace)
	newNamespace.Spec.Finalizers = oldNamespace.Spec.Finalizers
	newNamespace.Status = oldNamespace.Status
}

// Validate validates a new namespace.
func (namespaceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	namespace := obj.(*corev1.Namespace)
	return corevalidation.ValidateNamespace(namespace)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (namespaceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// Canonicalize normalizes the object after validation.
func (namespaceStrategy) Canonicalize(obj runtime.Object) {
	// Ensure the label matches the name for namespaces just created using GenerateName,
	// where the final name wasn't available for defaulting to make this change.
	// This code needs to be kept in sync with the implementation that exists
	// in Namespace defaulting (pkg/apis/core/v1)
	// Why this hook *and* defaults.go?
	//
	// CREATE:
	// Defaulting and PrepareForCreate happen before generateName is completed
	// (i.e. the name is not yet known). Validation happens after generateName
	// but should not modify objects. Canonicalize happens later, which makes
	// it the best hook for setting the label.
	//
	// UPDATE:
	// Defaulting and Canonicalize will both trigger with the full name.
	//
	// GET:
	// Only defaulting will be applied.
	ns := obj.(*corev1.Namespace)
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[v1.LabelMetadataName] = ns.Name
}

// AllowCreateOnUpdate is false for namespaces.
func (namespaceStrategy) AllowCreateOnUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user.
func (namespaceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	errorList := corevalidation.ValidateNamespace(obj.(*corev1.Namespace))
	return append(errorList, corevalidation.ValidateNamespaceUpdate(obj.(*corev1.Namespace), old.(*corev1.Namespace))...)
}

// WarningsOnUpdate returns warnings for the given update.
func (namespaceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (namespaceStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type namespaceStatusStrategy struct {
	namespaceStrategy
}

var StatusStrategy = namespaceStatusStrategy{Strategy}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (namespaceStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("spec"),
		),
	}
}

func (namespaceStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNamespace := obj.(*corev1.Namespace)
	oldNamespace := old.(*corev1.Namespace)
	newNamespace.Spec = oldNamespace.Spec
}

func (namespaceStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return corevalidation.ValidateNamespaceStatusUpdate(obj.(*corev1.Namespace), old.(*corev1.Namespace))
}

// WarningsOnUpdate returns warnings for the given update.
func (namespaceStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type namespaceFinalizeStrategy struct {
	namespaceStrategy
}

var FinalizeStrategy = namespaceFinalizeStrategy{Strategy}

func (namespaceFinalizeStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return corevalidation.ValidateNamespaceFinalizeUpdate(obj.(*corev1.Namespace), old.(*corev1.Namespace))
}

// WarningsOnUpdate returns warnings for the given update.
func (namespaceFinalizeStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (namespaceFinalizeStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{
		"v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (namespaceFinalizeStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newNamespace := obj.(*corev1.Namespace)
	oldNamespace := old.(*corev1.Namespace)
	newNamespace.Status = oldNamespace.Status
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	namespaceObj, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil, nil, fmt.Errorf("not a namespace")
	}
	return labels.Set(namespaceObj.Labels), NamespaceToSelectableFields(namespaceObj), nil
}

// MatchNamespace returns a generic matcher for a given label and field selector.
func MatchNamespace(label labels.Selector, field fields.Selector) apistorage.SelectionPredicate {
	return apistorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// NamespaceToSelectableFields returns a field set that represents the object
func NamespaceToSelectableFields(namespace *corev1.Namespace) fields.Set {
	objectMetaFieldsSet := generic.ObjectMetaFieldsSet(&namespace.ObjectMeta, false)
	specificFieldsSet := fields.Set{
		"status.phase": string(namespace.Status.Phase),
		// This is a bug, but we need to support it for backward compatibility.
		"name": namespace.Name,
	}
	return generic.MergeFieldsSets(objectMetaFieldsSet, specificFieldsSet)
}
