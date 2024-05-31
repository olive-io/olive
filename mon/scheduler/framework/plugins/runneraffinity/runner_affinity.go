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

package runneraffinity

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	configv1 "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/helper"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
	"github.com/olive-io/olive/mon/scheduler/util"
	"github.com/olive-io/olive/pkg/scheduling/corev1/runneraffinity"
)

// RunnerAffinity is a plugin that checks if a region runner selector matches the runner label.
type RunnerAffinity struct {
	handle              framework.Handle
	addedRunnerSelector *runneraffinity.RunnerSelector
	addedPrefSchedTerms *runneraffinity.PreferredSchedulingTerms
}

var _ framework.PreFilterPlugin = &RunnerAffinity{}
var _ framework.FilterPlugin = &RunnerAffinity{}
var _ framework.PreScorePlugin = &RunnerAffinity{}
var _ framework.ScorePlugin = &RunnerAffinity{}
var _ framework.EnqueueExtensions = &RunnerAffinity{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = names.RunnerAffinity

	// preScoreStateKey is the key in CycleState to RunnerAffinity pre-computed data for Scoring.
	preScoreStateKey = "PreScore" + Name

	// preFilterStateKey is the key in CycleState to RunnerAffinity pre-compute data for Filtering.
	preFilterStateKey = "PreFilter" + Name

	// ErrReasonRegion is the reason for Region's runner affinity/selector not matching.
	ErrReasonRegion = "runner(s) didn't match Region's runner affinity/selector"

	// errReasonEnforced is the reason for added runner affinity not matching.
	errReasonEnforced = "runner(s) didn't match scheduler-enforced runner affinity"

	// errReasonConflict is the reason for region's conflicting affinity rules.
	errReasonConflict = "region affinity terms conflict"
)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *RunnerAffinity) Name() string {
	return Name
}

type preFilterState struct {
	requiredRunnerSelectorAndAffinity runneraffinity.RequiredRunnerAffinity
}

// Clone just returns the same state because it is not affected by region additions or deletions.
func (s *preFilterState) Clone() framework.StateData {
	return s
}

// EventsToRegister returns the possible events that may make a Region
// failed by this plugin schedulable.
func (pl *RunnerAffinity) EventsToRegister() []framework.ClusterEventWithHint {
	return []framework.ClusterEventWithHint{
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.Update}, QueueingHintFn: pl.isSchedulableAfterRunnerChange},
	}
}

