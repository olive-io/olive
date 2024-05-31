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

package interregionaffinity

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/parallelize"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
)

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = names.InterRegionAffinity

var _ framework.PreFilterPlugin = &InterRegionAffinity{}
var _ framework.FilterPlugin = &InterRegionAffinity{}
var _ framework.PreScorePlugin = &InterRegionAffinity{}
var _ framework.ScorePlugin = &InterRegionAffinity{}
var _ framework.EnqueueExtensions = &InterRegionAffinity{}

// InterRegionAffinity is a plugin that checks inter pod affinity
type InterRegionAffinity struct {
	parallelizer parallelize.Parallelizer
	//args         config.InterRegionAffinityArgs
	sharedLister framework.SharedLister
	nsLister     listersv1.NamespaceLister
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *InterRegionAffinity) Name() string {
	return Name
}

// EventsToRegister returns the possible events that may make a failed Region
// schedulable
func (pl *InterRegionAffinity) EventsToRegister() []framework.ClusterEventWithHint {
	return []framework.ClusterEventWithHint{
		// All ActionType includes the following events:
		// - Delete. An unschedulable Region may fail due to violating an existing Region's anti-affinity constraints,
		// deleting an existing Region may make it schedulable.
		// - Update. Updating on an existing Region's labels (e.g., removal) may make
		// an unschedulable Region schedulable.
		// - Add. An unschedulable Region may fail due to violating pod-affinity constraints,
		// adding an assigned Region may make it schedulable.
		//
		// A note about UpdateNodeTaint event:
		// NodeAdd QueueingHint isn't always called because of the internal feature called preCheck.
		// As a common problematic scenario,
		// when a node is added but not ready, NodeAdd event is filtered out by preCheck and doesn't arrive.
		// In such cases, this plugin may miss some events that actually make pods schedulable.
		// As a workaround, we add UpdateNodeTaint event to catch the case.
		// We can remove UpdateNodeTaint when we remove the preCheck feature.
		// See: https://github.com/kubernetes/kubernetes/issues/110175
		{Event: framework.ClusterEvent{Resource: framework.Region, ActionType: framework.All}},
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.UpdateRunnerLabel | framework.UpdateRunnerTaint}},
	}
}

// New initializes a new plugin and returns it.
func New(_ context.Context, plArgs runtime.Object, h framework.Handle) (framework.Plugin, error) {
	if h.SnapshotSharedLister() == nil {
		return nil, fmt.Errorf("SnapshotSharedlister is nil")
	}
	//args, err := getArgs(plArgs)
	//if err != nil {
	//	return nil, err
	//}
	//if err := validation.ValidateInterRegionAffinityArgs(nil, &args); err != nil {
	//	return nil, err
	//}
	pl := &InterRegionAffinity{
		parallelizer: h.Parallelizer(),
		//args:         args,
		sharedLister: h.SnapshotSharedLister(),
		nsLister:     h.SharedInformerFactory().Core().V1().Namespaces().Lister(),
	}

	return pl, nil
}

//func getArgs(obj runtime.Object) (config.InterRegionAffinityArgs, error) {
//	ptr, ok := obj.(*config.InterRegionAffinityArgs)
//	if !ok {
//		return config.InterRegionAffinityArgs{}, fmt.Errorf("want args to be of type InterRegionAffinityArgs, got %T", obj)
//	}
//	return *ptr, nil
//}

// Updates Namespaces with the set of namespaces identified by NamespaceSelector.
// If successful, NamespaceSelector is set to nil.
// The assumption is that the term is for an incoming pod, in which case
// namespaceSelector is either unrolled into Namespaces (and so the selector
// is set to Nothing()) or is Empty(), which means match everything. Therefore,
// there when matching against this term, there is no need to lookup the existing
// pod's namespace labels to match them against term's namespaceSelector explicitly.
func (pl *InterRegionAffinity) mergeAffinityTermNamespacesIfNotEmpty(at *framework.AffinityTerm) error {
	if at.NamespaceSelector.Empty() {
		return nil
	}
	ns, err := pl.nsLister.List(at.NamespaceSelector)
	if err != nil {
		return err
	}
	for _, n := range ns {
		at.Namespaces.Insert(n.Name)
	}
	at.NamespaceSelector = labels.Nothing()
	return nil
}

// GetNamespaceLabelsSnapshot returns a snapshot of the labels associated with
// the namespace.
func GetNamespaceLabelsSnapshot(logger klog.Logger, ns string, nsLister listersv1.NamespaceLister) (nsLabels labels.Set) {
	podNS, err := nsLister.Get(ns)
	if err == nil {
		// Create and return snapshot of the labels.
		return labels.Merge(podNS.Labels, nil)
	}
	logger.V(3).Info("getting namespace, assuming empty set of namespace labels", "namespace", ns, "err", err)
	return
}
