package model

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/event"
)

type EventInstantiationAttemptedTrace struct {
	Event   event.Event
	Element bpmn.FlowNodeInterface
}

func (e EventInstantiationAttemptedTrace) TraceInterface() {}
