package inclusive

import (
	"context"
	"fmt"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/errors"
	"github.com/oliveio/olive/engine/flow/flow_interface"
	"github.com/oliveio/olive/engine/flow_node"
	"github.com/oliveio/olive/engine/flow_node/gateway"
	"github.com/oliveio/olive/engine/id"
	"github.com/oliveio/olive/engine/sequence_flow"
	"github.com/oliveio/olive/engine/tracing"
)

type NoEffectiveSequenceFlows struct {
	*bpmn.InclusiveGateway
}

func (e NoEffectiveSequenceFlows) Error() string {
	ownId := "<unnamed>"
	if ownIdPtr, present := e.InclusiveGateway.Id(); present {
		ownId = *ownIdPtr
	}
	return fmt.Sprintf("No effective sequence flows found in exclusive gateway `%v`", ownId)
}

type message interface {
	message()
}

type nextActionMessage struct {
	response chan flow_node.Action
	flow     flow_interface.T
}

func (m nextActionMessage) message() {}

type probingReport struct {
	result []int
	flowId id.Id
}

func (m probingReport) message() {}

type flowSync struct {
	response chan flow_node.Action
	flow     flow_interface.T
}

type Node struct {
	*flow_node.Wiring
	element                 *bpmn.InclusiveGateway
	runnerChannel           chan message
	defaultSequenceFlow     *sequence_flow.SequenceFlow
	nonDefaultSequenceFlows []*sequence_flow.SequenceFlow
	probing                 *chan flow_node.Action
	activated               *flowSync
	awaiting                []id.Id
	arrived                 []id.Id
	sync                    []chan flow_node.Action
	*flowTracker
	synchronized bool
}

func New(ctx context.Context, wiring *flow_node.Wiring, inclusiveGateway *bpmn.InclusiveGateway) (node *Node, err error) {
	var defaultSequenceFlow *sequence_flow.SequenceFlow

	if seqFlow, present := inclusiveGateway.Default(); present {
		if node, found := wiring.Process.FindBy(bpmn.ExactId(*seqFlow).
			And(bpmn.ElementType((*bpmn.SequenceFlow)(nil)))); found {
			defaultSequenceFlow = new(sequence_flow.SequenceFlow)
			*defaultSequenceFlow = sequence_flow.Make(
				node.(*bpmn.SequenceFlow),
				wiring.Definitions,
			)
		} else {
			err = errors.NotFoundError{
				Expected: fmt.Sprintf("default sequence flow with ID %s", *seqFlow),
			}
			return nil, err
		}
	}

	nonDefaultSequenceFlows := flow_node.AllSequenceFlows(&wiring.Outgoing,
		func(sequenceFlow *sequence_flow.SequenceFlow) bool {
			if defaultSequenceFlow == nil {
				return false
			}
			return *sequenceFlow == *defaultSequenceFlow
		},
	)

	node = &Node{
		Wiring:                  wiring,
		element:                 inclusiveGateway,
		runnerChannel:           make(chan message, len(wiring.Incoming)*2+1),
		nonDefaultSequenceFlows: nonDefaultSequenceFlows,
		defaultSequenceFlow:     defaultSequenceFlow,
		flowTracker:             newFlowTracker(ctx, wiring.Tracer, inclusiveGateway),
	}
	sender := node.Tracer.RegisterSender()
	go node.runner(ctx, sender)
	return
}

func (node *Node) runner(ctx context.Context, sender tracing.SenderHandle) {
	defer node.flowTracker.shutdown()
	activity := node.flowTracker.activity()

	defer sender.Done()

	for {
		select {
		case msg := <-node.runnerChannel:
			switch m := msg.(type) {
			case probingReport:
				response := node.probing
				if response == nil {
					// Reschedule, there's no next action yet
					go func() {
						node.runnerChannel <- m
					}()
					continue
				}
				node.probing = nil
				flow := make([]*sequence_flow.SequenceFlow, 0)
				for _, i := range m.result {
					flow = append(flow, node.nonDefaultSequenceFlows[i])
				}

				switch len(flow) {
				case 0:
					// no successful non-default sequence flows
					if node.defaultSequenceFlow == nil {
						// exception (Table 13.2)
						node.Wiring.Tracer.Trace(tracing.ErrorTrace{
							Error: NoEffectiveSequenceFlows{
								InclusiveGateway: node.element,
							},
						})
					} else {
						gateway.DistributeFlows(node.sync, []*sequence_flow.SequenceFlow{node.defaultSequenceFlow})
					}
				default:
					gateway.DistributeFlows(node.sync, flow)
				}
				node.synchronized = false
				node.activated = nil
			case nextActionMessage:
				if node.synchronized {
					if m.flow.Id() == node.activated.flow.Id() {
						// Activating flow returned
						node.sync = append(node.sync, m.response)
						node.probing = &m.response
						// and now we wait until the probe has returned
					}
				} else {
					if node.activated == nil {
						// Haven't been activated yet
						node.activated = &flowSync{response: m.response, flow: m.flow}
						node.awaiting = node.flowTracker.activeFlowsInCohort(m.flow.Id())
						node.arrived = []id.Id{m.flow.Id()}
						node.sync = make([]chan flow_node.Action, 0)
					} else {
						// Already activated
						node.arrived = append(node.arrived, m.flow.Id())
						node.sync = append(node.sync, m.response)
					}
					node.trySync()
				}

			default:
			}
		case <-activity:
			if !node.synchronized && node.activated != nil {
				node.awaiting = node.flowTracker.activeFlowsInCohort(node.activated.flow.Id())
				node.trySync()
			}
		case <-ctx.Done():
			node.Tracer.Trace(flow_node.CancellationTrace{Node: node.element})
			return
		}
	}
}

func (node *Node) trySync() {
	if !node.synchronized && len(node.arrived) >= len(node.awaiting) {
		// Have we got everybody?
		matches := 0
		for i := range node.arrived {
			for j := range node.awaiting {
				if node.awaiting[j] == node.arrived[i] {
					matches++
				}
			}
		}
		if matches == len(node.awaiting) {
			anId := node.activated.flow.Id()
			// Probe outgoing sequence flow using the first flow
			node.activated.response <- flow_node.ProbeAction{
				SequenceFlows: node.nonDefaultSequenceFlows,
				ProbeReport: func(indices []int) {
					node.runnerChannel <- probingReport{
						result: indices,
						flowId: anId,
					}
				},
			}

			node.synchronized = true
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
