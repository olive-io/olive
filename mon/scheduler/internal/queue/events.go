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

package queue

import (
	"github.com/olive-io/olive/mon/scheduler/framework"
)

const (
	// DefinitionAdd is the event when a new pod is added to API server.
	DefinitionAdd = "DefinitionAdd"
	// ScheduleAttemptFailure is the event when a schedule attempt fails.
	ScheduleAttemptFailure = "ScheduleAttemptFailure"
	// BackoffComplete is the event when a pod finishes backoff.
	BackoffComplete = "BackoffComplete"
	// ForceActivate is the event when a pod is moved from unschedulableDefinitions/backoffQ
	// to activeQ. Usually it's triggered by plugin implementations.
	ForceActivate = "ForceActivate"
	// DefinitionUpdate is the event when a pod is updated
	DefinitionUpdate = "DefinitionUpdate"
)

var (
	// AssignedDefinitionAdd is the event when a pod is added that causes pods with matching affinity terms
	// to be more schedulable.
	AssignedDefinitionAdd = framework.ClusterEvent{Resource: framework.Definition, ActionType: framework.Add, Label: "AssignedDefinitionAdd"}
	// RunnerAdd is the event when a new node is added to the cluster.
	RunnerAdd = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add, Label: "RunnerAdd"}
	// AssignedDefinitionUpdate is the event when a pod is updated that causes pods with matching affinity
	// terms to be more schedulable.
	AssignedDefinitionUpdate = framework.ClusterEvent{Resource: framework.Definition, ActionType: framework.Update, Label: "AssignedDefinitionUpdate"}
	// AssignedDefinitionDelete is the event when a pod is deleted that causes pods with matching affinity
	// terms to be more schedulable.
	AssignedDefinitionDelete = framework.ClusterEvent{Resource: framework.Definition, ActionType: framework.Delete, Label: "AssignedDefinitionDelete"}
	// RunnerSpecUnschedulableChange is the event when unschedulable node spec is changed.
	RunnerSpecUnschedulableChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint, Label: "RunnerSpecUnschedulableChange"}
	// RunnerAllocatableChange is the event when node allocatable is changed.
	RunnerAllocatableChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerAllocatable, Label: "RunnerAllocatableChange"}
	// RunnerLabelChange is the event when node label is changed.
	RunnerLabelChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerLabel, Label: "RunnerLabelChange"}
	// RunnerAnnotationChange is the event when node annotation is changed.
	RunnerAnnotationChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerAnnotation, Label: "RunnerAnnotationChange"}
	// RunnerTaintChange is the event when node taint is changed.
	RunnerTaintChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint, Label: "RunnerTaintChange"}
	// RunnerConditionChange is the event when node condition is changed.
	RunnerConditionChange = framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.UpdateRunnerCondition, Label: "RunnerConditionChange"}
	// WildCardEvent semantically matches all resources on all actions.
	WildCardEvent = framework.ClusterEvent{Resource: framework.WildCard, ActionType: framework.All, Label: "WildCardEvent"}
	// UnschedulableTimeout is the event when a pod stays in unschedulable for longer than timeout.
	UnschedulableTimeout = framework.ClusterEvent{Resource: framework.WildCard, ActionType: framework.All, Label: "UnschedulableTimeout"}
)
