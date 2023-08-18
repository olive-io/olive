package catch

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/event"
)

type ActiveListeningTrace struct {
	Node *bpmn.CatchEvent
}

func (t ActiveListeningTrace) TraceInterface() {}

// EventObservedTrace signals the fact that a particular event
// has been in fact observed by the node
type EventObservedTrace struct {
	Node  *bpmn.CatchEvent
	Event event.Event
}

func (t EventObservedTrace) TraceInterface() {}
