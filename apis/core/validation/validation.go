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

// ValidateNamespaceName can be used to check whether the given namespace name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateNamespaceName = apimachineryvalidation.ValidateNamespaceName

// ValidateRunnerName can be used to check whether the given runner name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateRunnerName = apimachineryvalidation.NameIsDNSSubdomain

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

// ValidateRunner tests if required fields are set.
func ValidateRunner(runner *corev1.Runner) field.ErrorList {
	allErrs := ValidateObjectMeta(&runner.ObjectMeta, false, ValidateRunnerName, field.NewPath("metadata"))
	return allErrs
}

// ValidateRunnerUpdate validates an update to a Runner and returns an ErrorList with any errors.
func ValidateRunnerUpdate(runner, oldRunner *corev1.Runner) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&runner.ObjectMeta, &oldRunner.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRunnerSpecUpdate(runner.Spec, oldRunner.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateRunnerSpecUpdate validates an update to a RunnerSpec and returns an ErrorList with any errors.
func ValidateRunnerSpecUpdate(spec, oldSpec corev1.RunnerSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateRunnerUpdateStatus validates an update to the status of a Runner and returns an ErrorList with any errors.
func ValidateRunnerUpdateStatus(runner, oldRunner *corev1.Runner) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&runner.ObjectMeta, &oldRunner.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRunnerStatusUpdate(runner, oldRunner)...)
	return allErrs
}

// ValidateRunnerStatusUpdate validates an update to a RunnerStatus and returns an ErrorList with any errors.
func ValidateRunnerStatusUpdate(runner, oldRunner *corev1.Runner) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateRunnerStatus(runner, statusFld)...)

	return allErrs
}

// validateRunnerStatus validates a RunnerStatus and returns an ErrorList with any errors.
func validateRunnerStatus(runner *corev1.Runner, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateRegionName can be used to check whether the given region name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateRegionName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateRegion tests if required fields are set.
func ValidateRegion(region *corev1.Region) field.ErrorList {
	allErrs := ValidateObjectMeta(&region.ObjectMeta, false, ValidateRegionName, field.NewPath("metadata"))
	return allErrs
}

// ValidateRegionUpdate validates an update to a Region and returns an ErrorList with any errors.
func ValidateRegionUpdate(region, oldRegion *corev1.Region) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&region.ObjectMeta, &oldRegion.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRegionSpecUpdate(region.Spec, oldRegion.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateRegionSpecUpdate validates an update to a RegionSpec and returns an ErrorList with any errors.
func ValidateRegionSpecUpdate(spec, oldSpec corev1.RegionSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateRegionUpdateStatus validates an update to the status of a Region and returns an ErrorList with any errors.
func ValidateRegionUpdateStatus(region, oldRegion *corev1.Region) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&region.ObjectMeta, &oldRegion.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRegionStatusUpdate(region, oldRegion)...)
	return allErrs
}

// ValidateRegionStatusUpdate validates an update to a RegionStatus and returns an ErrorList with any errors.
func ValidateRegionStatusUpdate(region, oldRegion *corev1.Region) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateRegionStatus(region, statusFld)...)

	return allErrs
}

// validateRegionStatus validates a RegionStatus and returns an ErrorList with any errors.
func validateRegionStatus(region *corev1.Region, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateDefinition tests if required fields are set.
func ValidateDefinition(definition *corev1.Definition) field.ErrorList {
	allErrs := ValidateObjectMeta(&definition.ObjectMeta, true, ValidateDefinitionName, field.NewPath("metadata"))
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

// ValidateProcessName can be used to check whether the given processInstance name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateProcessName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateProcess tests if required fields are set.
func ValidateProcess(processInstance *corev1.Process) field.ErrorList {
	allErrs := ValidateObjectMeta(&processInstance.ObjectMeta, false, ValidateProcessName, field.NewPath("metadata"))
	return allErrs
}

// ValidateProcessUpdate validates an update to a Process and returns an ErrorList with any errors.
func ValidateProcessUpdate(processInstance, oldProcess *corev1.Process) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&processInstance.ObjectMeta, &oldProcess.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateProcessSpecUpdate(processInstance.Spec, oldProcess.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateProcessSpecUpdate validates an update to a ProcessSpec and returns an ErrorList with any errors.
func ValidateProcessSpecUpdate(spec, oldSpec corev1.ProcessSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateProcessUpdateStatus validates an update to the status of a Process and returns an ErrorList with any errors.
func ValidateProcessUpdateStatus(processInstance, oldProcess *corev1.Process) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&processInstance.ObjectMeta, &oldProcess.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateProcessStatusUpdate(processInstance, oldProcess)...)
	return allErrs
}

// ValidateProcessStatusUpdate validates an update to a ProcessStatus and returns an ErrorList with any errors.
func ValidateProcessStatusUpdate(processInstance, oldProcess *corev1.Process) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateProcessStatus(processInstance, statusFld)...)

	return allErrs
}

// validateProcessStatus validates a ProcessStatus and returns an ErrorList with any errors.
func validateProcessStatus(processInstance *corev1.Process, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateNamespace tests if required fields are set.
func ValidateNamespace(namespace *corev1.Namespace) field.ErrorList {
	allErrs := ValidateObjectMeta(&namespace.ObjectMeta, false, ValidateNamespaceName, field.NewPath("metadata"))
	for i := range namespace.Spec.Finalizers {
		allErrs = append(allErrs, validateFinalizerName(string(namespace.Spec.Finalizers[i]), field.NewPath("spec", "finalizers"))...)
	}
	return allErrs
}

// ValidateNamespaceUpdate tests to make sure a namespace update can be applied.
func ValidateNamespaceUpdate(newNamespace *corev1.Namespace, oldNamespace *corev1.Namespace) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&newNamespace.ObjectMeta, &oldNamespace.ObjectMeta, field.NewPath("metadata"))
	return allErrs
}

// ValidateNamespaceStatusUpdate tests to see if the update is legal for an end user to make.
func ValidateNamespaceStatusUpdate(newNamespace, oldNamespace *corev1.Namespace) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&newNamespace.ObjectMeta, &oldNamespace.ObjectMeta, field.NewPath("metadata"))
	if newNamespace.DeletionTimestamp.IsZero() {
		if newNamespace.Status.Phase != corev1.NamespaceActive {
			allErrs = append(allErrs, field.Invalid(field.NewPath("status", "Phase"), newNamespace.Status.Phase, "may only be 'Active' if `deletionTimestamp` is empty"))
		}
	} else {
		if newNamespace.Status.Phase != corev1.NamespaceTerminating {
			allErrs = append(allErrs, field.Invalid(field.NewPath("status", "Phase"), newNamespace.Status.Phase, "may only be 'Terminating' if `deletionTimestamp` is not empty"))
		}
	}
	return allErrs
}

// ValidateNamespaceFinalizeUpdate tests to see if the update is legal for an end user to make.
func ValidateNamespaceFinalizeUpdate(newNamespace, oldNamespace *corev1.Namespace) field.ErrorList {
	allErrs := ValidateObjectMetaUpdate(&newNamespace.ObjectMeta, &oldNamespace.ObjectMeta, field.NewPath("metadata"))

	fldPath := field.NewPath("spec", "finalizers")
	for i := range newNamespace.Spec.Finalizers {
		idxPath := fldPath.Index(i)
		allErrs = append(allErrs, validateFinalizerName(string(newNamespace.Spec.Finalizers[i]), idxPath)...)
	}
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
