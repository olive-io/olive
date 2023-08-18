package parallel

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/flow/flow_interface"
)

// IncomingFlowProcessedTrace signals that a particular flow
// has been processed. If any action have been taken, it already happened
type IncomingFlowProcessedTrace struct {
	Node *bpmn.ParallelGateway
	Flow flow_interface.T
}

func (t IncomingFlowProcessedTrace) TraceInterface() {}
