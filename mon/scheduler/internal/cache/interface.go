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

package cache

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// Cache collects regions' information and provides node-level aggregated information.
// It's intended for generic scheduler to do efficient lookup.
// Cache's operations are region centric. It does incremental updates based on region events.
// Region events are sent via network. We don't have guaranteed delivery of all events:
// We use Reflector to list and watch from remote.
// Reflector might be slow and do a relist, which would lead to missing events.
//
// State Machine of a region's events in scheduler's cache:
//
//	+-------------------------------------------+  +----+
//	|                            Add            |  |    |
//	|                                           |  |    | Update
//	+      Assume                Add            v  v    |
//
// Initial +--------> Assumed +------------+---> Added <--+
//
//	^                +   +               |       +
//	|                |   |               |       |
//	|                |   |           Add |       | Remove
//	|                |   |               |       |
//	|                |   |               +       |
//	+----------------+   +-----------> Expired   +----> Deleted
//	      Forget             Expire
//
// Note that an assumed region can expire, because if we haven't received Add event notifying us
// for a while, there might be some problems and we shouldn't keep the region in cache anymore.
//
// Note that "Initial", "Expired", and "Deleted" regions do not actually exist in cache.
// Based on existing use cases, we are making the following assumptions:
//   - No region would be assumed twice
//   - A region could be added without going through scheduler. In this case, we will see Add but not Assume event.
//   - If a region wasn't added, it wouldn't be removed or updated.
//   - Both "Expired" and "Deleted" are valid end states. In case of some problems, e.g. network issue,
//     a region might have changed its state (e.g. added and deleted) without delivering notification to the cache.
type Cache interface {
	// RunnerCount returns the number of nodes in the cache.
	// DO NOT use outside of tests.
	RunnerCount() int

	// RegionCount returns the number of regions in the cache (including those from deleted nodes).
	// DO NOT use outside of tests.
	RegionCount() (int, error)

	// AssumeRegion assumes a region scheduled and aggregates the region's information into its node.
	// The implementation also decides the policy to expire region before being confirmed (receiving Add event).
	// After expiration, its information would be subtracted.
	AssumeRegion(logger klog.Logger, region *corev1.Region) error

	// FinishBinding signals that cache for assumed region can be expired
	FinishBinding(logger klog.Logger, region *corev1.Region) error

	// ForgetRegion removes an assumed region from cache.
	ForgetRegion(logger klog.Logger, region *corev1.Region) error

	// AddRegion either confirms a region if it's assumed, or adds it back if it's expired.
	// If added back, the region's information would be added again.
	AddRegion(logger klog.Logger, region *corev1.Region) error

	// UpdateRegion removes oldRegion's information and adds newRegion's information.
	UpdateRegion(logger klog.Logger, oldRegion, newRegion *corev1.Region) error

	// RemoveRegion removes a region. The region's information would be subtracted from assigned node.
	RemoveRegion(logger klog.Logger, region *corev1.Region) error

	// GetRegion returns the region from the cache with the same namespace and the
	// same name of the specified region.
	GetRegion(region *corev1.Region) (*corev1.Region, error)

	// IsAssumedRegion returns true if the region is assumed and not expired.
	IsAssumedRegion(region *corev1.Region) (bool, error)

	// AddRunner adds overall information about node.
	// It returns a clone of added RunnerInfo object.
	AddRunner(logger klog.Logger, node *corev1.Runner) *framework.RunnerInfo

	// UpdateRunner updates overall information about node.
	// It returns a clone of updated RunnerInfo object.
	UpdateRunner(logger klog.Logger, oldRunner, newRunner *corev1.Runner) *framework.RunnerInfo

	// RemoveRunner removes overall information about node.
	RemoveRunner(logger klog.Logger, node *corev1.Runner) error

	// UpdateSnapshot updates the passed infoSnapshot to the current contents of Cache.
	// The node info contains aggregated information of regions scheduled (including assumed to be)
	// on this node.
	// The snapshot only includes Runners that are not deleted at the time this function is called.
	// nodeinfo.Runner() is guaranteed to be not nil for all the nodes in the snapshot.
	UpdateSnapshot(logger klog.Logger, nodeSnapshot *Snapshot) error

	// Dump produces a dump of the current cache.
	Dump() *Dump
}

// Dump is a dump of the cache state.
type Dump struct {
	AssumedRegions sets.Set[string]
	Runners        map[string]*framework.RunnerInfo
}
