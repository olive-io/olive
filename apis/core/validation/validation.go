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

package validation

import (
	"strings"

	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/olive-io/olive/apis/core/helper"
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// ValidateNameFunc validates that the provided name is valid for a given resource type.
// Not all resources have the same validation rules for names. Prefix is true
// if the name will have a value appended to it.  If the name is not valid,
// this returns a list of descriptions of individual characteristics of the
// value that were not valid.  Otherwise this returns an empty list or nil.
type ValidateNameFunc apimachineryvalidation.ValidateNameFunc

// ValidateDefinitionName can be used to check whether the given definitions name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateDefinitionName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateObjectMeta validates an object's metadata on creation. It expects that name generation has already
// been performed.
// It doesn't return an error for rootscoped resources with namespace, because namespace should already be cleared before.
func ValidateObjectMeta(meta *metav1.ObjectMeta, requiresNamespace bool, nameFn ValidateNameFunc, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateObjectMeta(meta, requiresNamespace, apimachineryvalidation.ValidateNameFunc(nameFn), fldPath)
	// run additional checks for the finalizer name
	for i := range meta.Finalizers {
		allErrs = append(allErrs, validateKubeFinalizerName(string(meta.Finalizers[i]), fldPath.Child("finalizers").Index(i))...)
	}
	return allErrs
}

// ValidateObjectMetaUpdate validates an object's metadata when updated
func ValidateObjectMetaUpdate(newMeta, oldMeta *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateObjectMetaUpdate(newMeta, oldMeta, fldPath)
	// run additional checks for the finalizer name
	for i := range newMeta.Finalizers {
		allErrs = append(allErrs, validateKubeFinalizerName(string(newMeta.Finalizers[i]), fldPath.Child("finalizers").Index(i))...)
	}

	return allErrs
}

// ValidateDefinition tests if required fields are set.
func ValidateDefinition(definition *corev1.Definition) field.ErrorList {
	allErrs := ValidateObjectMeta(&definition.ObjectMeta, false, ValidateDefinitionName, field.NewPath("metadata"))
	return allErrs
}

// ValidateDefinitionUpdate validates an update to a Definition and returns an ErrorList with any errors.
func ValidateDefinitionUpdate(definition, oldDefinition *corev1.Definition) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&definition.ObjectMeta, &oldDefinition.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDefinitionSpecUpdate(definition.Spec, oldDefinition.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateDefinitionSpecUpdate validates an update to a DefinitionSpec and returns an ErrorList with any errors.
func ValidateDefinitionSpecUpdate(spec, oldSpec corev1.DefinitionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateDefinitionUpdateStatus validates an update to the status of a Definition and returns an ErrorList with any errors.
func ValidateDefinitionUpdateStatus(definition, oldDefinition *corev1.Definition) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&definition.ObjectMeta, &oldDefinition.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateDefinitionStatusUpdate(definition, oldDefinition)...)
	return allErrs
}

// ValidateDefinitionStatusUpdate validates an update to a DefinitionStatus and returns an ErrorList with any errors.
func ValidateDefinitionStatusUpdate(definition, oldDefinition *corev1.Definition) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateDefinitionStatus(definition, statusFld)...)

	return allErrs
}

// validateDefinitionStatus validates a DefinitionStatus and returns an ErrorList with any errors.
func validateDefinitionStatus(definition *corev1.Definition, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateProcessInstanceName can be used to check whether the given processInstance name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateProcessInstanceName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateProcessInstance tests if required fields are set.
func ValidateProcessInstance(processInstance *corev1.ProcessInstance) field.ErrorList {
	allErrs := ValidateObjectMeta(&processInstance.ObjectMeta, false, ValidateProcessInstanceName, field.NewPath("metadata"))
	return allErrs
}

// ValidateProcessInstanceUpdate validates an update to a ProcessInstance and returns an ErrorList with any errors.
func ValidateProcessInstanceUpdate(processInstance, oldProcessInstance *corev1.ProcessInstance) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&processInstance.ObjectMeta, &oldProcessInstance.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateProcessInstanceSpecUpdate(processInstance.Spec, oldProcessInstance.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateProcessInstanceSpecUpdate validates an update to a ProcessInstanceSpec and returns an ErrorList with any errors.
func ValidateProcessInstanceSpecUpdate(spec, oldSpec corev1.ProcessInstanceSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateProcessInstanceUpdateStatus validates an update to the status of a ProcessInstance and returns an ErrorList with any errors.
func ValidateProcessInstanceUpdateStatus(processInstance, oldProcessInstance *corev1.ProcessInstance) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&processInstance.ObjectMeta, &oldProcessInstance.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateProcessInstanceStatusUpdate(processInstance, oldProcessInstance)...)
	return allErrs
}

// ValidateProcessInstanceStatusUpdate validates an update to a ProcessInstanceStatus and returns an ErrorList with any errors.
func ValidateProcessInstanceStatusUpdate(processInstance, oldProcessInstance *corev1.ProcessInstance) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateProcessInstanceStatus(processInstance, statusFld)...)

	return allErrs
}

// validateProcessInstanceStatus validates a ProcessInstanceStatus and returns an ErrorList with any errors.
func validateProcessInstanceStatus(processInstance *corev1.ProcessInstance, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// Validate finalizer names
func validateFinalizerName(stringValue string, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateFinalizerName(stringValue, fldPath)
	allErrs = append(allErrs, validateKubeFinalizerName(stringValue, fldPath)...)
	return allErrs
}

// validateKubeFinalizerName checks for "standard" names of legacy finalizer
func validateKubeFinalizerName(stringValue string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(strings.Split(stringValue, "/")) == 1 {
		if !helper.IsStandardFinalizerName(stringValue) {
			return append(allErrs, field.Invalid(fldPath, stringValue, "name is neither a standard finalizer name nor is it fully qualified"))
		}
	}

	return allErrs
}
