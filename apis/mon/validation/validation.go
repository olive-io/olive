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
	apivalidation "github.com/olive-io/olive/apis/core/validation"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
)

// ValidateNameFunc validates that the provided name is valid for a given resource type.
// Not all resources have the same validation rules for names. Prefix is true
// if the name will have a value appended to it.  If the name is not valid,
// this returns a list of descriptions of individual characteristics of the
// value that were not valid.  Otherwise this returns an empty list or nil.
type ValidateNameFunc apimachineryvalidation.ValidateNameFunc

// ValidateRunnerName can be used to check whether the given runner name is valid.
// Prefix indicates this name will be used as part of generation, in which case
// trailing dashes are allowed.
var ValidateRunnerName = apimachineryvalidation.NameIsDNSSubdomain

// ValidateObjectMeta validates an object's metadata on creation. It expects that name generation has already
// been performed.
// It doesn't return an error for rootscoped resources with namespace, because namespace should already be cleared before.
// TODO: Remove calls to this method scattered in validations of specific resources, e.g., ValidatePodUpdate.
func ValidateObjectMeta(meta *metav1.ObjectMeta, requiresNamespace bool, nameFn ValidateNameFunc, fldPath *field.Path) field.ErrorList {
	allErrs := apimachineryvalidation.ValidateObjectMeta(meta, requiresNamespace, apimachineryvalidation.ValidateNameFunc(nameFn), fldPath)
	// run additional checks for the finalizer name
	for i := range meta.Finalizers {
		allErrs = append(allErrs, validateKubeFinalizerName(string(meta.Finalizers[i]), fldPath.Child("finalizers").Index(i))...)
	}
	return allErrs
}

// ValidateRunner tests if required fields are set.
func ValidateRunner(runner *monv1.Runner) field.ErrorList {
	allErrs := ValidateObjectMeta(&runner.ObjectMeta, false, ValidateRunnerName, field.NewPath("metadata"))
	return allErrs
}

// ValidateRunnerUpdate validates an update to a Runner and returns an ErrorList with any errors.
func ValidateRunnerUpdate(runner, oldRunner *monv1.Runner) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&runner.ObjectMeta, &oldRunner.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRunnerSpecUpdate(runner.Spec, oldRunner.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateRunnerSpecUpdate validates an update to a RunnerSpec and returns an ErrorList with any errors.
func ValidateRunnerSpecUpdate(spec, oldSpec monv1.RunnerSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// ValidateRunnerUpdateStatus validates an update to the status of a Runner and returns an ErrorList with any errors.
func ValidateRunnerUpdateStatus(runner, oldRunner *monv1.Runner) field.ErrorList {
	allErrs := apivalidation.ValidateObjectMetaUpdate(&runner.ObjectMeta, &oldRunner.ObjectMeta, field.NewPath("metadata"))
	allErrs = append(allErrs, ValidateRunnerStatusUpdate(runner, oldRunner)...)
	return allErrs
}

// ValidateRunnerStatusUpdate validates an update to a RunnerStatus and returns an ErrorList with any errors.
func ValidateRunnerStatusUpdate(runner, oldRunner *monv1.Runner) field.ErrorList {
	allErrs := field.ErrorList{}
	statusFld := field.NewPath("status")
	allErrs = append(allErrs, validateRunnerStatus(runner, statusFld)...)

	return allErrs
}

// validateRunnerStatus validates a RunnerStatus and returns an ErrorList with any errors.
func validateRunnerStatus(runner *monv1.Runner, fldPath *field.Path) field.ErrorList {
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
