package activity

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/data"
	"github.com/oliveio/olive/engine/event"
	"github.com/oliveio/olive/engine/flow"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/flow_node"
	"github.com/oliveio/olive/engine/flow_node/event/catch"
	"github.com/oliveio/olive/engine/id"
	"github.com/oliveio/olive/engine/tracing"
)

type message interface {
	message()
}

type nextActionMessage struct {
	flow     flow_interface.T
	response chan chan flow_node.Action
}

func (m nextActionMessage) message() {}

type Harness struct {
	*flow_node.Wiring
	element            bpmn.FlowNodeInterface
	runnerChannel      chan message
	activity           Activity
	active             int32
	cancellation       sync.Once
	eventConsumers     []event.Consumer
	eventConsumersLock sync.RWMutex
}

func (node *Harness) ConsumeEvent(ev event.Event) (result event.ConsumptionResult, err error) {
	node.eventConsumersLock.RLock()
	defer node.eventConsumersLock.RUnlock()
	if atomic.LoadInt32(&node.active) == 1 {
		result, err = event.ForwardEvent(ev, &node.eventConsumers)
	}
	return
}

func (node *Harness) RegisterEventConsumer(consumer event.Consumer) (err error) {
	node.eventConsumersLock.Lock()
	defer node.eventConsumersLock.Unlock()
	node.eventConsumers = append(node.eventConsumers, consumer)
	return
}

func (node *Harness) Activity() Activity {
	return node.activity
}

type Constructor = func(*flow_node.Wiring) (node Activity, err error)

func NewHarness(ctx context.Context,
	wiring *flow_node.Wiring,
	element *bpmn.FlowNode,
	idGenerator id.Generator,
	constructor Constructor,
	itemAwareLocator data.ItemAwareLocator,
) (node *Harness, err error) {
	var activity Activity
	activity, err = constructor(wiring)
	if err != nil {
		return
	}

	boundaryEvents := make([]*bpmn.BoundaryEvent, 0)

	for i := range *wiring.Process.BoundaryEvents() {
		boundaryEvent := &(*wiring.Process.BoundaryEvents())[i]
		if *boundaryEvent.AttachedToRef() == wiring.FlowNodeId {
			boundaryEvents = append(boundaryEvents, boundaryEvent)
		}
	}

	node = &Harness{
		Wiring:        wiring,
		element:       element,
		runnerChannel: make(chan message, len(wiring.Incoming)*2+1),
		activity:      activity,
	}

	err = node.EventEgress.RegisterEventConsumer(node)
	if err != nil {
		return
	}

	for i := range boundaryEvents {
		boundaryEvent := boundaryEvents[i]
		var catchEventFlowNode *flow_node.Wiring
		catchEventFlowNode, err = wiring.CloneFor(&boundaryEvent.FlowNode)
		if err != nil {
			return
		}
		// this node becomes event egress
		catchEventFlowNode.EventEgress = node

		var catchEvent *catch.Node
		catchEvent, err = catch.New(ctx, catchEventFlowNode, &boundaryEvent.CatchEvent)
		if err != nil {
			return
		} else {
			var actionTransformer flow_node.ActionTransformer
			if boundaryEvent.CancelActivity() {
				actionTransformer = func(sequenceFlowId *bpmn.IdRef, action flow_node.Action) flow_node.Action {
					node.cancellation.Do(func() {
						<-node.activity.Cancel()
					})
					return action
				}
			}
			newFlow := flow.New(node.Definitions, catchEvent, node.Tracer,
				node.FlowNodeMapping, node.FlowWaitGroup, idGenerator, actionTransformer, itemAwareLocator)
			newFlow.Start(ctx)
		}
	}
	sender := node.Tracer.RegisterSender()
	go node.runner(ctx, sender)
	return
}

func (node *Harness) runner(ctx context.Context, sender tracing.SenderHandle) {
	defer sender.Done()

	for {
		select {
		case msg := <-node.runnerChannel:
			switch m := msg.(type) {
			case nextActionMessage:
				atomic.StoreInt32(&node.active, 1)
				node.Tracer.Trace(ActiveBoundaryTrace{Start: true, Node: node.activity.Element()})
				in := node.activity.NextAction(m.flow)
				out := make(chan flow_node.Action)
				go func(ctx context.Context) {
					select {
					case out <- <-in:
						atomic.StoreInt32(&node.active, 0)
						node.Tracer.Trace(ActiveBoundaryTrace{Start: false, Node: node.activity.Element()})
					case <-ctx.Done():
						return
					}
				}(ctx)
				m.response <- out
			default:
			}
		case <-ctx.Done():
			node.Tracer.Trace(flow_node.CancellationTrace{Node: node.element})
			return
		}
	}
}

func (node *Harness) NextAction(flow flow_interface.T) chan flow_node.Action {
	response := make(chan chan flow_node.Action)
	node.runnerChannel <- nextActionMessage{flow: flow, response: response}
	return <-response
}

func (node *Harness) Element() bpmn.FlowNodeInterface {
	return node.element
}
