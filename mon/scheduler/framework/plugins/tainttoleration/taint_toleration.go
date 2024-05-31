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

package tainttoleration

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/helper"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
	v1helper "github.com/olive-io/olive/pkg/scheduling/corev1"
)

// TaintToleration is a plugin that checks if a region tolerates a runner's taints.
type TaintToleration struct {
	handle framework.Handle
}

var _ framework.FilterPlugin = &TaintToleration{}
var _ framework.PreScorePlugin = &TaintToleration{}
var _ framework.ScorePlugin = &TaintToleration{}
var _ framework.EnqueueExtensions = &TaintToleration{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = names.TaintToleration
	// preScoreStateKey is the key in CycleState to TaintToleration pre-computed data for Scoring.
	preScoreStateKey = "PreScore" + Name
	// ErrReasonNotMatch is the Filter reason status when not matching.
	ErrReasonNotMatch = "runner(s) had taints that the region didn't tolerate"
)

// Name returns name of the plugin. It is used in logs, etc.
func (pl *TaintToleration) Name() string {
	return Name
}

// EventsToRegister returns the possible events that may make a Region
// failed by this plugin schedulable.
func (pl *TaintToleration) EventsToRegister() []framework.ClusterEventWithHint {
	return []framework.ClusterEventWithHint{
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.Update}},
	}
}

// Filter invoked at the filter extension point.
func (pl *TaintToleration) Filter(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	runner := runnerInfo.Runner()

	taint, isUntolerated := v1helper.FindMatchingUntoleratedTaint(runner.Spec.Taints, region.Spec.Tolerations, helper.DoNotScheduleTaintsFilterFunc())
	if !isUntolerated {
		return nil
	}

	errReason := fmt.Sprintf("runner(s) had untolerated taint {%s: %s}", taint.Key, taint.Value)
	return framework.NewStatus(framework.UnschedulableAndUnresolvable, errReason)
}

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	tolerationsPreferNoSchedule []corev1.Toleration
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// getAllTolerationEffectPreferNoSchedule gets the list of all Tolerations with Effect PreferNoSchedule or with no effect.
func getAllTolerationPreferNoSchedule(tolerations []corev1.Toleration) (tolerationList []corev1.Toleration) {
	for _, toleration := range tolerations {
		// Empty effect means all effects which includes PreferNoSchedule, so we need to collect it as well.
		if len(toleration.Effect) == 0 || toleration.Effect == corev1.TaintEffectPreferNoSchedule {
			tolerationList = append(tolerationList, toleration)
		}
	}
	return
}

// PreScore builds and writes cycle state used by Score and NormalizeScore.
func (pl *TaintToleration) PreScore(ctx context.Context, cycleState *framework.CycleState, region *corev1.Region, runners []*framework.RunnerInfo) *framework.Status {
	if len(runners) == 0 {
		return nil
	}
	tolerationsPreferNoSchedule := getAllTolerationPreferNoSchedule(region.Spec.Tolerations)
	state := &preScoreState{
		tolerationsPreferNoSchedule: tolerationsPreferNoSchedule,
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
		return nil, fmt.Errorf("%+v convert to tainttoleration.preScoreState error", c)
	}
	return s, nil
}

// CountIntolerableTaintsPreferNoSchedule gives the count of intolerable taints of a region with effect PreferNoSchedule
func countIntolerableTaintsPreferNoSchedule(taints []corev1.Taint, tolerations []corev1.Toleration) (intolerableTaints int) {
	for _, taint := range taints {
		// check only on taints that have effect PreferNoSchedule
		if taint.Effect != corev1.TaintEffectPreferNoSchedule {
			continue
		}

		if !v1helper.TolerationsTolerateTaint(tolerations, &taint) {
			intolerableTaints++
		}
	}
	return
}

// Score invoked at the Score extension point.
func (pl *TaintToleration) Score(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerName string) (int64, *framework.Status) {
	runnerInfo, err := pl.handle.SnapshotSharedLister().RunnerInfos().Get(runnerName)
	if err != nil {
		return 0, framework.AsStatus(fmt.Errorf("getting runner %q from Snapshot: %w", runnerName, err))
	}
	runner := runnerInfo.Runner()

	s, err := getPreScoreState(state)
	if err != nil {
		return 0, framework.AsStatus(err)
	}

	score := int64(countIntolerableTaintsPreferNoSchedule(runner.Spec.Taints, s.tolerationsPreferNoSchedule))
	return score, nil
}

// NormalizeScore invoked after scoring all runners.
func (pl *TaintToleration) NormalizeScore(ctx context.Context, _ *framework.CycleState, region *corev1.Region, scores framework.RunnerScoreList) *framework.Status {
	return helper.DefaultNormalizeScore(framework.MaxRunnerScore, true, scores)
}

// ScoreExtensions of the Score plugin.
func (pl *TaintToleration) ScoreExtensions() framework.ScoreExtensions {
	return pl
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, h framework.Handle) (framework.Plugin, error) {
	return &TaintToleration{handle: h}, nil
}
