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

package runnerresources

import (
	"context"

	"k8s.io/klog/v2"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	schedutil "github.com/olive-io/olive/mon/scheduler/util"
	resourcehelper "github.com/olive-io/olive/pkg/api/v1/resource"
)

// scorer is decorator for resourceAllocationScorer
type scorer func(args *config.RunnerResourcesFitArgs) *resourceAllocationScorer

// resourceAllocationScorer contains information to calculate resource allocation score.
type resourceAllocationScorer struct {
	Name string
	// used to decide whether to use Requested or NonZeroRequested for
	// cpu and memory.
	useRequested bool
	scorer       func(requested, allocable []int64) int64
	resources    []config.ResourceSpec
}

// score will use `scorer` function to calculate the score.
func (r *resourceAllocationScorer) score(
	ctx context.Context,
	region *corev1.Region,
	runnerInfo *framework.RunnerInfo,
	regionRequests []int64) (int64, *framework.Status) {
	logger := klog.FromContext(ctx)
	runner := runnerInfo.Runner()

	// resources not set, nothing scheduled,
	if len(r.resources) == 0 {
		return 0, framework.NewStatus(framework.Error, "resources not found")
	}

	requested := make([]int64, len(r.resources))
	allocatable := make([]int64, len(r.resources))
	for i := range r.resources {
		alloc, req := r.calculateResourceAllocatableRequest(logger, runnerInfo, corev1.ResourceName(r.resources[i].Name), regionRequests[i])
		// Only fill the extended resource entry when it's non-zero.
		if alloc == 0 {
			continue
		}
		allocatable[i] = alloc
		requested[i] = req
	}

	score := r.scorer(requested, allocatable)

	if loggerV := logger.V(10); loggerV.Enabled() { // Serializing these maps is costly.
		loggerV.Info("Listed internal info for allocatable resources, requested resources and score", "region",
			klog.KObj(region), "runner", klog.KObj(runner), "resourceAllocationScorer", r.Name,
			"allocatableResource", allocatable, "requestedResource", requested, "resourceScore", score,
		)
	}

	return score, nil
}

// calculateResourceAllocatableRequest returns 2 parameters:
// - 1st param: quantity of allocatable resource on the runner.
// - 2nd param: aggregated quantity of requested resource on the runner.
// Note: if it's an extended resource, and the region doesn't request it, (0, 0) is returned.
func (r *resourceAllocationScorer) calculateResourceAllocatableRequest(logger klog.Logger, runnerInfo *framework.RunnerInfo, resource corev1.ResourceName, regionRequest int64) (int64, int64) {
	requested := runnerInfo.NonZeroRequested
	if r.useRequested {
		requested = runnerInfo.Requested
	}

	// If it's an extended resource, and the region doesn't request it. We return (0, 0)
	// as an implication to bypass scoring on this resource.
	if regionRequest == 0 && schedutil.IsScalarResourceName(resource) {
		return 0, 0
	}
	switch resource {
	case corev1.ResourceCPU:
		return runnerInfo.Allocatable.MilliCPU, (requested.MilliCPU + regionRequest)
	case corev1.ResourceMemory:
		return runnerInfo.Allocatable.Memory, (requested.Memory + regionRequest)
	case corev1.ResourceEphemeralStorage:
		return runnerInfo.Allocatable.EphemeralStorage, (runnerInfo.Requested.EphemeralStorage + regionRequest)
	default:
		if _, exists := runnerInfo.Allocatable.ScalarResources[resource]; exists {
			return runnerInfo.Allocatable.ScalarResources[resource], (runnerInfo.Requested.ScalarResources[resource] + regionRequest)
		}
	}
	logger.V(10).Info("Requested resource is omitted for runner score calculation", "resourceName", resource)
	return 0, 0
}

// calculateRegionResourceRequest returns the total non-zero requests. If Overhead is defined for the region
// the Overhead is added to the result.
func (r *resourceAllocationScorer) calculateRegionResourceRequest(region *corev1.Region, resourceName corev1.ResourceName) int64 {

	opts := resourcehelper.RegionResourcesOptions{
		InPlaceRegionVerticalScalingEnabled: false,
	}
	if !r.useRequested {
		//opts.NonMissingContainerRequests = corev1.ResourceList{
		//	v1.ResourceCPU:    *resource.NewMilliQuantity(schedutil.DefaultMilliCPURequest, resource.DecimalSI),
		//	v1.ResourceMemory: *resource.NewQuantity(schedutil.DefaultMemoryRequest, resource.DecimalSI),
		//}
	}

	requests := resourcehelper.RegionRequests(region, opts)

	quantity := requests[resourceName]
	if resourceName == corev1.ResourceCPU {
		return quantity.MilliValue()
	}
	return quantity.Value()
}

func (r *resourceAllocationScorer) calculateRegionResourceRequestList(region *corev1.Region, resources []config.ResourceSpec) []int64 {
	regionRequests := make([]int64, len(resources))
	for i := range resources {
		regionRequests[i] = r.calculateRegionResourceRequest(region, corev1.ResourceName(resources[i].Name))
	}
	return regionRequests
}
