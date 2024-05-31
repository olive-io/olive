/*
Copyright 2019 The Kubernetes Authors.

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

package plugins

import (
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/defaultbinder"
	plfeature "github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/interregionaffinity"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/queuesort"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runneraffinity"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnername"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnerresources"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnerunschedulable"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/schedulinggates"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/tainttoleration"
	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
)

// NewInTreeRegistry builds the registry with all the in-tree plugins.
// A scheduler that runs out of tree plugins can register additional plugins
// through the WithFrameworkOutOfTreeRegistry option.
func NewInTreeRegistry() runtime.Registry {
	fts := plfeature.Features{
		//EnableRegionDisruptionConditions:   feature.DefaultFeatureGate.Enabled(features.RegionDisruptionConditions),
		//EnableInPlaceRegionVerticalScaling: feature.DefaultFeatureGate.Enabled(features.InPlaceRegionVerticalScaling),
	}

	registry := runtime.Registry{
		tainttoleration.Name:                   tainttoleration.New,
		runnername.Name:                        runnername.New,
		runneraffinity.Name:                    runneraffinity.New,
		runnerunschedulable.Name:               runnerunschedulable.New,
		runnerresources.Name:                   runtime.FactoryAdapter(fts, runnerresources.NewFit),
		runnerresources.BalancedAllocationName: runtime.FactoryAdapter(fts, runnerresources.NewBalancedAllocation),
		interregionaffinity.Name:               interregionaffinity.New,
		queuesort.Name:                         queuesort.New,
		defaultbinder.Name:                     defaultbinder.New,
		schedulinggates.Name:                   schedulinggates.New,
	}

	return registry
}
