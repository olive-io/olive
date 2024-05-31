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
	"sync/atomic"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

const (
	// preFilterStateKey is the key in CycleState to InterRegionAffinity pre-computed data for Filtering.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + Name

	// ErrReasonExistingAntiAffinityRulesNotMatch is used for ExistingRegionsAntiAffinityRulesNotMatch predicate error.
	ErrReasonExistingAntiAffinityRulesNotMatch = "runner(s) didn't satisfy existing regions anti-affinity rules"
	// ErrReasonAffinityRulesNotMatch is used for RegionAffinityRulesNotMatch predicate error.
	ErrReasonAffinityRulesNotMatch = "runner(s) didn't match region affinity rules"
	// ErrReasonAntiAffinityRulesNotMatch is used for RegionAntiAffinityRulesNotMatch predicate error.
	ErrReasonAntiAffinityRulesNotMatch = "runner(s) didn't match region anti-affinity rules"
)

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	// A map of topology pairs to the number of existing regions that has anti-affinity terms that match the "region".
	existingAntiAffinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing regions that match the affinity terms of the "region".
	affinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing regions that match the anti-affinity terms of the "region".
	antiAffinityCounts topologyToMatchedTermCount
	// regionInfo of the incoming region.
	regionInfo *framework.RegionInfo
	// A copy of the incoming region's namespace labels.
	namespaceLabels labels.Set
}

// Clone the prefilter state.
func (s *preFilterState) Clone() framework.StateData {
	if s == nil {
		return nil
	}

	copy := preFilterState{}
	copy.affinityCounts = s.affinityCounts.clone()
	copy.antiAffinityCounts = s.antiAffinityCounts.clone()
	copy.existingAntiAffinityCounts = s.existingAntiAffinityCounts.clone()
	// No need to deep copy the regionInfo because it shouldn't change.
	copy.regionInfo = s.regionInfo
	copy.namespaceLabels = s.namespaceLabels
	return &copy
}

// updateWithRegion updates the preFilterState counters with the (anti)affinity matches for the given regionInfo.
func (s *preFilterState) updateWithRegion(pInfo *framework.RegionInfo, runner *corev1.Runner, multiplier int64) {
	if s == nil {
		return
	}

	s.existingAntiAffinityCounts.updateWithAntiAffinityTerms(pInfo.RequiredAntiAffinityTerms, s.regionInfo.Region, s.namespaceLabels, runner, multiplier)
	s.affinityCounts.updateWithAffinityTerms(s.regionInfo.RequiredAffinityTerms, pInfo.Region, runner, multiplier)
	// The incoming region's terms have the namespaceSelector merged into the namespaces, and so
	// here we don't lookup the updated region's namespace labels, hence passing nil for nsLabels.
	s.antiAffinityCounts.updateWithAntiAffinityTerms(s.regionInfo.RequiredAntiAffinityTerms, pInfo.Region, nil, runner, multiplier)
}

type topologyPair struct {
	key   string
	value string
}
type topologyToMatchedTermCount map[topologyPair]int64

func (m topologyToMatchedTermCount) append(toAppend topologyToMatchedTermCount) {
	for pair := range toAppend {
		m[pair] += toAppend[pair]
	}
}

func (m topologyToMatchedTermCount) clone() topologyToMatchedTermCount {
	copy := make(topologyToMatchedTermCount, len(m))
	copy.append(m)
	return copy
}

func (m topologyToMatchedTermCount) update(runner *corev1.Runner, tk string, value int64) {
	if tv, ok := runner.Labels[tk]; ok {
		pair := topologyPair{key: tk, value: tv}
		m[pair] += value
		// value could be negative, hence we delete the entry if it is down to zero.
		if m[pair] == 0 {
			delete(m, pair)
		}
	}
}

// updates the topologyToMatchedTermCount map with the specified value
// for each affinity term if "targetRegion" matches ALL terms.
func (m topologyToMatchedTermCount) updateWithAffinityTerms(
	terms []framework.AffinityTerm, region *corev1.Region, runner *corev1.Runner, value int64) {
	if regionMatchesAllAffinityTerms(terms, region) {
		for _, t := range terms {
			m.update(runner, t.TopologyKey, value)
		}
	}
}

// updates the topologyToMatchedTermCount map with the specified value
// for each anti-affinity term matched the target region.
func (m topologyToMatchedTermCount) updateWithAntiAffinityTerms(terms []framework.AffinityTerm, region *corev1.Region, nsLabels labels.Set, runner *corev1.Runner, value int64) {
	// Check anti-affinity terms.
	for _, t := range terms {
		if t.Matches(region, nsLabels) {
			m.update(runner, t.TopologyKey, value)
		}
	}
}

