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

package resource

import (
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// RegionResourcesOptions controls the behavior of RegionRequests and RegionLimits.
type RegionResourcesOptions struct {
	// Reuse, if provided will be reused to accumulate resources and returned by the RegionRequests or RegionLimits
	// functions. All existing values in Reuse will be lost.
	Reuse corev1.ResourceList
	// InPlaceRegionVerticalScalingEnabled indicates that the in-place region vertical scaling feature gate is enabled.
	InPlaceRegionVerticalScalingEnabled bool
	// ExcludeOverhead controls if region overhead is excluded from the calculation.
	ExcludeOverhead bool
	// NonMissingRegionRequests if provided will replace any missing container level requests for the specified resources
	// with the given values.  If the requests for those resources are explicitly set, even if zero, they will not be modified.
	NonMissingRegionRequests corev1.ResourceList
}

// RegionRequests computes the region requests per the RegionResourcesOptions supplied. If RegionResourcesOptions is nil, then
// the requests are returned including region overhead. The computation is part of the API and must be reviewed
// as an API change.
func RegionRequests(region *corev1.Region, opts RegionResourcesOptions) corev1.ResourceList {
	// attempt to reuse the maps if passed, or allocate otherwise
	reqs := reuseOrClearResourceList(opts.Reuse)

	return reqs
}

// applyNonMissing will return a copy of the given resource list with any missing values replaced by the nonMissing values
func applyNonMissing(reqs corev1.ResourceList, nonMissing corev1.ResourceList) corev1.ResourceList {
	cp := corev1.ResourceList{}
	for k, v := range reqs {
		cp[k] = v.DeepCopy()
	}

	for k, v := range nonMissing {
		if _, found := reqs[k]; !found {
			rk := cp[k]
			rk.Add(v)
			cp[k] = rk
		}
	}
	return cp
}

// RegionLimits computes the region limits per the RegionResourcesOptions supplied. If RegionResourcesOptions is nil, then
// the limits are returned including region overhead for any non-zero limits. The computation is part of the API and must be reviewed
// as an API change.
func RegionLimits(region *corev1.Region, opts RegionResourcesOptions) corev1.ResourceList {
	// attempt to reuse the maps if passed, or allocate otherwise
	limits := reuseOrClearResourceList(opts.Reuse)

	return limits
}

// reuseOrClearResourceList is a helper for avoiding excessive allocations of
// resource lists within the inner loop of resource calculations.
func reuseOrClearResourceList(reuse corev1.ResourceList) corev1.ResourceList {
	if reuse == nil {
		return make(corev1.ResourceList, 4)
	}
	for k := range reuse {
		delete(reuse, k)
	}
	return reuse
}
