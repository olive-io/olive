package timer

import (
	"context"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/clock"
	"github.com/oliveio/olive/engine/event"
	"github.com/oliveio/olive/engine/tracing"
)

type eventDefinitionInstanceBuilder struct {
	// context here keeps context from the creation of the instance builder
	// It is not a final decision, but it currently seems to make more sense
	// to "attach" it to this context instead of the context that can be passed
	// through event.DefinitionInstance.NewEventDefinitionInstance. Time will tell.
	context      context.Context
	eventIngress event.Consumer
	tracer       tracing.Tracer
}

type eventDefinitionInstance struct {
	definition bpmn.TimerEventDefinition
}

func (e *eventDefinitionInstance) EventDefinition() bpmn.EventDefinitionInterface {
	return &e.definition
}

func (e *eventDefinitionInstanceBuilder) NewEventDefinitionInstance(def bpmn.EventDefinitionInterface) (definitionInstance event.DefinitionInstance, err error) {
	if timerEventDefinition, ok := def.(*bpmn.TimerEventDefinition); ok {
		var c clock.Clock
		c, err = clock.FromContext(e.context)
		if err != nil {
			return
		}
		var timer chan bpmn.TimerEventDefinition
		timer, err = New(e.context, c, *timerEventDefinition)
		if err != nil {
			return
		}
		definitionInstance = &eventDefinitionInstance{*timerEventDefinition}
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case _, ok := <-timer:
					if !ok {
						return
					}
					_, err := e.eventIngress.ConsumeEvent(event.MakeTimerEvent(definitionInstance))
					if err != nil {
						e.tracer.Trace(tracing.ErrorTrace{Error: err})
					}
				}
			}
		}(e.context)
	}
	return
}

func EventDefinitionInstanceBuilder(
	ctx context.Context,
	eventIngress event.Consumer,
	tracer tracing.Tracer,
) event.DefinitionInstanceBuilder {
	return &eventDefinitionInstanceBuilder{
		context:      ctx,
		eventIngress: eventIngress,
		tracer:       tracer,
	}
}
