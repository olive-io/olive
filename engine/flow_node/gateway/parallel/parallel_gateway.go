package parallel

import (
	"context"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/flow_node"
	"github.com/oliveio/olive/engine/flow_node/gateway"
	"github.com/oliveio/olive/engine/tracing"
)

type message interface {
	message()
}

type nextActionMessage struct {
	response chan flow_node.Action
	flow     flow_interface.T
}

func (m nextActionMessage) message() {}

type Node struct {
	*flow_node.Wiring
	element               *bpmn.ParallelGateway
	runnerChannel         chan message
	reportedIncomingFlows int
	awaitingActions       []chan flow_node.Action
	noOfIncomingFlows     int
}

func New(ctx context.Context, wiring *flow_node.Wiring, parallelGateway *bpmn.ParallelGateway) (node *Node, err error) {
	node = &Node{
		Wiring:                wiring,
		element:               parallelGateway,
		runnerChannel:         make(chan message, len(wiring.Incoming)*2+1),
		reportedIncomingFlows: 0,
		awaitingActions:       make([]chan flow_node.Action, 0),
		noOfIncomingFlows:     len(wiring.Incoming),
	}
	sender := node.Tracer.RegisterSender()
	go node.runner(ctx, sender)
	return
}

func (node *Node) flowWhenReady() {
	if node.reportedIncomingFlows == node.noOfIncomingFlows {
		node.reportedIncomingFlows = 0
		awaitingActions := node.awaitingActions
		node.awaitingActions = make([]chan flow_node.Action, 0)
		sequenceFlows := flow_node.AllSequenceFlows(&node.Outgoing)
		gateway.DistributeFlows(awaitingActions, sequenceFlows)
	}

}

func (node *Node) runner(ctx context.Context, sender tracing.SenderHandle) {
	defer sender.Done()

	for {
		select {
		case msg := <-node.runnerChannel:
			switch m := msg.(type) {
			case nextActionMessage:
				node.reportedIncomingFlows++
				node.awaitingActions = append(node.awaitingActions, m.response)
				node.flowWhenReady()
				node.Tracer.Trace(IncomingFlowProcessedTrace{Node: node.element, Flow: m.flow})
			default:
			}
		case <-ctx.Done():
			node.Tracer.Trace(flow_node.CancellationTrace{Node: node.element})
			return
		}
	}
}

func (node *Node) NextAction(flow flow_interface.T) chan flow_node.Action {
	response := make(chan flow_node.Action)
	node.runnerChannel <- nextActionMessage{response: response, flow: flow}
	return response
}

func (node *Node) Element() bpmn.FlowNodeInterface {
	return node.element
}
