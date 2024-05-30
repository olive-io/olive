/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// Cache collects definitions' information and provides node-level aggregated information.
// It's intended for generic scheduler to do efficient lookup.
// Cache's operations are definition centric. It does incremental updates based on definition events.
// Definition events are sent via network. We don't have guaranteed delivery of all events:
// We use Reflector to list and watch from remote.
// Reflector might be slow and do a relist, which would lead to missing events.
//
// State Machine of a definition's events in scheduler's cache:
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
// Note that an assumed definition can expire, because if we haven't received Add event notifying us
// for a while, there might be some problems and we shouldn't keep the definition in cache anymore.
//
// Note that "Initial", "Expired", and "Deleted" definitions do not actually exist in cache.
// Based on existing use cases, we are making the following assumptions:
//   - No definition would be assumed twice
//   - A definition could be added without going through scheduler. In this case, we will see Add but not Assume event.
//   - If a definition wasn't added, it wouldn't be removed or updated.
//   - Both "Expired" and "Deleted" are valid end states. In case of some problems, e.g. network issue,
//     a definition might have changed its state (e.g. added and deleted) without delivering notification to the cache.
type Cache interface {
	// RunnerCount returns the number of nodes in the cache.
	// DO NOT use outside of tests.
	RunnerCount() int

	// DefinitionCount returns the number of definitions in the cache (including those from deleted nodes).
	// DO NOT use outside of tests.
	DefinitionCount() (int, error)

	// AssumeDefinition assumes a definition scheduled and aggregates the definition's information into its node.
	// The implementation also decides the policy to expire definition before being confirmed (receiving Add event).
	// After expiration, its information would be subtracted.
	AssumeDefinition(logger klog.Logger, definition *corev1.Definition) error

	// FinishBinding signals that cache for assumed definition can be expired
	FinishBinding(logger klog.Logger, definition *corev1.Definition) error

	// ForgetDefinition removes an assumed definition from cache.
	ForgetDefinition(logger klog.Logger, definition *corev1.Definition) error

	// AddDefinition either confirms a definition if it's assumed, or adds it back if it's expired.
	// If added back, the definition's information would be added again.
	AddDefinition(logger klog.Logger, definition *corev1.Definition) error

	// UpdateDefinition removes oldDefinition's information and adds newDefinition's information.
	UpdateDefinition(logger klog.Logger, oldDefinition, newDefinition *corev1.Definition) error

	// RemoveDefinition removes a definition. The definition's information would be subtracted from assigned node.
	RemoveDefinition(logger klog.Logger, definition *corev1.Definition) error

	// GetDefinition returns the definition from the cache with the same namespace and the
	// same name of the specified definition.
	GetDefinition(definition *corev1.Definition) (*corev1.Definition, error)

	// IsAssumedDefinition returns true if the definition is assumed and not expired.
	IsAssumedDefinition(definition *corev1.Definition) (bool, error)

	// AddRunner adds overall information about node.
	// It returns a clone of added RunnerInfo object.
	AddRunner(logger klog.Logger, node *monv1.Runner) *framework.RunnerInfo

	// UpdateRunner updates overall information about node.
	// It returns a clone of updated RunnerInfo object.
	UpdateRunner(logger klog.Logger, oldRunner, newRunner *monv1.Runner) *framework.RunnerInfo

	// RemoveRunner removes overall information about node.
	RemoveRunner(logger klog.Logger, node *monv1.Runner) error

	// UpdateSnapshot updates the passed infoSnapshot to the current contents of Cache.
	// The node info contains aggregated information of definitions scheduled (including assumed to be)
	// on this node.
	// The snapshot only includes Runners that are not deleted at the time this function is called.
	// nodeinfo.Runner() is guaranteed to be not nil for all the nodes in the snapshot.
	UpdateSnapshot(logger klog.Logger, nodeSnapshot *Snapshot) error

	// Dump produces a dump of the current cache.
	Dump() *Dump
}

// Dump is a dump of the cache state.
type Dump struct {
	AssumedDefinitions sets.Set[string]
	Runners            map[string]*framework.RunnerInfo
}
