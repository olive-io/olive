package event_based

import "github.com/oliveio/olive/bpmn"

type DeterminationMadeTrace struct {
	bpmn.Element
}

func (trace DeterminationMadeTrace) TraceInterface() {}
