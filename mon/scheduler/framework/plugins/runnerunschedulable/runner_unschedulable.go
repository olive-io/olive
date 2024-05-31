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

package runnerunschedulable

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
	"github.com/olive-io/olive/mon/scheduler/util"
	v1helper "github.com/olive-io/olive/pkg/scheduling/corev1"
)

// RunnerUnschedulable plugin filters runners that set runner.Spec.Unschedulable=true unless
// the region tolerates {key=runner.kubernetes.io/unschedulable, effect:NoSchedule} taint.
type RunnerUnschedulable struct {
}

var _ framework.FilterPlugin = &RunnerUnschedulable{}
var _ framework.EnqueueExtensions = &RunnerUnschedulable{}

// Name is the name of the plugin used in the plugin registry and configurations.
const Name = names.RunnerUnschedulable

const (
	// ErrReasonUnknownCondition is used for RunnerUnknownCondition predicate error.
	ErrReasonUnknownCondition = "runner(s) had unknown conditions"
	// ErrReasonUnschedulable is used for RunnerUnschedulable predicate error.
	ErrReasonUnschedulable = "runner(s) were unschedulable"
)

// EventsToRegister returns the possible events that may make a Region
// failed by this plugin schedulable.
func (pl *RunnerUnschedulable) EventsToRegister() []framework.ClusterEventWithHint {
	return []framework.ClusterEventWithHint{
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.Update}, QueueingHintFn: pl.isSchedulableAfterRunnerChange},
	}
}

// isSchedulableAfterRunnerChange is invoked for all runner events reported by
// an informer. It checks whether that change made a previously unschedulable
// region schedulable.
func (pl *RunnerUnschedulable) isSchedulableAfterRunnerChange(logger klog.Logger, region *corev1.Region, oldObj, newObj interface{}) (framework.QueueingHint, error) {
	_, modifiedRunner, err := util.As[*corev1.Runner](oldObj, newObj)
	if err != nil {
		return framework.Queue, err
	}

	if !modifiedRunner.Spec.Unschedulable {
		logger.V(5).Info("runner was created or updated, region may be schedulable now", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
		return framework.Queue, nil
	}

	logger.V(5).Info("runner was created or updated, but it doesn't make this region schedulable", "region", klog.KObj(region), "runner", klog.KObj(modifiedRunner))
	return framework.QueueSkip, nil
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *RunnerUnschedulable) Name() string {
	return Name
}

// Filter invoked at the filter extension point.
func (pl *RunnerUnschedulable) Filter(ctx context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
	runner := runnerInfo.Runner()

	if !runner.Spec.Unschedulable {
		return nil
	}

	// If region tolerate unschedulable taint, it's also tolerate `runner.Spec.Unschedulable`.
	regionToleratesUnschedulable := v1helper.TolerationsTolerateTaint(region.Spec.Tolerations, &corev1.Taint{
		Key:    corev1.TaintRunnerUnschedulable,
		Effect: corev1.TaintEffectNoSchedule,
	})
	if !regionToleratesUnschedulable {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonUnschedulable)
	}

	return nil
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &RunnerUnschedulable{}, nil
}
