package flow_node

import (
	"context"
	"sync"
	"testing"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/event"
	"github.com/oliveio/olive/engine/tracing"
	"github.com/oliveio/olive/test"
	"github.com/stretchr/testify/assert"
)

var defaultDefinitions = bpmn.DefaultDefinitions()

var sampleDoc bpmn.Definitions

func init() {
	test.LoadTestFile("sample/flow_node/sample.bpmn", &sampleDoc)
}

func TestNewWiring(t *testing.T) {
	var waitGroup sync.WaitGroup
	if proc, found := sampleDoc.FindBy(bpmn.ExactId("sample")); found {
		if flowNode, found := sampleDoc.FindBy(bpmn.ExactId("either")); found {
			node, err := NewWiring(
				nil,
				proc.(*bpmn.Process),
				&defaultDefinitions,
				&flowNode.(*bpmn.ParallelGateway).FlowNode,
				event.VoidConsumer{},
				event.VoidSource{},
				tracing.NewTracer(context.Background()), NewLockedFlowNodeMapping(),
				&waitGroup,
				event.WrappingDefinitionInstanceBuilder,
			)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(node.Incoming))
			t.Logf("%+v", node.Incoming[0])
			if incomingSeqFlowId, present := node.Incoming[0].Id(); present {
				assert.Equal(t, *incomingSeqFlowId, "x1")
			} else {
				t.Fatalf("Sequence flow x1 has no matching ID")
			}
			assert.Equal(t, 2, len(node.Outgoing))
			if outgoingSeqFlowId, present := node.Outgoing[0].Id(); present {
				assert.Equal(t, *outgoingSeqFlowId, "x2")
			} else {
				t.Fatalf("Sequence flow x2 has no matching ID")
			}
			if outgoingSeqFlowId, present := node.Outgoing[1].Id(); present {
				assert.Equal(t, *outgoingSeqFlowId, "x3")
			} else {
				t.Fatalf("Sequence flow x3 has no matching ID")
			}
		} else {
			t.Fatalf("Can't find flow node `either`")
		}
	} else {
		t.Fatalf("Can't find process `sample`")
	}
}
