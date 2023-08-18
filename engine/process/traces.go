package process

import (
	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/tracing"
)

// Trace wraps any trace within a given process
type Trace struct {
	Process *bpmn.Process
	Trace   tracing.Trace
}

func (t Trace) Unwrap() tracing.Trace {
	return t.Trace
}

func (t Trace) TraceInterface() {}
