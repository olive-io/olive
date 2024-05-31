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

package util

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// For each of these resources, a container that doesn't request the resource explicitly
// will be treated as having requested the amount indicated below, for the purpose
// of computing priority only. This ensures that when scheduling zero-request pods, such
// pods will not all be scheduled to the node with the smallest in-use request,
// and that when scheduling regular pods, such pods will not see zero-request pods as
// consuming no resources whatsoever. We chose these values to be similar to the
// resources that we give to cluster addon pods (#10653). But they are pretty arbitrary.
// As described in #11713, we use request instead of limit to deal with resource requirements.
const (
	// DefaultMilliCPURequest defines default milli cpu request number.
	DefaultMilliCPURequest int64 = 100 // 0.1 core
	// DefaultMemoryRequest defines default memory request size.
	DefaultMemoryRequest int64 = 200 * 1024 * 1024 // 200 MB
)

// GetNonzeroRequests returns the default cpu in milli-cpu and memory in bytes resource requests if none is found or
// what is provided on the request.
func GetNonzeroRequests(requests *v1.ResourceList) (int64, int64) {
	cpu := GetRequestForResource(v1.ResourceCPU, requests, true)
	mem := GetRequestForResource(v1.ResourceMemory, requests, true)
	return cpu.MilliValue(), mem.Value()

}

// GetRequestForResource returns the requested values unless nonZero is true and there is no defined request
// for CPU and memory.
// If nonZero is true and the resource has no defined request for CPU or memory, it returns a default value.
func GetRequestForResource(resourceName v1.ResourceName, requests *v1.ResourceList, nonZero bool) resource.Quantity {
	if requests == nil {
		return resource.Quantity{}
	}
	switch resourceName {
	case v1.ResourceCPU:
		// Override if un-set, but not if explicitly set to zero
		if _, found := (*requests)[v1.ResourceCPU]; !found && nonZero {
			return *resource.NewMilliQuantity(DefaultMilliCPURequest, resource.DecimalSI)
		}
		return requests.Cpu().DeepCopy()
	case v1.ResourceMemory:
		// Override if un-set, but not if explicitly set to zero
		if _, found := (*requests)[v1.ResourceMemory]; !found && nonZero {
			return *resource.NewQuantity(DefaultMemoryRequest, resource.DecimalSI)
		}
		return requests.Memory().DeepCopy()
	default:
		quantity, found := (*requests)[resourceName]
		if !found {
			return resource.Quantity{}
		}
		return quantity.DeepCopy()
	}
}