// isSchedulableAfterRunnerChange is invoked whenever a runner changed. It checks whether
// that change made a previously unschedulable region schedulable.
func (pl *RunnerAffinity) isSchedulableAfterRunnerChange(logger klog.Logger, region *corev1.Region, oldObj, newObj interface{}) (framework.QueueingHint, error) {
	_, modifiedRunner, err := util.As[*corev1.Runner](oldObj, newObj)
	if err != nil {
		return framework.Queue, err
	}

	if pl.addedRunnerSelector != nil && !pl.addedRunnerSelector.Match(modifiedRunner) {
		logger.V(4).Info("added or modified runner didn't match scheduler-enforced runner affinity and this event won't make the Region schedulable", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
		return framework.QueueSkip, nil
	}

	requiredRunnerAffinity := runneraffinity.GetRequiredRunnerAffinity(region)
	isMatched, err := requiredRunnerAffinity.Match(modifiedRunner)
	if err != nil {
		return framework.Queue, err
	}
	if isMatched {
		logger.V(4).Info("runner was created or updated, and matches with the region's RunnerAffinity", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
		return framework.Queue, nil
	}

	// TODO: also check if the original runner meets the region's requestments once preCheck is completely removed.
	// See: https://github.com/kubernetes/kubernetes/issues/110175

	logger.V(4).Info("runner was created or updated, but it doesn't make this region schedulable", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
	return framework.QueueSkip, nil
}

// PreFilter builds and writes cycle state used by Filter.
func (pl *RunnerAffinity) PreFilter(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region) (*framework.PreFilterResult, *framework.Status) {
	affinity := region.Spec.Affinity
	noRunnerAffinity := (affinity == nil ||
		affinity.RunnerAffinity == nil ||
		affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil)
	if noRunnerAffinity && pl.addedRunnerSelector == nil && region.Spec.RunnerSelector == nil {
		// RunnerAffinity Filter has nothing to do with the Region.
		return nil, framework.NewStatus(framework.Skip)
	}

	state := &preFilterState{requiredRunnerSelectorAndAffinity: runneraffinity.GetRequiredRunnerAffinity(region)}
	cycleState.Write(preFilterStateKey, state)

	if noRunnerAffinity || len(affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution.RunnerSelectorTerms) == 0 {
		return nil, nil
	}

	// Check if there is affinity to a specific runner and return it.
	terms := affinity.RunnerAffinity.RequiredDuringSchedulingIgnoredDuringExecution.RunnerSelectorTerms
	var runnerNames sets.Set[string]
	for _, t := range terms {
		var termRunnerNames sets.Set[string]
		for _, r := range t.MatchFields {
			if r.Key == metav1.ObjectNameField && r.Operator == corev1.RunnerSelectorOpIn {
				// The requirements represent ANDed constraints, and so we need to
				// find the intersection of runners.
				s := sets.New(r.Values...)
				if termRunnerNames == nil {
					termRunnerNames = s
				} else {
					termRunnerNames = termRunnerNames.Intersection(s)
				}
			}
		}
		if termRunnerNames == nil {
			// If this term has no runner.Name field affinity,
			// then all runners are eligible because the terms are ORed.
			return nil, nil
		}
		runnerNames = runnerNames.Union(termRunnerNames)
	}
	// If runnerNames is not nil, but length is 0, it means each term have conflicting affinity to runner.Name;
	// therefore, region will not match any runner.
	if runnerNames != nil && len(runnerNames) == 0 {
		return nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, errReasonConflict)
	} else if len(runnerNames) > 0 {
		return &framework.PreFilterResult{RunnerNames: runnerNames}, nil
	}
	return nil, nil

}

// PreFilterExtensions not necessary for this plugin as state doesn't depend on region additions or deletions.
func (pl *RunnerAffinity) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// Filter checks if the Runner matches the Region .spec.affinity.runnerAffinity and
// the plugin's added affinity.
func (pl *RunnerAffinity) Filter(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	runner := runnerInfo.Runner()

	if pl.addedRunnerSelector != nil && !pl.addedRunnerSelector.Match(runner) {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, errReasonEnforced)
	}

	s, err := getPreFilterState(state)
	if err != nil {
		// Fallback to calculate requiredRunnerSelector and requiredRunnerAffinity
		// here when PreFilter is disabled.
		s = &preFilterState{requiredRunnerSelectorAndAffinity: runneraffinity.GetRequiredRunnerAffinity(region)}
	}

	// Ignore parsing errors for backwards compatibility.
	match, _ := s.requiredRunnerSelectorAndAffinity.Match(runner)
	if !match {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion)
	}

	return nil
}

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	preferredRunnerAffinity *runneraffinity.PreferredSchedulingTerms
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// PreScore builds and writes cycle state used by Score and NormalizeScore.
func (pl *RunnerAffinity) PreScore(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	if len(runners) == 0 {
		return nil
	}
	preferredRunnerAffinity, err := getRegionPreferredRunnerAffinity(region)
	if err != nil {
		return framework.AsStatus(err)
	}
	if preferredRunnerAffinity == nil && pl.addedPrefSchedTerms == nil {
		// RunnerAffinity Score has nothing to do with the Region.
		return framework.NewStatus(framework.Skip)
	}
	state := &preScoreState{
		preferredRunnerAffinity: preferredRunnerAffinity,
	}
	cycleState.Write(preScoreStateKey, state)
	return nil
}

