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
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	config "github.com/olive-io/olive/apis/config/v1"
	"github.com/olive-io/olive/apis/config/validation"
	v1helper "github.com/olive-io/olive/apis/core/helper"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
	schedutil "github.com/olive-io/olive/mon/scheduler/util"
	"github.com/olive-io/olive/pkg/api/v1/resource"
)

var _ framework.PreFilterPlugin = &Fit{}
var _ framework.FilterPlugin = &Fit{}
var _ framework.EnqueueExtensions = &Fit{}
var _ framework.PreScorePlugin = &Fit{}
var _ framework.ScorePlugin = &Fit{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = names.RunnerResourcesFit

	// preFilterStateKey is the key in CycleState to RunnerResourcesFit pre-computed data.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + Name

	// preScoreStateKey is the key in CycleState to RunnerResourcesFit pre-computed data for Scoring.
	preScoreStateKey = "PreScore" + Name
)

// runnerResourceStrategyTypeMap maps strategy to scorer implementation
var runnerResourceStrategyTypeMap = map[config.ScoringStrategyType]scorer{
	config.LeastAllocated: func(args *config.RunnerResourcesFitArgs) *resourceAllocationScorer {
		resources := args.ScoringStrategy.Resources
		return &resourceAllocationScorer{
			Name:      string(config.LeastAllocated),
			scorer:    leastResourceScorer(resources),
			resources: resources,
		}
	},
	config.MostAllocated: func(args *config.RunnerResourcesFitArgs) *resourceAllocationScorer {
		resources := args.ScoringStrategy.Resources
		return &resourceAllocationScorer{
			Name:      string(config.MostAllocated),
			scorer:    mostResourceScorer(resources),
			resources: resources,
		}
	},
	config.RequestedToCapacityRatio: func(args *config.RunnerResourcesFitArgs) *resourceAllocationScorer {
		resources := args.ScoringStrategy.Resources
		return &resourceAllocationScorer{
			Name:      string(config.RequestedToCapacityRatio),
			scorer:    requestedToCapacityRatioScorer(resources, args.ScoringStrategy.RequestedToCapacityRatio.Shape),
			resources: resources,
		}
	},
}

// Fit is a plugin that checks if a runner has sufficient resources.
type Fit struct {
	ignoredResources                   sets.Set[string]
	ignoredResourceGroups              sets.Set[string]
	enableInPlaceRegionVerticalScaling bool
	handle                             framework.Handle
	resourceAllocationScorer
}

// ScoreExtensions of the Score plugin.
func (f *Fit) ScoreExtensions() framework.ScoreExtensions {
	return nil
}

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	framework.Resource
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	// regionRequests have the same order as the resources defined in RunnerResourcesBalancedAllocationArgs.Resources,
	// same for other place we store a list like that.
	regionRequests []int64
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// PreScore calculates incoming region's resource requests and writes them to the cycle state used.
func (f *Fit) PreScore(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	state := &preScoreState{
		regionRequests: f.calculateRegionResourceRequestList(region, f.resources),
	}
	cycleState.Write(preScoreStateKey, state)
	return nil
}

