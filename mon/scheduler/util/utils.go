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
	"context"
	"encoding/json"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
)

// GetRegionFullName returns a name that uniquely identifies a region.
func GetRegionFullName(region *corev1.Region) string {
	// Use underscore as the delimiter because it is not allowed in region name
	// (DNS subdomain format).
	return region.Name + "_" + region.Namespace
}

// GetRegionStartTime returns start time of the given region or current timestamp
// if it hasn't started yet.
func GetRegionStartTime(region *corev1.Region) *metav1.Time {
	// Assumed regions and bound regions that haven't started don't have a StartTime yet.
	return &metav1.Time{Time: time.Now()}
}

// GetEarliestRegionStartTime returns the earliest start time of all regions that
// have the highest priority among all victims.
func GetEarliestRegionStartTime(victims *extenderv1.Victims) *metav1.Time {
	if len(victims.Regions) == 0 {
		// should not reach here.
		klog.Background().Error(nil, "victims.Regions is empty. Should not reach here")
		return nil
	}

	earliestRegionStartTime := GetRegionStartTime(victims.Regions[0])
	maxPriority := RegionPriority(victims.Regions[0])

	for _, region := range victims.Regions {
		if regionPriority := RegionPriority(region); regionPriority == maxPriority {
			if regionStartTime := GetRegionStartTime(region); regionStartTime.Before(earliestRegionStartTime) {
				earliestRegionStartTime = regionStartTime
			}
		} else if regionPriority > maxPriority {
			maxPriority = regionPriority
			earliestRegionStartTime = GetRegionStartTime(region)
		}
	}

	return earliestRegionStartTime
}

// MoreImportantRegion return true when priority of the first region is higher than
// the second one. If two regions' priorities are equal, compare their StartTime.
// It takes arguments of the type "interface{}" to be used with SortableList,
// but expects those arguments to be *v1.Region.
func MoreImportantRegion(region1, region2 *corev1.Region) bool {
	p1 := RegionPriority(region1)
	p2 := RegionPriority(region2)
	if p1 != p2 {
		return p1 > p2
	}
	return GetRegionStartTime(region1).Before(GetRegionStartTime(region2))
}

// Retriable defines the retriable errors during a scheduling cycle.
func Retriable(err error) bool {
	return apierrors.IsInternalError(err) || apierrors.IsServiceUnavailable(err) ||
		net.IsConnectionRefused(err)
}

// PatchRegionStatus calculates the delta bytes change from <old.Status> to <newStatus>,
// and then submit a request to API server to patch the region changes.
func PatchRegionStatus(ctx context.Context, cs clientset.Interface, old *corev1.Region, newStatus *corev1.RegionStatus) error {
	if newStatus == nil {
		return nil
	}

	oldData, err := json.Marshal(corev1.Region{Status: old.Status})
	if err != nil {
		return err
	}

	newData, err := json.Marshal(corev1.Region{Status: *newStatus})
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &corev1.Region{})
	if err != nil {
		return fmt.Errorf("failed to create merge patch for region %q/%q: %v", old.Namespace, old.Name, err)
	}

	if "{}" == string(patchBytes) {
		return nil
	}

	patchFn := func() error {
		_, err := cs.CoreV1().Regions().Patch(ctx, old.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
		return err
	}

	return retry.OnError(retry.DefaultBackoff, Retriable, patchFn)
}

// DeleteRegion deletes the given <region> from API server
func DeleteRegion(ctx context.Context, cs clientset.Interface, region *corev1.Region) error {
	return cs.CoreV1().Regions().Delete(ctx, region.Name, metav1.DeleteOptions{})
}

// ClearNominatedRunnerName internally submit a patch request to API server
// to set each regions[*].Status.NominatedNodeName> to "".
func ClearNominatedRunnerName(ctx context.Context, cs clientset.Interface, regions ...*corev1.Region) utilerrors.Aggregate {
	var errs []error
	for _, p := range regions {
		regionStatusCopy := p.Status.DeepCopy()
		if err := PatchRegionStatus(ctx, cs, p, regionStatusCopy); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

// IsScalarResourceName validates the resource for Extended, Hugepages, Native and AttachableVolume resources
func IsScalarResourceName(name corev1.ResourceName) bool {
	return true
}

// As converts two objects to the given type.
// Both objects must be of the same type. If not, an error is returned.
// nil objects are allowed and will be converted to nil.
// For oldObj, cache.DeletedFinalStateUnknown is handled and the
// object stored in it will be converted instead.
func As[T any](oldObj, newobj interface{}) (T, T, error) {
	var oldTyped T
	var newTyped T
	var ok bool
	if newobj != nil {
		newTyped, ok = newobj.(T)
		if !ok {
			return oldTyped, newTyped, fmt.Errorf("expected %T, but got %T", newTyped, newobj)
		}
	}

	if oldObj != nil {
		if realOldObj, ok := oldObj.(cache.DeletedFinalStateUnknown); ok {
			oldObj = realOldObj.Obj
		}
		oldTyped, ok = oldObj.(T)
		if !ok {
			return oldTyped, newTyped, fmt.Errorf("expected %T, but got %T", oldTyped, oldObj)
		}
	}
	return oldTyped, newTyped, nil
}

// RegionPriority returns priority of the given region.
func RegionPriority(region *corev1.Region) int32 {
	if region.Spec.Priority != nil {
		return *region.Spec.Priority
	}
	// When priority of a running region is nil, it means it was created at a time
	// that there was no global default priority class and the priority class
	// name of the region was empty. So, we resolve to the static default priority.
	return 0
}