// returns true IFF the given region matches all the given terms.
func regionMatchesAllAffinityTerms(terms []framework.AffinityTerm, region *corev1.Region) bool {
	if len(terms) == 0 {
		return false
	}
	for _, t := range terms {
		// The incoming region NamespaceSelector was merged into the Namespaces set, and so
		// we are not explicitly passing in namespace labels.
		if !t.Matches(region, nil) {
			return false
		}
	}
	return true
}

// calculates the following for each existing region on each runner:
//  1. Whether it has RegionAntiAffinity
//  2. Whether any AntiAffinityTerm matches the incoming region
func (pl *InterRegionAffinity) getExistingAntiAffinityCounts(ctx context.Context, region *corev1.Region, nsLabels labels.Set, runners []*framework.RunnerInfo) topologyToMatchedTermCount {
	topoMaps := make([]topologyToMatchedTermCount, len(runners))
	index := int32(-1)
	processRunner := func(i int) {
		runnerInfo := runners[i]
		runner := runnerInfo.Runner()

		topoMap := make(topologyToMatchedTermCount)
		for _, existingRegion := range runnerInfo.RegionsWithRequiredAntiAffinity {
			topoMap.updateWithAntiAffinityTerms(existingRegion.RequiredAntiAffinityTerms, region, nsLabels, runner, 1)
		}
		if len(topoMap) != 0 {
			topoMaps[atomic.AddInt32(&index, 1)] = topoMap
		}
	}
	pl.parallelizer.Until(ctx, len(runners), processRunner, pl.Name())

	result := make(topologyToMatchedTermCount)
	for i := 0; i <= int(index); i++ {
		result.append(topoMaps[i])
	}

	return result
}

// finds existing Regions that match affinity terms of the incoming region's (anti)affinity terms.
// It returns a topologyToMatchedTermCount that are checked later by the affinity
// predicate. With this topologyToMatchedTermCount available, the affinity predicate does not
// need to check all the regions in the cluster.
func (pl *InterRegionAffinity) getIncomingAffinityAntiAffinityCounts(ctx context.Context, regionInfo *framework.RegionInfo, allRunners []*framework.RunnerInfo) (topologyToMatchedTermCount, topologyToMatchedTermCount) {
	affinityCounts := make(topologyToMatchedTermCount)
	antiAffinityCounts := make(topologyToMatchedTermCount)
	if len(regionInfo.RequiredAffinityTerms) == 0 && len(regionInfo.RequiredAntiAffinityTerms) == 0 {
		return affinityCounts, antiAffinityCounts
	}

	affinityCountsList := make([]topologyToMatchedTermCount, len(allRunners))
	antiAffinityCountsList := make([]topologyToMatchedTermCount, len(allRunners))
	index := int32(-1)
	processRunner := func(i int) {
		runnerInfo := allRunners[i]
		runner := runnerInfo.Runner()

		affinity := make(topologyToMatchedTermCount)
		antiAffinity := make(topologyToMatchedTermCount)
		for _, existingRegion := range runnerInfo.Regions {
			affinity.updateWithAffinityTerms(regionInfo.RequiredAffinityTerms, existingRegion.Region, runner, 1)
			// The incoming region's terms have the namespaceSelector merged into the namespaces, and so
			// here we don't lookup the existing region's namespace labels, hence passing nil for nsLabels.
			antiAffinity.updateWithAntiAffinityTerms(regionInfo.RequiredAntiAffinityTerms, existingRegion.Region, nil, runner, 1)
		}

		if len(affinity) > 0 || len(antiAffinity) > 0 {
			k := atomic.AddInt32(&index, 1)
			affinityCountsList[k] = affinity
			antiAffinityCountsList[k] = antiAffinity
		}
	}
	pl.parallelizer.Until(ctx, len(allRunners), processRunner, pl.Name())

	for i := 0; i <= int(index); i++ {
		affinityCounts.append(affinityCountsList[i])
		antiAffinityCounts.append(antiAffinityCountsList[i])
	}

	return affinityCounts, antiAffinityCounts
}

