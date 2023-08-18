package task

import (
	"context"
	"testing"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/flow"
	"github.com/oliveio/olive/engine/process"
	"github.com/oliveio/olive/engine/tracing"
	"github.com/oliveio/olive/test"
	_ "github.com/stretchr/testify/assert"
)

var testTask bpmn.Definitions

func init() {
	test.LoadTestFile("sample/task/task.bpmn", &testTask)
}

func TestTask(t *testing.T) {
	processElement := (*testTask.Processes())[0]
	proc := process.New(&processElement, &testTask)
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
					if *id == "task" {
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