func getPreScoreState(cycleState *framework.CycleState) (*preScoreState, error) {
	c, err := cycleState.Read(preScoreStateKey)
	if err != nil {
		return nil, fmt.Errorf("reading %q from cycleState: %w", preScoreStateKey, err)
	}

	s, ok := c.(*preScoreState)
	if !ok {
		return nil, fmt.Errorf("invalid PreScore state, got type %T", c)
	}
	return s, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (f *Fit) Name() string {
	return Name
}

// NewFit initializes a new plugin and returns it.
func NewFit(_ context.Context, plArgs runtime.Object, h framework.Handle, fts feature.Features) (framework.Plugin, error) {
	args, ok := plArgs.(*config.RunnerResourcesFitArgs)
	if !ok {
		return nil, fmt.Errorf("want args to be of type RunnerResourcesFitArgs, got %T", plArgs)
	}
	if err := validation.ValidateRunnerResourcesFitArgs(nil, args); err != nil {
		return nil, err
	}

	if args.ScoringStrategy == nil {
		return nil, fmt.Errorf("scoring strategy not specified")
	}

	strategy := args.ScoringStrategy.Type
	scorePlugin, exists := runnerResourceStrategyTypeMap[strategy]
	if !exists {
		return nil, fmt.Errorf("scoring strategy %s is not supported", strategy)
	}

	return &Fit{
		ignoredResources:                   sets.New(args.IgnoredResources...),
		ignoredResourceGroups:              sets.New(args.IgnoredResourceGroups...),
		enableInPlaceRegionVerticalScaling: fts.EnableInPlaceRegionVerticalScaling,
		handle:                             h,
		resourceAllocationScorer:           *scorePlugin(args),
	}, nil
}

// computeRegionResourceRequest returns a framework.Resource that covers the largest
// width in each resource dimension. Because init-containers run sequentially, we collect
// the max in each dimension iteratively. In contrast, we sum the resource vectors for
// regular containers since they run simultaneously.
//
// # The resources defined for Overhead should be added to the calculated Resource request sum
//
// Example:
//
// Region:
//
//	InitContainers
//	  IC1:
//	    CPU: 2
//	    Memory: 1G
//	  IC2:
//	    CPU: 2
//	    Memory: 3G
//	Containers
//	  C1:
//	    CPU: 2
//	    Memory: 1G
//	  C2:
//	    CPU: 1
//	    Memory: 1G
//
// Result: CPU: 3, Memory: 3G
func computeRegionResourceRequest(region *corev1.Region) *preFilterState {
	// region hasn't scheduled yet so we don't need to worry about InPlaceRegionVerticalScalingEnabled
	reqs := resource.RegionRequests(region, resource.RegionResourcesOptions{})
	result := &preFilterState{}
	result.SetMaxResource(reqs)
	return result
}

// PreFilter invoked at the prefilter extension point.
func (f *Fit) PreFilter(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	cycleState.Write(preFilterStateKey, computeRegionResourceRequest(region))
	return nil, nil
}

// PreFilterExtensions returns prefilter extensions, region add and remove.
func (f *Fit) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		// preFilterState doesn't exist, likely PreFilter wasn't invoked.
		return nil, fmt.Errorf("error reading %q from cycleState: %w", preFilterStateKey, err)
	}

	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to RunnerResourcesFit.preFilterState error", c)
	}
	return s, nil
}

// EventsToRegister returns the possible events that may make a Region
// failed by this plugin schedulable.
func (f *Fit) EventsToRegister() []framework.ClusterEventWithHint {
	regionActionType := framework.Delete
	if f.enableInPlaceRegionVerticalScaling {
		// If InPlaceRegionVerticalScaling (KEP 1287) is enabled, then RegionUpdate event should be registered
		// for this plugin since a Region update may free up resources that make other Regions schedulable.
		regionActionType |= framework.Update
	}
	return []framework.ClusterEventWithHint{
		{Event: framework.ClusterEvent{Resource: framework.Region, ActionType: regionActionType}, QueueingHintFn: f.isSchedulableAfterRegionChange},
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.Update}, QueueingHintFn: f.isSchedulableAfterRunnerChange},
	}
}

// isSchedulableAfterRegionChange is invoked whenever a region deleted or updated. It checks whether
// that change made a previously unschedulable region schedulable.
func (f *Fit) isSchedulableAfterRegionChange(logger klog.Logger, region *corev1.Region, oldObj, newObj interface{}) (framework.QueueingHint, error) {
	originalRegion, modifiedRegion, err := schedutil.As[*corev1.Region](oldObj, newObj)
	if err != nil {
		return framework.Queue, err
	}

	if modifiedRegion == nil {
		if len(originalRegion.Spec.RunnerNames) == 0 {
			logger.V(5).Info("the deleted region was unscheduled and it wouldn't make the unscheduled region schedulable", "region", klog.KObj(region), "deletedRegion", klog.KObj(originalRegion))
			return framework.QueueSkip, nil
		}
		logger.V(5).Info("another scheduled region was deleted, and it may make the unscheduled region schedulable", "region", klog.KObj(region), "deletedRegion", klog.KObj(originalRegion))
		return framework.Queue, nil
	}

	if !f.enableInPlaceRegionVerticalScaling {
		// If InPlaceRegionVerticalScaling (KEP 1287) is disabled, it cannot free up resources.
		logger.V(5).Info("another region was modified, but InPlaceRegionVerticalScaling is disabled, so it doesn't make the unscheduled region schedulable", "region", klog.KObj(region), "modifiedRegion", klog.KObj(modifiedRegion))
		return framework.QueueSkip, nil
	}

	// Modifications may or may not be relevant. We only care about modifications that
	// change the other region's resource request and the resource is also requested by the
	// region we are trying to schedule.
	if !f.isResourceScaleDown(region, originalRegion, modifiedRegion) {
		if loggerV := logger.V(10); loggerV.Enabled() {
			// Log more information.
			loggerV.Info("another Region got modified, but the modification isn't related to the resource request", "region", klog.KObj(region), "modifiedRegion", klog.KObj(modifiedRegion), "diff", cmp.Diff(originalRegion, modifiedRegion))
		} else {
			logger.V(5).Info("another Region got modified, but the modification isn't related to the resource request", "region", klog.KObj(region), "modifiedRegion", klog.KObj(modifiedRegion))
		}
		return framework.QueueSkip, nil
	}

	logger.V(5).Info("the max request resources of another scheduled region got reduced and it may make the unscheduled region schedulable", "region", klog.KObj(region), "modifiedRegion", klog.KObj(modifiedRegion))
	return framework.Queue, nil
}

