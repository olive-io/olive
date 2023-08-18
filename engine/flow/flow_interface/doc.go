package flow_interface

import (
	"github.com/oliveio/olive/engine/id"
	"github.com/oliveio/olive/engine/sequence_flow"
)

// T specifies an interface for BPMN flows
type T interface {
	// Id returns flow's unique identifier
	Id() id.Id
	// SequenceFlow returns an inbound sequence flow this flow
	// is currently at.
	SequenceFlow() *sequence_flow.SequenceFlow
}
