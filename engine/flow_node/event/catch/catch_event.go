package catch

import (
	"context"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/event"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/flow_node"
	"github.com/oliveio/olive/engine/logic"
	"github.com/oliveio/olive/engine/tracing"
)

type message interface {
	message()
}

type nextActionMessage struct {
	response chan flow_node.Action
}

func (m nextActionMessage) message() {}

type processEventMessage struct {
	event event.Event
}

func (m processEventMessage) message() {}

type Node struct {
	*flow_node.Wiring
	element         *bpmn.CatchEvent
	runnerChannel   chan message
	activated       bool
	awaitingActions []chan flow_node.Action
	satisfier       *logic.CatchEventSatisfier
}

func New(ctx context.Context, wiring *flow_node.Wiring, catchEvent *bpmn.CatchEvent) (node *Node, err error) {
	node = &Node{
		Wiring:          wiring,
		element:         catchEvent,
		runnerChannel:   make(chan message, len(wiring.Incoming)*2+1),
		activated:       false,
		awaitingActions: make([]chan flow_node.Action, 0),
		satisfier:       logic.NewCatchEventSatisfier(catchEvent, wiring.EventDefinitionInstanceBuilder),
	}
	sender := node.Tracer.RegisterSender()
	go node.runner(ctx, sender)
	err = node.EventEgress.RegisterEventConsumer(node)
	if err != nil {
		return
	}
	return
}

func (node *Node) runner(ctx context.Context, sender tracing.SenderHandle) {
	defer sender.Done()

	for {
		select {
		case msg := <-node.runnerChannel:
			switch m := msg.(type) {
			case processEventMessage:
				if node.activated {
					node.Tracer.Trace(EventObservedTrace{Node: node.element, Event: m.event})
					if satisfied, _ := node.satisfier.Satisfy(m.event); satisfied {
						awaitingActions := node.awaitingActions
						for _, actionChan := range awaitingActions {
							actionChan <- flow_node.FlowAction{SequenceFlows: flow_node.AllSequenceFlows(&node.Outgoing)}
						}
						node.awaitingActions = make([]chan flow_node.Action, 0)
						node.activated = false
					}
				}
			case nextActionMessage:
				if !node.activated {
					node.activated = true
					node.Tracer.Trace(ActiveListeningTrace{Node: node.element})
				}
				node.awaitingActions = append(node.awaitingActions, m.response)
			default:
			}
		case <-ctx.Done():
			node.Tracer.Trace(flow_node.CancellationTrace{Node: node.element})
			return
		}
	}
}

func (node *Node) ConsumeEvent(ev event.Event) (result event.ConsumptionResult, err error) {
	node.runnerChannel <- processEventMessage{event: ev}
	result = event.Consumed
	return
}

func (node *Node) NextAction(flow_interface.T) chan flow_node.Action {
	response := make(chan flow_node.Action)
	node.runnerChannel <- nextActionMessage{response: response}
	return response
}

func (node *Node) Element() bpmn.FlowNodeInterface {
	return node.element
}
