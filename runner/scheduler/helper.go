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

package scheduler

import (
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/v2"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
)

func flowId(node schema.FlowNodeInterface) (id string) {
	ptr, _ := node.Id()
	if ptr != nil {
		id = *ptr
	}
	return
}

func flowType(node schema.FlowNodeInterface) corev1.FlowNodeType {
	switch tv := node.(type) {
	case *schema.StartEvent:
		return corev1.StartEvent
	case *schema.EndEvent:
		return corev1.EndEvent
	case *schema.BoundaryEvent:
		return corev1.BoundaryEvent
	case *schema.IntermediateCatchEvent:
		return corev1.IntermediateCatchEvent
	case *schema.Task:
		return corev1.Task
	case *schema.SendTask:
		return corev1.SendTask
	case *schema.ReceiveTask:
		return corev1.ReceiveTask
	case *schema.ServiceTask:
		return corev1.ServiceTask
	case *schema.UserTask:
		return corev1.UserTask
	case *schema.ScriptTask:
		return corev1.ScriptTask
	case *schema.ManualTask:
		return corev1.ManualTask
	case *schema.CallActivity:
		return corev1.CallActivity
	case *schema.BusinessRuleTask:
		return corev1.BusinessRuleTask
	case *schema.SubProcess:
		return corev1.SendTask
	case *schema.EventBasedGateway:
		return corev1.EventBasedGateway
	case *schema.ExclusiveGateway:
		return corev1.ExclusiveGateway
	case *schema.InclusiveGateway:
		return corev1.InclusiveGateway
	case *schema.ParallelGateway:
		return corev1.ParallelGateway
	case bpmn.IFlowNode:
		if vv, ok := tv.(*bpmn.Harness); ok {
			return flowType(vv.Activity().Element())
		}
		return corev1.FlowNodeUnknown
	default:
		return corev1.FlowNodeUnknown
	}
}

func isEvent(node schema.FlowNodeInterface) bool {
	switch node.(type) {
	case *schema.StartEvent:
		return true
	case *schema.EndEvent:
		return true
	case *schema.BoundaryEvent:
		return true
	case *schema.IntermediateCatchEvent:
		return true
	default:
		return false
	}
}
