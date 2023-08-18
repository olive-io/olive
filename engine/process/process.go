package process

import (
	"context"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/event"
	"github.com/oliveio/olive/engine/id"
	"github.com/oliveio/olive/engine/process/instance"
	"github.com/oliveio/olive/engine/tracing"
)

type Process struct {
	Element                        *bpmn.Process
	Definitions                    *bpmn.Definitions
	instances                      []*instance.Instance
	EventIngress                   event.Consumer
	EventEgress                    event.Source
	idGeneratorBuilder             id.GeneratorBuilder
	eventDefinitionInstanceBuilder event.DefinitionInstanceBuilder
	Tracer                         tracing.Tracer
	subTracerMaker                 func() tracing.Tracer
}

type Option func(context.Context, *Process) context.Context

func WithIdGenerator(builder id.GeneratorBuilder) Option {
	return func(ctx context.Context, process *Process) context.Context {
		process.idGeneratorBuilder = builder
		return ctx
	}
}

func WithEventIngress(consumer event.Consumer) Option {
	return func(ctx context.Context, process *Process) context.Context {
		process.EventIngress = consumer
		return ctx
	}
}

func WithEventEgress(source event.Source) Option {
	return func(ctx context.Context, process *Process) context.Context {
		process.EventEgress = source
		return ctx
	}
}

func WithEventDefinitionInstanceBuilder(builder event.DefinitionInstanceBuilder) Option {
	return func(ctx context.Context, process *Process) context.Context {
		process.eventDefinitionInstanceBuilder = builder
		return ctx
	}
}

// WithTracer overrides process's tracer
func WithTracer(tracer tracing.Tracer) Option {
	return func(ctx context.Context, process *Process) context.Context {
		process.Tracer = tracer
		return ctx
	}
}

// WithContext will pass a given context to a new process
// instead of implicitly generated one
func WithContext(newCtx context.Context) Option {
	return func(ctx context.Context, process *Process) context.Context {
		return newCtx
	}
}

func Make(element *bpmn.Process, definitions *bpmn.Definitions, options ...Option) Process {
	process := Process{
		Element:     element,
		Definitions: definitions,
		instances:   make([]*instance.Instance, 0),
	}

	ctx := context.Background()

	for _, option := range options {
		ctx = option(ctx, &process)
	}

	if process.idGeneratorBuilder == nil {
		process.idGeneratorBuilder = id.DefaultIdGeneratorBuilder
	}

	if process.eventDefinitionInstanceBuilder == nil {
		process.eventDefinitionInstanceBuilder = event.WrappingDefinitionInstanceBuilder
	}

	if process.EventIngress == nil && process.EventEgress == nil {
		fanOut := event.NewFanOut()
		process.EventIngress = fanOut
		process.EventEgress = fanOut
	}

	if process.Tracer == nil {
		process.Tracer = tracing.NewTracer(ctx)
	}

	process.subTracerMaker = func() tracing.Tracer {
		subTracer := tracing.NewTracer(ctx)
		tracing.NewRelay(ctx, subTracer, process.Tracer, func(trace tracing.Trace) []tracing.Trace {
			return []tracing.Trace{Trace{
				Process: process.Element,
				Trace:   trace,
			}}
		})
		return subTracer
	}

	return process
}

func New(element *bpmn.Process, definitions *bpmn.Definitions, options ...Option) *Process {
	process := Make(element, definitions, options...)
	return &process
}

func (process *Process) Instantiate(options ...instance.Option) (inst *instance.Instance, err error) {
	subTracer := process.subTracerMaker()

	options = append([]instance.Option{
		instance.WithIdGenerator(process.idGeneratorBuilder),
		instance.WithEventDefinitionInstanceBuilder(process.eventDefinitionInstanceBuilder),
		instance.WithEventEgress(process.EventEgress),
		instance.WithEventIngress(process.EventIngress),
		instance.WithTracer(subTracer),
	}, options...)
	inst, err = instance.NewInstance(process.Element, process.Definitions, options...)
	if err != nil {
		return
	}

	return
}