// isResourceScaleDown checks whether the resource request of the modified region is less than the original region
// for the resources requested by the region we are trying to schedule.
func (f *Fit) isResourceScaleDown(targetRegion, originalOtherRegion, modifiedOtherRegion *corev1.Region) bool {
	if len(modifiedOtherRegion.Spec.RunnerNames) == 0 {
		// no resource is freed up whatever the region is modified.
		return false
	}

	// the other region was scheduled, so modification or deletion may free up some resources.
	originalMaxResourceReq, modifiedMaxResourceReq := &framework.Resource{}, &framework.Resource{}
	originalMaxResourceReq.SetMaxResource(resource.RegionRequests(originalOtherRegion, resource.RegionResourcesOptions{InPlaceRegionVerticalScalingEnabled: f.enableInPlaceRegionVerticalScaling}))
	modifiedMaxResourceReq.SetMaxResource(resource.RegionRequests(modifiedOtherRegion, resource.RegionResourcesOptions{InPlaceRegionVerticalScalingEnabled: f.enableInPlaceRegionVerticalScaling}))

	// check whether the resource request of the modified region is less than the original region.
	regionRequests := resource.RegionRequests(targetRegion, resource.RegionResourcesOptions{InPlaceRegionVerticalScalingEnabled: f.enableInPlaceRegionVerticalScaling})
	for rName, rValue := range regionRequests {
		if rValue.IsZero() {
			// We only care about the resources requested by the region we are trying to schedule.
			continue
		}
		switch rName {
		case corev1.ResourceCPU:
			if originalMaxResourceReq.MilliCPU > modifiedMaxResourceReq.MilliCPU {
				return true
			}
		case corev1.ResourceMemory:
			if originalMaxResourceReq.Memory > modifiedMaxResourceReq.Memory {
				return true
			}
		case corev1.ResourceEphemeralStorage:
			if originalMaxResourceReq.EphemeralStorage > modifiedMaxResourceReq.EphemeralStorage {
				return true
			}
		default:
			if schedutil.IsScalarResourceName(rName) && originalMaxResourceReq.ScalarResources[rName] > modifiedMaxResourceReq.ScalarResources[rName] {
				return true
			}
		}
	}
	return false
}

