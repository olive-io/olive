package activity

import "github.com/oliveio/olive/bpmn"

type ActiveBoundaryTrace struct {
	Start bool
	Node  bpmn.FlowNodeInterface
}

func (b ActiveBoundaryTrace) TraceInterface() {}
