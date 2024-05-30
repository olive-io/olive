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

package interdefinitionaffinity

import (
	"context"
	"fmt"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

const (
	// preFilterStateKey is the key in CycleState to InterDefinitionAffinity pre-computed data for Filtering.
	// Using the name of the plugin will likely help us avoid collisions with other plugins.
	preFilterStateKey = "PreFilter" + Name

	// ErrReasonExistingAntiAffinityRulesNotMatch is used for ExistingDefinitionsAntiAffinityRulesNotMatch predicate error.
	ErrReasonExistingAntiAffinityRulesNotMatch = "runner(s) didn't satisfy existing definitions anti-affinity rules"
	// ErrReasonAffinityRulesNotMatch is used for DefinitionAffinityRulesNotMatch predicate error.
	ErrReasonAffinityRulesNotMatch = "runner(s) didn't match definition affinity rules"
	// ErrReasonAntiAffinityRulesNotMatch is used for DefinitionAntiAffinityRulesNotMatch predicate error.
	ErrReasonAntiAffinityRulesNotMatch = "runner(s) didn't match definition anti-affinity rules"
)

// preFilterState computed at PreFilter and used at Filter.
type preFilterState struct {
	// A map of topology pairs to the number of existing definitions that has anti-affinity terms that match the "definition".
	existingAntiAffinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing definitions that match the affinity terms of the "definition".
	affinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing definitions that match the anti-affinity terms of the "definition".
	antiAffinityCounts topologyToMatchedTermCount
	// definitionInfo of the incoming definition.
	definitionInfo *framework.DefinitionInfo
	// A copy of the incoming definition's namespace labels.
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
	// No need to deep copy the definitionInfo because it shouldn't change.
	copy.definitionInfo = s.definitionInfo
	copy.namespaceLabels = s.namespaceLabels
	return &copy
}

// updateWithDefinition updates the preFilterState counters with the (anti)affinity matches for the given definitionInfo.
func (s *preFilterState) updateWithDefinition(pInfo *framework.DefinitionInfo, runner *monv1.Runner, multiplier int64) {
	if s == nil {
		return
	}

	s.existingAntiAffinityCounts.updateWithAntiAffinityTerms(pInfo.RequiredAntiAffinityTerms, s.definitionInfo.Definition, s.namespaceLabels, runner, multiplier)
	s.affinityCounts.updateWithAffinityTerms(s.definitionInfo.RequiredAffinityTerms, pInfo.Definition, runner, multiplier)
	// The incoming definition's terms have the namespaceSelector merged into the namespaces, and so
	// here we don't lookup the updated definition's namespace labels, hence passing nil for nsLabels.
	s.antiAffinityCounts.updateWithAntiAffinityTerms(s.definitionInfo.RequiredAntiAffinityTerms, pInfo.Definition, nil, runner, multiplier)
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

func (m topologyToMatchedTermCount) update(runner *monv1.Runner, tk string, value int64) {
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
// for each affinity term if "targetDefinition" matches ALL terms.
func (m topologyToMatchedTermCount) updateWithAffinityTerms(
	terms []framework.AffinityTerm, definition *corev1.Definition, runner *monv1.Runner, value int64) {
	if definitionMatchesAllAffinityTerms(terms, definition) {
		for _, t := range terms {
			m.update(runner, t.TopologyKey, value)
		}
	}
}

// updates the topologyToMatchedTermCount map with the specified value
// for each anti-affinity term matched the target definition.
func (m topologyToMatchedTermCount) updateWithAntiAffinityTerms(terms []framework.AffinityTerm, definition *corev1.Definition, nsLabels labels.Set, runner *monv1.Runner, value int64) {
	// Check anti-affinity terms.
	for _, t := range terms {
		if t.Matches(definition, nsLabels) {
			m.update(runner, t.TopologyKey, value)
		}
	}
}

// returns true IFF the given definition matches all the given terms.
func definitionMatchesAllAffinityTerms(terms []framework.AffinityTerm, definition *corev1.Definition) bool {
	if len(terms) == 0 {
		return false
	}
	for _, t := range terms {
		// The incoming definition NamespaceSelector was merged into the Namespaces set, and so
		// we are not explicitly passing in namespace labels.
		if !t.Matches(definition, nil) {
			return false
		}
	}
	return true
}

// calculates the following for each existing definition on each runner:
//  1. Whether it has DefinitionAntiAffinity
//  2. Whether any AntiAffinityTerm matches the incoming definition
func (pl *InterDefinitionAffinity) getExistingAntiAffinityCounts(ctx context.Context, definition *corev1.Definition, nsLabels labels.Set, runners []*framework.RunnerInfo) topologyToMatchedTermCount {
	topoMaps := make([]topologyToMatchedTermCount, len(runners))
	index := int32(-1)
	processRunner := func(i int) {
		runnerInfo := runners[i]
		runner := runnerInfo.Runner()

		topoMap := make(topologyToMatchedTermCount)
		for _, existingDefinition := range runnerInfo.DefinitionsWithRequiredAntiAffinity {
			topoMap.updateWithAntiAffinityTerms(existingDefinition.RequiredAntiAffinityTerms, definition, nsLabels, runner, 1)
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

// finds existing Definitions that match affinity terms of the incoming definition's (anti)affinity terms.
// It returns a topologyToMatchedTermCount that are checked later by the affinity
// predicate. With this topologyToMatchedTermCount available, the affinity predicate does not
// need to check all the definitions in the cluster.
func (pl *InterDefinitionAffinity) getIncomingAffinityAntiAffinityCounts(ctx context.Context, definitionInfo *framework.DefinitionInfo, allRunners []*framework.RunnerInfo) (topologyToMatchedTermCount, topologyToMatchedTermCount) {
	affinityCounts := make(topologyToMatchedTermCount)
	antiAffinityCounts := make(topologyToMatchedTermCount)
	if len(definitionInfo.RequiredAffinityTerms) == 0 && len(definitionInfo.RequiredAntiAffinityTerms) == 0 {
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
		for _, existingDefinition := range runnerInfo.Definitions {
			affinity.updateWithAffinityTerms(definitionInfo.RequiredAffinityTerms, existingDefinition.Definition, runner, 1)
			// The incoming definition's terms have the namespaceSelector merged into the namespaces, and so
			// here we don't lookup the existing definition's namespace labels, hence passing nil for nsLabels.
			antiAffinity.updateWithAntiAffinityTerms(definitionInfo.RequiredAntiAffinityTerms, existingDefinition.Definition, nil, runner, 1)
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
func (pl *InterDefinitionAffinity) PreFilter(ctx context.Context, cycleState *framework.CycleState, definition *corev1.Definition) (*framework.PreFilterResult, *framework.Status) {
	var allRunners []*framework.RunnerInfo
	var runnersWithRequiredAntiAffinityDefinitions []*framework.RunnerInfo
	var err error
	if allRunners, err = pl.sharedLister.RunnerInfos().List(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("failed to list RunnerInfos: %w", err))
	}
	if runnersWithRequiredAntiAffinityDefinitions, err = pl.sharedLister.RunnerInfos().HaveDefinitionsWithRequiredAntiAffinityList(); err != nil {
		return nil, framework.AsStatus(fmt.Errorf("failed to list RunnerInfos with definitions with affinity: %w", err))
	}

	s := &preFilterState{}

	if s.definitionInfo, err = framework.NewDefinitionInfo(definition); err != nil {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("parsing definition: %+v", err))
	}

	for i := range s.definitionInfo.RequiredAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.definitionInfo.RequiredAffinityTerms[i]); err != nil {
			return nil, framework.AsStatus(err)
		}
	}
	for i := range s.definitionInfo.RequiredAntiAffinityTerms {
		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.definitionInfo.RequiredAntiAffinityTerms[i]); err != nil {
			return nil, framework.AsStatus(err)
		}
	}
	logger := klog.FromContext(ctx)
	s.namespaceLabels = GetNamespaceLabelsSnapshot(logger, definition.Namespace, pl.nsLister)

	s.existingAntiAffinityCounts = pl.getExistingAntiAffinityCounts(ctx, definition, s.namespaceLabels, runnersWithRequiredAntiAffinityDefinitions)
	s.affinityCounts, s.antiAffinityCounts = pl.getIncomingAffinityAntiAffinityCounts(ctx, s.definitionInfo, allRunners)

	if len(s.existingAntiAffinityCounts) == 0 && len(s.definitionInfo.RequiredAffinityTerms) == 0 && len(s.definitionInfo.RequiredAntiAffinityTerms) == 0 {
		return nil, framework.NewStatus(framework.Skip)
	}

	cycleState.Write(preFilterStateKey, s)
	return nil, nil
}

// PreFilterExtensions returns prefilter extensions, definition add and remove.
func (pl *InterDefinitionAffinity) PreFilterExtensions() framework.PreFilterExtensions {
	return pl
}

// AddDefinition from pre-computed data in cycleState.
func (pl *InterDefinitionAffinity) AddDefinition(ctx context.Context, cycleState *framework.CycleState, definitionToSchedule *corev1.Definition, definitionInfoToAdd *framework.DefinitionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	state.updateWithDefinition(definitionInfoToAdd, runnerInfo.Runner(), 1)
	return nil
}

// RemoveDefinition from pre-computed data in cycleState.
func (pl *InterDefinitionAffinity) RemoveDefinition(ctx context.Context, cycleState *framework.CycleState, definitionToSchedule *corev1.Definition, definitionInfoToRemove *framework.DefinitionInfo, runnerInfo *framework.RunnerInfo) *framework.Status {
	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}
	state.updateWithDefinition(definitionInfoToRemove, runnerInfo.Runner(), -1)
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
		return nil, fmt.Errorf("%+v  convert to interdefinitionaffinity.state error", c)
	}
	return s, nil
}

// Checks if scheduling the definition onto this runner would break any anti-affinity
// terms indicated by the existing definitions.
func satisfyExistingDefinitionsAntiAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	if len(state.existingAntiAffinityCounts) > 0 {
		// Iterate over topology pairs to get any of the definitions being affected by
		// the scheduled definition anti-affinity terms
		for topologyKey, topologyValue := range runnerInfo.Runner().Labels {
			tp := topologyPair{key: topologyKey, value: topologyValue}
			if state.existingAntiAffinityCounts[tp] > 0 {
				return false
			}
		}
	}
	return true
}

// Checks if the runner satisfies the incoming definition's anti-affinity rules.
func satisfyDefinitionAntiAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	if len(state.antiAffinityCounts) > 0 {
		for _, term := range state.definitionInfo.RequiredAntiAffinityTerms {
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

// Checks if the runner satisfies the incoming definition's affinity rules.
func satisfyDefinitionAffinity(state *preFilterState, runnerInfo *framework.RunnerInfo) bool {
	definitionsExist := true
	for _, term := range state.definitionInfo.RequiredAffinityTerms {
		if topologyValue, ok := runnerInfo.Runner().Labels[term.TopologyKey]; ok {
			tp := topologyPair{key: term.TopologyKey, value: topologyValue}
			if state.affinityCounts[tp] <= 0 {
				definitionsExist = false
			}
		} else {
			// All topology labels must exist on the runner.
			return false
		}
	}

	if !definitionsExist {
		// This definition may be the first definition in a series that have affinity to themselves. In order
		// to not leave such definitions in pending state forever, we check that if no other definition
		// in the cluster matches the namespace and selector of this definition, the definition matches
		// its own terms, and the runner has all the requested topologies, then we allow the definition
		// to pass the affinity check.
		if len(state.affinityCounts) == 0 && definitionMatchesAllAffinityTerms(state.definitionInfo.RequiredAffinityTerms, state.definitionInfo.Definition) {
			return true
		}
		return false
	}
	return true
}

// Filter invoked at the filter extension point.
// It checks if a definition can be scheduled on the specified runner with definition affinity/anti-affinity configuration.
func (pl *InterDefinitionAffinity) Filter(ctx context.Context, cycleState *framework.CycleState, definition *corev1.Definition, runnerInfo *framework.RunnerInfo) *framework.Status {

	state, err := getPreFilterState(cycleState)
	if err != nil {
		return framework.AsStatus(err)
	}

	if !satisfyDefinitionAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonAffinityRulesNotMatch)
	}

	if !satisfyDefinitionAntiAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.Unschedulable, ErrReasonAntiAffinityRulesNotMatch)
	}

	if !satisfyExistingDefinitionsAntiAffinity(state, runnerInfo) {
		return framework.NewStatus(framework.Unschedulable, ErrReasonExistingAntiAffinityRulesNotMatch)
	}

	return nil
}
