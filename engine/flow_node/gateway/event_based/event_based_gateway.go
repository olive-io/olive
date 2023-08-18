package event_based

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/errors"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/flow_node"
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
	element       *bpmn.EventBasedGateway
	runnerChannel chan message
	activated     bool
}

func New(ctx context.Context, wiring *flow_node.Wiring, eventBasedGateway *bpmn.EventBasedGateway) (node *Node, err error) {
	node = &Node{
		Wiring:        wiring,
		element:       eventBasedGateway,
		runnerChannel: make(chan message, len(wiring.Incoming)*2+1),
		activated:     false,
	}
	sender := node.Tracer.RegisterSender()
	go node.runner(ctx, sender)
	return
}

func (node *Node) runner(ctx context.Context, sender tracing.SenderHandle) {
	defer sender.Done()

	for {
		select {
		case msg := <-node.runnerChannel:
			switch m := msg.(type) {
			case nextActionMessage:
				var first int32 = 0
				sequenceFlows := flow_node.AllSequenceFlows(&node.Outgoing)
				terminationChannels := make(map[bpmn.IdRef]chan bool)
				for _, sequenceFlow := range sequenceFlows {
					if idPtr, present := sequenceFlow.Id(); present {
						terminationChannels[*idPtr] = make(chan bool)
					} else {
						node.Tracer.Trace(tracing.ErrorTrace{Error: errors.NotFoundError{
							Expected: fmt.Sprintf("id for %#v", sequenceFlow),
						}})
					}
				}

				action := flow_node.FlowAction{
					Terminate: func(sequenceFlowId *bpmn.IdRef) chan bool {
						return terminationChannels[*sequenceFlowId]
					},
					SequenceFlows: sequenceFlows,
					ActionTransformer: func(sequenceFlowId *bpmn.IdRef, action flow_node.Action) flow_node.Action {
						// only first one is to flow
						if atomic.CompareAndSwapInt32(&first, 0, 1) {
							node.Tracer.Trace(DeterminationMadeTrace{Element: node.element})
							for terminationCandidateId, ch := range terminationChannels {
								if sequenceFlowId != nil && terminationCandidateId != *sequenceFlowId {
									ch <- true
								}
								close(ch)
							}
							terminationChannels = make(map[bpmn.IdRef]chan bool)
							return action
						} else {
							return flow_node.CompleteAction{}
						}
					},
				}

				m.response <- action
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