// Score returns the sum of the weights of the terms that match the Runner.
// Terms came from the Region .spec.affinity.runnerAffinity and from the plugin's
// default affinity.
func (pl *RunnerAffinity) Score(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	runnerInfo, err := pl.handle.SnapshotSharedLister().RunnerInfos().Get(runnerName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting runner %q from Snapshot: %w", runnerName, err))
	}

	runner := runnerInfo.Runner()

	var count int64
	if pl.addedPrefSchedTerms != nil {
		count += pl.addedPrefSchedTerms.Score(runner)
	}

	s, err := getPreScoreState(state)
	if err != nil {
		// Fallback to calculate preferredRunnerAffinity here when PreScore is disabled.
		preferredRunnerAffinity, err := getRegionPreferredRunnerAffinity(region)
		if err != nil {
			return 0, framework.AsStatus(err)
		}
		s = &preScoreState{
			preferredRunnerAffinity: preferredRunnerAffinity,
		}
	}

	if s.preferredRunnerAffinity != nil {
		count += s.preferredRunnerAffinity.Score(runner)
	}

	return count, nil
}

// NormalizeScore invoked after scoring all runners.
func (pl *RunnerAffinity) NormalizeScore(ctx context.Context, state *framework.CycleState, region *corev1.Region, scores framework.RunnerScoreList) *framework.Status {
	return helper.DefaultNormalizeScore(framework.MaxRunnerScore, false, scores)
}

// ScoreExtensions of the Score plugin.
func (pl *RunnerAffinity) ScoreExtensions() framework.ScoreExtensions {
	return pl
}

// New initializes a new plugin and returns it.
func New(_ context.Context, plArgs runtime.Object, h framework.Handle) (framework.Plugin, error) {
	args, err := getArgs(plArgs)
	if err != nil {
		return nil, err
	}
	pl := &RunnerAffinity{
		handle: h,
	}
	if args.AddedAffinity != nil {
		if ns := args.AddedAffinity.RequiredDuringSchedulingIgnoredDuringExecution; ns != nil {
			pl.addedRunnerSelector, err = runneraffinity.NewRunnerSelector(ns)
			if err != nil {
				return nil, fmt.Errorf("parsing addedAffinity.requiredDuringSchedulingIgnoredDuringExecution: %w", err)
			}
		}
		// TODO: parse requiredDuringSchedulingRequiredDuringExecution when it gets added to the API.
		if terms := args.AddedAffinity.PreferredDuringSchedulingIgnoredDuringExecution; len(terms) != 0 {
			pl.addedPrefSchedTerms, err = runneraffinity.NewPreferredSchedulingTerms(terms)
			if err != nil {
				return nil, fmt.Errorf("parsing addedAffinity.preferredDuringSchedulingIgnoredDuringExecution: %w", err)
			}
		}
	}
	return pl, nil
}

func getArgs(obj runtime.Object) (configv1.RunnerAffinityArgs, error) {
	ptr, ok := obj.(*configv1.RunnerAffinityArgs)
	if !ok {
		return configv1.RunnerAffinityArgs{}, fmt.Errorf("args are not of type RunnerAffinityArgs, got %T", obj)
	}
	return *ptr, nil
}

func getRegionPreferredRunnerAffinity(region *corev1.Region) (*runneraffinity.PreferredSchedulingTerms, error) {
	affinity := region.Spec.Affinity
	if affinity != nil && affinity.RunnerAffinity != nil && affinity.RunnerAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
		return runneraffinity.NewPreferredSchedulingTerms(affinity.RunnerAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}
	return nil, nil
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

func getPreFilterState(cycleState *framework.CycleState) (*preFilterState, error) {
	c, err := cycleState.Read(preFilterStateKey)
	if err != nil {
		return nil, fmt.Errorf("reading %q from cycleState: %v", preFilterStateKey, err)
	}

	s, ok := c.(*preFilterState)
	if !ok {
		return nil, fmt.Errorf("invalid PreFilter state, got type %T", c)
	}
	return s, nil
}
