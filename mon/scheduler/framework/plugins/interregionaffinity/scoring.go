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
	"math"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// preScoreStateKey is the key in CycleState to InterRegionAffinity pre-computed data for Scoring.
const preScoreStateKey = "PreScore" + Name

type scoreMap map[string]map[string]int64

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	topologyScore scoreMap
	regionInfo    *framework.RegionInfo
	// A copy of the incoming region's namespace labels.
	namespaceLabels labels.Set
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

func (m scoreMap) processTerm(term *framework.AffinityTerm, weight int32, region *corev1.Region, nsLabels labels.Set, runner *corev1.Runner, multiplier int32) {
	if term.Matches(region, nsLabels) {
		if tpValue, tpValueExist := runner.Labels[term.TopologyKey]; tpValueExist {
			if m[term.TopologyKey] == nil {
				m[term.TopologyKey] = make(map[string]int64)
			}
			m[term.TopologyKey][tpValue] += int64(weight * multiplier)
		}
	}
}

func (m scoreMap) processTerms(terms []framework.WeightedAffinityTerm, region *corev1.Region, nsLabels labels.Set, runner *corev1.Runner, multiplier int32) {
	for _, term := range terms {
		m.processTerm(&term.AffinityTerm, term.Weight, region, nsLabels, runner, multiplier)
	}
}

func (m scoreMap) append(other scoreMap) {
	for topology, oScores := range other {
		scores := m[topology]
		if scores == nil {
			m[topology] = oScores
			continue
		}
		for k, v := range oScores {
			scores[k] += v
		}
	}
}

func (pl *InterRegionAffinity) processExistingRegion(
	state *preScoreState,
	existingRegion *framework.RegionInfo,
	existingRegionRunnerInfo *framework.RunnerInfo,
	incomingRegion *corev1.Region,
	topoScore scoreMap,
) {
	existingRegionRunner := existingRegionRunnerInfo.Runner()
	if len(existingRegionRunner.Labels) == 0 {
		return
	}

	// For every soft region affinity term of <region>, if <existingRegion> matches the term,
	// increment <p.counts> for every runner in the cluster with the same <term.TopologyKey>
	// value as that of <existingRegions>`s runner by the term`s weight.
	// Note that the incoming region's terms have the namespaceSelector merged into the namespaces, and so
	// here we don't lookup the existing region's namespace labels, hence passing nil for nsLabels.
	topoScore.processTerms(state.regionInfo.PreferredAffinityTerms, existingRegion.Region, nil, existingRegionRunner, 1)

	// For every soft region anti-affinity term of <region>, if <existingRegion> matches the term,
	// decrement <p.counts> for every runner in the cluster with the same <term.TopologyKey>
	// value as that of <existingRegion>`s runner by the term`s weight.
	// Note that the incoming region's terms have the namespaceSelector merged into the namespaces, and so
	// here we don't lookup the existing region's namespace labels, hence passing nil for nsLabels.
	topoScore.processTerms(state.regionInfo.PreferredAntiAffinityTerms, existingRegion.Region, nil, existingRegionRunner, -1)

	// For every hard region affinity term of <existingRegion>, if <region> matches the term,
	// increment <p.counts> for every runner in the cluster with the same <term.TopologyKey>
	// value as that of <existingRegion>'s runner by the constant <args.hardRegionAffinityWeight>
	//if pl.args.HardRegionAffinityWeight > 0 && len(existingRegionRunner.Labels) != 0 {
	//	for _, t := range existingRegion.RequiredAffinityTerms {
	//		topoScore.processTerm(&t, pl.args.HardRegionAffinityWeight, incomingRegion, state.namespaceLabels, existingRegionRunner, 1)
	//	}
	//}

	// For every soft region affinity term of <existingRegion>, if <region> matches the term,
	// increment <p.counts> for every runner in the cluster with the same <term.TopologyKey>
	// value as that of <existingRegion>'s runner by the term's weight.
	topoScore.processTerms(existingRegion.PreferredAffinityTerms, incomingRegion, state.namespaceLabels, existingRegionRunner, 1)

	// For every soft region anti-affinity term of <existingRegion>, if <region> matches the term,
	// decrement <pm.counts> for every runner in the cluster with the same <term.TopologyKey>
	// value as that of <existingRegion>'s runner by the term's weight.
	topoScore.processTerms(existingRegion.PreferredAntiAffinityTerms, incomingRegion, state.namespaceLabels, existingRegionRunner, -1)
}