// isSchedulableAfterRunnerChange is invoked whenever a runner added or changed. It checks whether
// that change made a previously unschedulable region schedulable.
func (f *Fit) isSchedulableAfterRunnerChange(logger klog.Logger, region *corev1.Region, oldObj, newObj interface{}) (framework.QueueingHint, error) {
	_, modifiedRunner, err := schedutil.As[*corev1.Runner](oldObj, newObj)
	if err != nil {
		return framework.Queue, err
	}
	// TODO: also check if the original runner meets the region's resource requestments once preCheck is completely removed.
	// See: https://github.com/kubernetes/kubernetes/issues/110175
	if isFit(region, modifiedRunner) {
		logger.V(5).Info("runner was updated, and may fit with the region's resource requestments", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
		return framework.Queue, nil
	}

	logger.V(5).Info("runner was created or updated, but it doesn't have enough resource(s) to accommodate this region", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
	return framework.QueueSkip, nil
}

// isFit checks if the region fits the runner. If the runner is nil, it returns false.
// It constructs a fake RunnerInfo object for the runner and checks if the region fits the runner.
func isFit(region *corev1.Region, runner *corev1.Runner) bool {
	if runner == nil {
		return false
	}
	runnerInfo := framework.NewRunnerInfo()
	runnerInfo.SetRunner(runner)
	return len(Fits(region, runnerInfo)) == 0
}

// Filter invoked at the filter extension point.
// Checks if a runner has sufficient resources, such as cpu, memory, gpu, opaque int resources etc to run a region.
// It returns a list of insufficient resources, if empty, then the runner has all the resources requested by the region.
func (f *Fit) Filter(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	s, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	insufficientResources := fitsRequest(s, runnerInfo, f.ignoredResources, f.ignoredResourceGroups)

	if len(insufficientResources) != 0 {
		// We will keep all failure reasons.
		failureReasons := make([]string, 0, len(insufficientResources))
		for i := range insufficientResources {
			failureReasons = append(failureReasons, insufficientResources[i].Reason)
		}
		return framework.NewStatus(framework.Unschedulable, failureReasons...)
	}
	return nil
}

// InsufficientResource describes what kind of resource limit is hit and caused the region to not fit the runner.
type InsufficientResource struct {
	ResourceName corev1.ResourceName
	// We explicitly have a parameter for reason to avoid formatting a message on the fly
	// for common resources, which is expensive for cluster autoscaler simulations.
	Reason    string
	Requested int64
	Used      int64
	Capacity  int64
}

// Fits checks if runner have enough resources to host the region.
func Fits(region *corev1.Region, runnerInfo *framework.RunnerInfo) []InsufficientResource {
	return fitsRequest(computeRegionResourceRequest(region), runnerInfo, nil, nil)
}

func fitsRequest(regionRequest *preFilterState, runnerInfo *framework.RunnerInfo, ignoredExtendedResources, ignoredResourceGroups sets.Set[string]) []InsufficientResource {
	insufficientResources := make([]InsufficientResource, 0, 4)

	allowedRegionNumber := runnerInfo.Allocatable.AllowedRegionNumber
	if len(runnerInfo.Regions)+1 > allowedRegionNumber {
		insufficientResources = append(insufficientResources, InsufficientResource{
			ResourceName: corev1.ResourceRegions,
			Reason:       "Too many regions",
			Requested:    1,
			Used:         int64(len(runnerInfo.Regions)),
			Capacity:     int64(allowedRegionNumber),
		})
	}

	if regionRequest.MilliCPU == 0 &&
		regionRequest.Memory == 0 &&
		regionRequest.EphemeralStorage == 0 &&
		len(regionRequest.ScalarResources) == 0 {
		return insufficientResources
	}

	if regionRequest.MilliCPU > 0 && regionRequest.MilliCPU > (runnerInfo.Allocatable.MilliCPU-runnerInfo.Requested.MilliCPU) {
		insufficientResources = append(insufficientResources, InsufficientResource{
			ResourceName: corev1.ResourceCPU,
			Reason:       "Insufficient cpu",
			Requested:    regionRequest.MilliCPU,
			Used:         runnerInfo.Requested.MilliCPU,
			Capacity:     runnerInfo.Allocatable.MilliCPU,
		})
	}
	if regionRequest.Memory > 0 && regionRequest.Memory > (runnerInfo.Allocatable.Memory-runnerInfo.Requested.Memory) {
		insufficientResources = append(insufficientResources, InsufficientResource{
			ResourceName: corev1.ResourceMemory,
			Reason:       "Insufficient memory",
			Requested:    regionRequest.Memory,
			Used:         runnerInfo.Requested.Memory,
			Capacity:     runnerInfo.Allocatable.Memory,
		})
	}
	if regionRequest.EphemeralStorage > 0 &&
		regionRequest.EphemeralStorage > (runnerInfo.Allocatable.EphemeralStorage-runnerInfo.Requested.EphemeralStorage) {
		insufficientResources = append(insufficientResources, InsufficientResource{
			ResourceName: corev1.ResourceEphemeralStorage,
			Reason:       "Insufficient ephemeral-storage",
			Requested:    regionRequest.EphemeralStorage,
			Used:         runnerInfo.Requested.EphemeralStorage,
			Capacity:     runnerInfo.Allocatable.EphemeralStorage,
		})
	}

	for rName, rQuant := range regionRequest.ScalarResources {
		// Skip in case request quantity is zero
		if rQuant == 0 {
			continue
		}

		if v1helper.IsExtendedResourceName(rName) {
			// If this resource is one of the extended resources that should be ignored, we will skip checking it.
			// rName is guaranteed to have a slash due to API validation.
			var rNamePrefix string
			if ignoredResourceGroups.Len() > 0 {
				rNamePrefix = strings.Split(string(rName), "/")[0]
			}
			if ignoredExtendedResources.Has(string(rName)) || ignoredResourceGroups.Has(rNamePrefix) {
				continue
			}
		}

		if rQuant > (runnerInfo.Allocatable.ScalarResources[rName] - runnerInfo.Requested.ScalarResources[rName]) {
			insufficientResources = append(insufficientResources, InsufficientResource{
				ResourceName: rName,
				Reason:       fmt.Sprintf("Insufficient %v", rName),
				Requested:    regionRequest.ScalarResources[rName],
				Used:         runnerInfo.Requested.ScalarResources[rName],
				Capacity:     runnerInfo.Allocatable.ScalarResources[rName],
			})
		}
	}

	return insufficientResources
}

// Score invoked at the Score extension point.
func (f *Fit) Score(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	runnerInfo, err := f.handle.SnapshotSharedLister().RunnerInfos().Get(runnerName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting runner %q from Snapshot: %w", runnerName, err))
	}

	s, err := getPreScoreState(state)
	if err != nil {
		s = &preScoreState{
			regionRequests: f.calculateRegionResourceRequestList(region, f.resources),
		}
	}

	return f.score(ctx, region, runnerInfo, s.regionRequests)
}
