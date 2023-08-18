package process

import (
	"context"
	"testing"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/flow_node"
	"github.com/oliveio/olive/engine/process/instance"
	"github.com/oliveio/olive/engine/tracing"
	"github.com/oliveio/olive/test"
	"github.com/stretchr/testify/assert"
)

var defaultDefinitions = bpmn.DefaultDefinitions()

var sampleDoc bpmn.Definitions

func init() {
	test.LoadTestFile("sample/process/sample.bpmn", &sampleDoc)
}

func TestExplicitInstantiation(t *testing.T) {
	if proc, found := sampleDoc.FindBy(bpmn.ExactId("sample")); found {
		process := New(proc.(*bpmn.Process), &defaultDefinitions)
		inst, err := process.Instantiate()
		assert.Nil(t, err)
		assert.NotNil(t, inst)
	} else {
		t.Fatalf("Can't find process `sample`")
	}
}

func TestCancellation(t *testing.T) {
	if proc, found := sampleDoc.FindBy(bpmn.ExactId("sample")); found {
		ctx, cancel := context.WithCancel(context.Background())

		process := New(proc.(*bpmn.Process), &defaultDefinitions, WithContext(ctx))

		tracer := tracing.NewTracer(ctx)
		traces := tracer.SubscribeChannel(make(chan tracing.Trace, 128))

		inst, err := process.Instantiate(instance.WithContext(ctx), instance.WithTracer(tracer))
		assert.Nil(t, err)
		assert.NotNil(t, inst)

		cancel()

		cancelledFlowNodes := make([]bpmn.FlowNodeInterface, 0)

		for trace := range traces {
			trace = tracing.Unwrap(trace)
			switch trace := trace.(type) {
			case flow_node.CancellationTrace:
				cancelledFlowNodes = append(cancelledFlowNodes, trace.Node)
			default:
			}
		}

		assert.NotEmpty(t, cancelledFlowNodes)
	} else {
		t.Fatalf("Can't find process `sample`")
	}
}
