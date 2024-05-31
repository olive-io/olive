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

package helper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/olive-io/olive/apis/core"
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// IsHugePageResourceName returns true if the resource name has the huge page
// resource prefix.
func IsHugePageResourceName(name corev1.ResourceName) bool {
	return strings.HasPrefix(string(name), corev1.ResourceHugePagesPrefix)
}

// IsHugePageResourceValueDivisible returns true if the resource value of storage is
// integer multiple of page size.
func IsHugePageResourceValueDivisible(name corev1.ResourceName, quantity resource.Quantity) bool {
	pageSize, err := HugePageSizeFromResourceName(name)
	if err != nil {
		return false
	}

	if pageSize.Sign() <= 0 || pageSize.MilliValue()%int64(1000) != int64(0) {
		return false
	}

	return quantity.Value()%pageSize.Value() == 0
}

// HugePageResourceName returns a ResourceName with the canonical hugepage
// prefix prepended for the specified page size.  The page size is converted
// to its canonical representation.
func HugePageResourceName(pageSize resource.Quantity) core.ResourceName {
	return core.ResourceName(fmt.Sprintf("%s%s", core.ResourceHugePagesPrefix, pageSize.String()))
}

// HugePageSizeFromResourceName returns the page size for the specified huge page
// resource name.  If the specified input is not a valid huge page resource name
// an error is returned.
func HugePageSizeFromResourceName(name corev1.ResourceName) (resource.Quantity, error) {
	if !IsHugePageResourceName(name) {
		return resource.Quantity{}, fmt.Errorf("resource name: %s is an invalid hugepage name", name)
	}
	pageSize := strings.TrimPrefix(string(name), core.ResourceHugePagesPrefix)
	return resource.ParseQuantity(pageSize)
}

// Semantic can do semantic deep equality checks for core objects.
// Example: apiequality.Semantic.DeepEqual(aDefinition, aPodWithNonNilButEmptyMaps) == true
var Semantic = conversion.EqualitiesOrDie(
	func(a, b resource.Quantity) bool {
		// Ignore formatting, only care that numeric value stayed the same.
		// TODO: if we decide it's important, it should be safe to start comparing the format.
		//
		// Uninitialized quantities are equivalent to 0 quantities.
		return a.Cmp(b) == 0
	},
	func(a, b metav1.MicroTime) bool {
		return a.UTC() == b.UTC()
	},
	func(a, b metav1.Time) bool {
		return a.UTC() == b.UTC()
	},
	func(a, b labels.Selector) bool {
		return a.String() == b.String()
	},
	func(a, b fields.Selector) bool {
		return a.String() == b.String()
	},
)

var standardContainerResources = sets.New(
	corev1.ResourceCPU,
	corev1.ResourceMemory,
	corev1.ResourceEphemeralStorage,
)

// IsStandardContainerResourceName returns true if the container can make a resource request
// for the specified resource
func IsStandardContainerResourceName(name corev1.ResourceName) bool {
	return standardContainerResources.Has(name) || IsHugePageResourceName(name)
}

// IsNativeResource returns true if the resource name is in the
// *kubernetes.io/ namespace. Partially-qualified (unprefixed) names are
// implicitly in the kubernetes.io/ namespace.
func IsNativeResource(name corev1.ResourceName) bool {
	return !strings.Contains(string(name), "/") ||
		strings.Contains(string(name), core.ResourceDefaultNamespacePrefix)
}

// IsOvercommitAllowed returns true if the resource is in the default
// namespace and is not hugepages.
func IsOvercommitAllowed(name corev1.ResourceName) bool {
	return IsNativeResource(name) &&
		!IsHugePageResourceName(name)
}

var standardQuotaResources = sets.New(
	core.ResourceCPU,
	core.ResourceMemory,
	core.ResourceEphemeralStorage,
)

var standardFinalizers = sets.New(
	string(core.FinalizerOlive),
	metav1.FinalizerOrphanDependents,
	metav1.FinalizerDeleteDependents,
)

// IsStandardFinalizerName checks if the input string is a standard finalizer name
func IsStandardFinalizerName(str string) bool {
	return standardFinalizers.Has(str)
}

// IsExtendedResourceName returns true if:
// 1. the resource name is not in the default namespace;
// 2. resource name does not have "requests." prefix,
// to avoid confusion with the convention in quota
// 3. it satisfies the rules in IsQualifiedName() after converted into quota resource name
func IsExtendedResourceName(name corev1.ResourceName) bool {
	if IsNativeResource(name) || strings.HasPrefix(string(name), corev1.DefaultResourceRequestsPrefix) {
		return false
	}
	// Ensure it satisfies the rules in IsQualifiedName() after converted into quota resource name
	nameForQuota := fmt.Sprintf("%s%s", corev1.DefaultResourceRequestsPrefix, string(name))
	if errs := validation.IsQualifiedName(nameForQuota); len(errs) != 0 {
		return false
	}
	return true
}
