/*
Copyright 2025 The olive Authors

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
	"encoding/json"
	"strconv"

	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/v2"

	"github.com/olive-io/olive/api/types"
)

func jsonValueToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case []byte:
		return string(v)
	default:
		data, _ := json.Marshal(value)
		return string(data)
	}
}

func parseTaskType(at bpmn.ActivityType) types.FlowNodeType {
	switch at {
	case bpmn.TaskActivity:
		return types.FlowNodeType_Task
	case bpmn.BusinessRuleActivity:
		return types.FlowNodeType_BusinessRuleTask
	case bpmn.CallActivity:
		return types.FlowNodeType_CallActivity
	case bpmn.ManualTaskActivity:
		return types.FlowNodeType_ManualTask
	case bpmn.ScriptTaskActivity:
		return types.FlowNodeType_ScriptTask
	case bpmn.SendTaskActivity:
		return types.FlowNodeType_SendTask
	case bpmn.ServiceTaskActivity:
		return types.FlowNodeType_ServiceTask
	case bpmn.SubprocessActivity:
		return types.FlowNodeType_SubProcess
	case bpmn.ReceiveTaskActivity:
		return types.FlowNodeType_ReceiveTask
	case bpmn.UserTaskActivity:
		return types.FlowNodeType_UserTask
	default:
		return types.FlowNodeType_UserTask
	}
}

func parseElementType(elem schema.FlowElementInterface) types.FlowNodeType {
	switch elem.(type) {
	case *schema.StartEvent:
		return types.FlowNodeType_StartEvent
	case *schema.EndEvent:
		return types.FlowNodeType_EndEvent
	case *schema.CatchEvent:
		return types.FlowNodeType_IntermediateCatchEvent
	case *schema.BoundaryEvent:
		return types.FlowNodeType_BoundaryEvent
	case *schema.ParallelGateway:
		return types.FlowNodeType_ParallelGateway
	case *schema.InclusiveGateway:
		return types.FlowNodeType_InclusiveGateway
	case *schema.ExclusiveGateway:
		return types.FlowNodeType_ExclusiveGateway
	case *schema.EventBasedGateway:
		return types.FlowNodeType_EventBasedGateway
	case *schema.Task:
		return types.FlowNodeType_Task
	case *schema.BusinessRuleTask:
		return types.FlowNodeType_BusinessRuleTask
	case *schema.CallActivity:
		return types.FlowNodeType_CallActivity
	case *schema.ManualTask:
		return types.FlowNodeType_ManualTask
	case *schema.SendTask:
		return types.FlowNodeType_SendTask
	case *schema.ScriptTask:
		return types.FlowNodeType_ScriptTask
	case *schema.ServiceTask:
		return types.FlowNodeType_ServiceTask
	case *schema.SubProcess:
		return types.FlowNodeType_SubProcess
	case *schema.ReceiveTask:
		return types.FlowNodeType_ReceiveTask
	case *schema.UserTask:
		return types.FlowNodeType_UserTask
	default:
		return types.FlowNodeType_UnknownNode
	}
}
