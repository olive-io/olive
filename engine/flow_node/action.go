package flow_node

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/sequence_flow"
)

type Action interface {
	action()
}

type ProbeAction struct {
	SequenceFlows []*sequence_flow.SequenceFlow
	// ProbeReport is a function that needs to be called
	// wth sequence flow indices that have successful
	// condition expressions (or none)
	ProbeReport func([]int)
}

func (action ProbeAction) action() {}

type ActionTransformer func(sequenceFlowId *bpmn.IdRef, action Action) Action
type Terminate func(sequenceFlowId *bpmn.IdRef) chan bool

type FlowAction struct {
	SequenceFlows []*sequence_flow.SequenceFlow
	// Index of sequence flows that should flow without
	// conditionExpression being evaluated
	UnconditionalFlows []int
	// The actions produced by the targets should be processed by
	// this function
	ActionTransformer
	// If supplied channel sends a function that returns true, the flow action
	// is to be terminated if it wasn't already
	Terminate
}

func (action FlowAction) action() {}

type CompleteAction struct{}

func (action CompleteAction) action() {}

type NoAction struct{}

func (action NoAction) action() {}