// PreFilter invoked at the prefilter extension point.
func (pl *InterRegionAffinity) PreFilter(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	var allRunners []*framework.RunnerInfo
	var runnersWithRequiredAntiAffinityRegions []*framework.RunnerInfo
	var err error
	if allRunners, err = pl.sharedLister.RunnerInfos().List(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("failed to list RunnerInfos: %w", err))
	}
	if runnersWithRequiredAntiAffinityRegions, err = pl.sharedLister.RunnerInfos().HaveRegionsWithRequiredAntiAffinityList(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("failed to list RunnerInfos with regions with affinity: %w", err))
	}

	s := &preFilterState{}

	if s.regionInfo, err = framework.NewRegionInfo(region); err != nil {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("parsing region: %+v", err))
	}

	for i := range s.regionInfo.RequiredAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.regionInfo.RequiredAffinityTerms[i]); err != nil {
			return nil, framework.AsStatus(err)
		}
	}
	for i := range s.regionInfo.RequiredAntiAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.regionInfo.RequiredAntiAffinityTerms[i]); err != nil {
			return nil, framework.AsStatus(err)
		}
	}
	logger := klog.FromContext(ctx)
	s.namespaceLabels = GetNamespaceLabelsSnapshot(logger, region.Namespace, pl.nsLister)

	s.existingAntiAffinityCounts = pl.getExistingAntiAffinityCounts(ctx, region, s.namespaceLabels, runnersWithRequiredAntiAffinityRegions)
	s.affinityCounts, s.antiAffinityCounts = pl.getIncomingAffinityAntiAffinityCounts(ctx, s.regionInfo, allRunners)

	if len(s.existingAntiAffinityCounts) == 0 && len(s.regionInfo.RequiredAffinityTerms) == 0 && len(s.regionInfo.RequiredAntiAffinityTerms) == 0 {
		return nil, framework.NewStatus(framework.Skip)
	}

	cycleState.Write(preFilterStateKey, s)
	return nil, nil
}

// PreFilterExtensions returns prefilter extensions, region add and remove.
func (pl *InterRegionAffinity) PreFilterExtensions() framework.PreFilterExtensions {
	return pl
}

// AddRegion from pre-computed data in cycleState.
func (pl *InterRegionAffinity) AddRegion(ctx context.Context, cycleState *framework.CycleState, regionToSchedule *corev1.Region, regionInfoToAdd *framework.RegionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	state.updateWithRegion(regionInfoToAdd, runnerInfo.Runner(), 1)
	return nil
}

// RemoveRegion from pre-computed data in cycleState.
func (pl *InterRegionAffinity) RemoveRegion(ctx context.Context, cycleState *framework.CycleState, regionToSchedule *corev1.Region, regionInfoToRemove *framework.RegionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	state.updateWithRegion(regionInfoToRemove, runnerInfo.Runner(), -1)
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
		return nil, fmt.Errorf("%+v  convert to interregionaffinity.state error", c)
	}
	return s, nil
}

// Checks if scheduling the region onto this runner would break any anti-affinity
// terms indicated by the existing regions.
func satisfyExistingRegionsAntiAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	if len(state.existingAntiAffinityCounts) > 0 {
		// Iterate over topology pairs to get any of the regions being affected by
		// the scheduled region anti-affinity terms
		for topologyKey, topologyValue := range runnerInfo.Runner().Labels {
			tp := topologyPair{key: topologyKey, value: topologyValue}
			if state.existingAntiAffinityCounts[tp] > 0 {
				return false
			}
		}
	}
	return true
}

// Checks if the runner satisfies the incoming region's anti-affinity rules.
func satisfyRegionAntiAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	if len(state.antiAffinityCounts) > 0 {
		for _, term := range state.regionInfo.RequiredAntiAffinityTerms {
			if topologyValue, ok := runnerInfo.Runner().Labels[term.TopologyKey]; ok {
				tp := topologyPair{key: term.TopologyKey, value: topologyValue}
				if state.antiAffinityCounts[tp] > 0 {
					return false
				}
			}
		}
	}
	return true
}

// Checks if the runner satisfies the incoming region's affinity rules.
func satisfyRegionAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	regionsExist := true
	for _, term := range state.regionInfo.RequiredAffinityTerms {
		if topologyValue, ok := runnerInfo.Runner().Labels[term.TopologyKey]; ok {
			tp := topologyPair{key: term.TopologyKey, value: topologyValue}
			if state.affinityCounts[tp] <= 0 {
				regionsExist = false
			}
		} else {
			// All topology labels must exist on the runner.
			return false
		}
	}

	if !regionsExist {
		// This region may be the first region in a series that have affinity to themselves. In order
		// to not leave such regions in pending state forever, we check that if no other region
		// in the cluster matches the namespace and selector of this region, the region matches
		// its own terms, and the runner has all the requested topologies, then we allow the region
		// to pass the affinity check.
		if len(state.affinityCounts) == 0 && regionMatchesAllAffinityTerms(state.regionInfo.RequiredAffinityTerms, state.regionInfo.Region) {
			return true
		}
		return false
	}
	return true
}

// Filter invoked at the filter extension point.
// It checks if a region can be scheduled on the specified runner with region affinity/anti-affinity configuration.
func (pl *InterRegionAffinity) Filter(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {

	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	if !satisfyRegionAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonAffinityRulesNotMatch)
	}

	if !satisfyRegionAntiAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.Unschedulable, ErrReasonAntiAffinityRulesNotMatch)
	}

	if !satisfyExistingRegionsAntiAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.Unschedulable, ErrReasonExistingAntiAffinityRulesNotMatch)
	}

	return nil
}
