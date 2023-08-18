package flow

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/id"
)

type NewFlowTrace struct {
	FlowId id.Id
}

func (t NewFlowTrace) TraceInterface() {}

type Trace struct {
	Source bpmn.FlowNodeInterface
	Flows  []Snapshot
}

func (t Trace) TraceInterface() {}

type TerminationTrace struct {
	FlowId id.Id
	Source bpmn.FlowNodeInterface
}

func (t TerminationTrace) TraceInterface() {}

type CancellationTrace struct {
	FlowId id.Id
}

func (t CancellationTrace) TraceInterface() {}

type CompletionTrace struct {
	Node bpmn.FlowNodeInterface
}

func (t CompletionTrace) TraceInterface() {}

type CeaseFlowTrace struct{}

func (t CeaseFlowTrace) TraceInterface() {}

type VisitTrace struct {
	Node bpmn.FlowNodeInterface
}

func (t VisitTrace) TraceInterface() {}