// PreScore builds and writes cycle state used by Score and NormalizeScore.
func (pl *InterRegionAffinity) PreScore(
	pCtx context.Context,
	cycleState *framework.CycleState,
	region *corev1.Region,
	runners []*framework.RunnerInfo,
) *framework.Status {
	if len(runners) == 0 {
		// No runners to score.
		return framework.NewStatus(framework.Skip)
	}

	if pl.sharedLister == nil {
		return framework.NewStatus(framework.Error, "empty shared lister in InterRegionAffinity PreScore")
	}

	//affinity := region.Spec.Affinity
	//hasPreferredAffinityConstraints := affinity != nil && affinity.RegionAffinity != nil && len(affinity.RegionAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0
	//hasPreferredAntiAffinityConstraints := affinity != nil && affinity.RegionAntiAffinity != nil && len(affinity.RegionAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0
	//hasConstraints := hasPreferredAffinityConstraints || hasPreferredAntiAffinityConstraints
	hasConstraints := true
	//
	//// Optionally ignore calculating preferences of existing regions' affinity rules
	//// if the incoming region has no inter-region affinities.
	//if pl.args.IgnorePreferredTermsOfExistingRegions && !hasConstraints {
	//	return framework.NewStatus(framework.Skip)
	//}

	// Unless the region being scheduled has preferred affinity terms, we only
	// need to process runners hosting regions with affinity.
	var allRunners []*framework.RunnerInfo
	var err error
	if hasConstraints {
		allRunners, err = pl.sharedLister.RunnerInfos().List()
		if err != nil {
			return framework.AsStatus(fmt.Errorf("failed to get all runners from shared lister: %w", err))
		}
	} else {
		allRunners, err = pl.sharedLister.RunnerInfos().HaveRegionsWithAffinityList()
		if err != nil {
			return framework.AsStatus(fmt.Errorf("failed to get regions with affinity list: %w", err))
		}
	}

	state := &preScoreState{
		topologyScore: make(map[string]map[string]int64),
	}

	if state.regionInfo, err = framework.NewRegionInfo(region); err != nil {
		// Ideally we never reach here, because errors will be caught by PreFilter
		return framework.AsStatus(fmt.Errorf("failed to parse region: %w", err))
	}

	for i := range state.regionInfo.PreferredAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&state.regionInfo.PreferredAffinityTerms[i].AffinityTerm); err != nil {
			return framework.AsStatus(fmt.Errorf("updating PreferredAffinityTerms: %w", err))
		}
	}
	for i := range state.regionInfo.PreferredAntiAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&state.regionInfo.PreferredAntiAffinityTerms[i].AffinityTerm); err != nil {
			return framework.AsStatus(fmt.Errorf("updating PreferredAntiAffinityTerms: %w", err))
		}
	}
	logger := klog.FromContext(pCtx)
	state.namespaceLabels = GetNamespaceLabelsSnapshot(logger, region.Namespace, pl.nsLister)

	topoScores := make([]scoreMap, len(allRunners))
	index := int32(-1)
	processRunner := func(i int) {
		runnerInfo := allRunners[i]

		// Unless the region being scheduled has preferred affinity terms, we only
		// need to process regions with affinity in the runner.
		regionsToProcess := runnerInfo.RegionsWithAffinity
		if hasConstraints {
			// We need to process all the regions.
			regionsToProcess = runnerInfo.Regions
		}

		topoScore := make(scoreMap)
		for _, existingRegion := range regionsToProcess {
			pl.processExistingRegion(state, existingRegion, runnerInfo, region, topoScore)
		}
		if len(topoScore) > 0 {
			topoScores[atomic.AddInt32(&index, 1)] = topoScore
		}
	}
	pl.parallelizer.Until(pCtx, len(allRunners), processRunner, pl.Name())

	if index == -1 {
		return framework.NewStatus(framework.Skip)
	}

	for i := 0; i <= int(index); i++ {
		state.topologyScore.append(topoScores[i])
	}

	cycleState.Write(preScoreStateKey, state)
	return nil
}

func getPreScoreState(cycleState *framework.CycleState) (*preScoreState, error) {
	c, err := cycleState.Read(preScoreStateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q from cycleState: %w", preScoreStateKey, err)
	}

	s, ok := c.(*preScoreState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to interregionaffinity.preScoreState error", c)
	}
	return s, nil
}

// Score invoked at the Score extension point.
// The "score" returned in this function is the sum of weights got from cycleState which have its topologyKey matching with the runner's labels.
// it is normalized later.
// Note: the returned "score" is positive for region-affinity, and negative for region-antiaffinity.
func (pl *InterRegionAffinity) Score(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	runnerInfo, err := pl.sharedLister.RunnerInfos().Get(runnerName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("failed to get runner %q from Snapshot: %w", runnerName, err))
	}
	runner := runnerInfo.Runner()

	s, err := getPreScoreState(cycleState)
	if err != nil {
		return 0, framework.AsStatus(err)
	}
	var score int64
	for tpKey, tpValues := range s.topologyScore {
		if v, exist := runner.Labels[tpKey]; exist {
			score += tpValues[v]
		}
	}

	return score, nil
}

// NormalizeScore normalizes the score for each filteredRunner.
func (pl *InterRegionAffinity) NormalizeScore(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, scores framework.RunnerScoreList) *framework.Status {
	s, err := getPreScoreState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	if len(s.topologyScore) == 0 {
		return nil
	}

	var minCount int64 = math.MaxInt64
	var maxCount int64 = math.MinInt64
	for i := range scores {
		score := scores[i].Score
		if score > maxCount {
			maxCount = score
		}
		if score < minCount {
			minCount = score
		}
	}

	maxMinDiff := maxCount - minCount
	for i := range scores {
		fScore := float64(0)
		if maxMinDiff > 0 {
			fScore = float64(framework.MaxRunnerScore) * (float64(scores[i].Score-minCount) / float64(maxMinDiff))
		}

		scores[i].Score = int64(fScore)
	}

	return nil
}

// ScoreExtensions of the Score plugin.
func (pl *InterRegionAffinity) ScoreExtensions() framework.ScoreExtensions {
	return pl
}
