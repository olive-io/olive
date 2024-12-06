package controller

import (
	"context"
	"encoding/xml"

	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/tracing"
)

type Process struct {
	ctx    context.Context
	cancel context.CancelFunc

	id string

	definitions *schema.Definitions
	tracer      tracing.ITracer
	processes   []*bpmn.Process
	instances   []*bpmn.Instance
}

func NewProcess(ctx context.Context, data []byte, opts ...bpmn.Option) (*Process, error) {

	var definitions schema.Definitions
	err := xml.Unmarshal(data, &definitions)
	if err != nil {
		return nil, err
	}

	pctx, cancel := context.WithCancel(ctx)
	tracer := tracing.NewTracer(ctx)

	instances := make([]*bpmn.Instance, 0)

	opts = append(opts, bpmn.WithContext(ctx), bpmn.WithTracer(tracer))
	options := bpmn.NewOptions(opts...)

	var id string
	for i := range *definitions.Processes() {
		element := (*definitions.Processes())[i]
		executable, ok := element.IsExecutable()
		if !ok || !executable {
			continue
		}

		instance, err := bpmn.NewInstance(&element, &definitions, options)
		if err != nil {
			cancel()
			return nil, err
		}

		if id == "" {
			id = instance.Id().String()
		}

		instances = append(instances, instance)
	}

	p := &Process{
		ctx:         pctx,
		cancel:      cancel,
		definitions: &definitions,
		tracer:      tracer,
		instances:   instances,
	}

	return p, nil
}

func (p *Process) ID() string {
	return p.id
}

func (p *Process) Score() int64 {
	return 100
}

type ProcessHandler func(trace tracing.ITrace)

func (p *Process) Run(handler ProcessHandler) error {
	ctx := p.ctx
	traces := p.tracer.Subscribe()

	defer p.tracer.Unsubscribe(traces)
	defer p.cancel()

	for _, instance := range p.instances {
		if err := instance.StartAll(); err != nil {
			return err
		}

	LOOP:
		for {
			wrapped, ok := <-traces
			if !ok {
				break LOOP
			}

			trace := tracing.Unwrap(wrapped)

			handler(trace)
			if _, ok := trace.(bpmn.CeaseFlowTrace); ok {
				break LOOP
			}
		}

		instance.WaitUntilComplete(ctx)
	}

	return nil
}
