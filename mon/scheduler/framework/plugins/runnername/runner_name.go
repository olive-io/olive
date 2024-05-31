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

package runnername

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/plugins/names"
)

// RunnerName is a plugin that checks if a region spec runner name matches the current runner.
type RunnerName struct{}

var _ framework.FilterPlugin = &RunnerName{}
var _ framework.EnqueueExtensions = &RunnerName{}

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name = names.RunnerName

	// ErrReason returned when runner name doesn't match.
	ErrReason = "runner(s) didn't match the requested runner name"
)

// EventsToRegister returns the possible events that may make a Region
// failed by this plugin schedulable.
func (pl *RunnerName) EventsToRegister() []framework.ClusterEventWithHint {
	return []framework.ClusterEventWithHint{
		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add | framework.Update}},
	}
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *RunnerName) Name() string {
	return Name
}

// Filter invoked at the filter extension point.
func (pl *RunnerName) Filter(ctx context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {

	if !Fits(region, runnerInfo) {
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReason)
	}
	return nil
}

// Fits actually checks if the region fits the runner.
func Fits(region *corev1.Region, runnerInfo *framework.RunnerInfo) bool {
	return len(region.Spec.RunnerNames) == 0 ||
		sets.NewString(region.Spec.RunnerNames...).Has(runnerInfo.Runner().Name)
}

// New initializes a new plugin and returns it.
func New(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
	return &RunnerName{}, nil
}
