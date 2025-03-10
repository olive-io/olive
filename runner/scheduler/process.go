/*
Copyright 2024 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package scheduler

import (
	"context"
	"encoding/xml"

	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/data"
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

type ProcessHandler func(trace tracing.ITrace, locator data.IFlowDataLocator)

func (p *Process) Run(handler ProcessHandler) error {
	ctx := p.ctx
	traces := p.tracer.Subscribe()

	defer func() {
		p.tracer.Unsubscribe(traces)
		p.cancel()
	}()

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

			handler(trace, instance.Locator())
			if _, ok := trace.(bpmn.CeaseFlowTrace); ok {
				break LOOP
			}
		}

		instance.WaitUntilComplete(ctx)
	}

	return nil
}
