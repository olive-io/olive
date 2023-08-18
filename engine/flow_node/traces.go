package flow_node

import "github.com/oliveio/olive/bpmn"

type CancellationTrace struct {
	Node bpmn.FlowNodeInterface
}

func (t CancellationTrace) TraceInterface() {}

type NewFlowNodeTrace struct {
	Node bpmn.FlowNodeInterface
}

func (t NewFlowNodeTrace) TraceInterface() {}
