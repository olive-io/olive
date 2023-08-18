package flow_node

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/sequence_flow"
)

type Outgoing interface {
	NextAction(flow flow_interface.T) chan Action
}

type FlowNodeInterface interface {
	Outgoing
	Element() bpmn.FlowNodeInterface
}

func AllSequenceFlows(
	sequenceFlows *[]sequence_flow.SequenceFlow,
	exclusion ...func(*sequence_flow.SequenceFlow) bool,
) (result []*sequence_flow.SequenceFlow) {
	result = make([]*sequence_flow.SequenceFlow, 0)
sequenceFlowsLoop:
	for i := range *sequenceFlows {
		for _, exclFun := range exclusion {
			if exclFun(&(*sequenceFlows)[i]) {
				continue sequenceFlowsLoop
			}
		}
		result = append(result, &(*sequenceFlows)[i])
	}
	return
}
