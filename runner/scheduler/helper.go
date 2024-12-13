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

	"github.com/olive-io/olive/api/types"
)

func flowId(node schema.FlowNodeInterface) (id string) {
	ptr, _ := node.Id()
	if ptr != nil {
		id = *ptr
	}
	return
}

func flowType(node schema.FlowNodeInterface) types.FlowNodeType {
	switch tv := node.(type) {
	case *schema.StartEvent:
		return types.StartEvent
	case *schema.EndEvent:
		return types.EndEvent
	case *schema.BoundaryEvent:
		return types.BoundaryEvent
	case *schema.IntermediateCatchEvent:
		return types.IntermediateCatchEvent
	case *schema.Task:
		return types.Task
	case *schema.SendTask:
		return types.SendTask
	case *schema.ReceiveTask:
		return types.ReceiveTask
	case *schema.ServiceTask:
		return types.ServiceTask
	case *schema.UserTask:
		return types.UserTask
	case *schema.ScriptTask:
		return types.ScriptTask
	case *schema.ManualTask:
		return types.ManualTask
	case *schema.CallActivity:
		return types.CallActivity
	case *schema.BusinessRuleTask:
		return types.BusinessRuleTask
	case *schema.SubProcess:
		return types.SendTask
	case *schema.EventBasedGateway:
		return types.EventBasedGateway
	case *schema.ExclusiveGateway:
		return types.ExclusiveGateway
	case *schema.InclusiveGateway:
		return types.InclusiveGateway
	case *schema.ParallelGateway:
		return types.ParallelGateway
	case bpmn.IFlowNode:
		if vv, ok := tv.(*bpmn.Harness); ok {
			return flowType(vv.Activity().Element())
		}
		return types.FlowNodeUnknown
	default:
		return types.FlowNodeUnknown
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
