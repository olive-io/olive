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
		return types.FlowNodeType_StartEvent
	case *schema.EndEvent:
		return types.FlowNodeType_EndEvent
	case *schema.BoundaryEvent:
		return types.FlowNodeType_BoundaryEvent
	case *schema.IntermediateCatchEvent:
		return types.FlowNodeType_IntermediateCatchEvent
	case *schema.Task:
		return types.FlowNodeType_Task
	case *schema.SendTask:
		return types.FlowNodeType_SendTask
	case *schema.ReceiveTask:
		return types.FlowNodeType_ReceiveTask
	case *schema.ServiceTask:
		return types.FlowNodeType_ServiceTask
	case *schema.UserTask:
		return types.FlowNodeType_UserTask
	case *schema.ScriptTask:
		return types.FlowNodeType_ScriptTask
	case *schema.ManualTask:
		return types.FlowNodeType_ManualTask
	case *schema.CallActivity:
		return types.FlowNodeType_CallActivity
	case *schema.BusinessRuleTask:
		return types.FlowNodeType_BusinessRuleTask
	case *schema.SubProcess:
		return types.FlowNodeType_SubProcess
	case *schema.EventBasedGateway:
		return types.FlowNodeType_EventBasedGateway
	case *schema.ExclusiveGateway:
		return types.FlowNodeType_ExclusiveGateway
	case *schema.InclusiveGateway:
		return types.FlowNodeType_InclusiveGateway
	case *schema.ParallelGateway:
		return types.FlowNodeType_ParallelGateway
	case bpmn.IFlowNode:
		if vv, ok := tv.(*bpmn.Harness); ok {
			return flowType(vv.Activity().Element())
		}
		return types.FlowNodeType_UnknownNode
	default:
		return types.FlowNodeType_UnknownNode
	}
}

func isEvent(node schema.FlowNodeInterface) bool {
	switch node.(type) {
	case *schema.StartEvent,
		*schema.EndEvent,
		*schema.BoundaryEvent,
		*schema.IntermediateCatchEvent:
		return true
	default:
		return false
	}
}
