package start_event

import (
	"context"
	"testing"

	"github.com/oliveio/olive/bpmn/flow"
	"github.com/oliveio/olive/bpmn/process"
	"github.com/oliveio/olive/bpmn/schema"
	"github.com/oliveio/olive/bpmn/tracing"
	"github.com/oliveio/olive/test"
	_ "github.com/stretchr/testify/assert"
)

var testDoc schema.Definitions

func init() {
	test.LoadTestFile("sample/start_event/start.bpmn", &testDoc)
}

func TestStartEvent(t *testing.T) {
	processElement := (*testDoc.Processes())[0]
	proc := process.New(&processElement, &testDoc)
	if instance, err := proc.Instantiate(); err == nil {
		traces := instance.Tracer.Subscribe()
		err := instance.StartAll(context.Background())
		if err != nil {
			t.Fatalf("failed to run the instance: %s", err)
		}
	loop:
		for {
			trace := tracing.Unwrap(<-traces)
			switch trace := trace.(type) {
			case flow.Trace:
				if id, present := trace.Source.Id(); present {
					if *id == "start" {
						// success!
						break loop
					}

				}
			case tracing.ErrorTrace:
				t.Fatalf("%#v", trace)
			default:
				t.Logf("%#v", trace)
			}
		}
		instance.Tracer.Unsubscribe(traces)
	} else {
		t.Fatalf("failed to instantiate the process: %s", err)
	}
}
