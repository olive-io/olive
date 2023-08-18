package model

import (
	"context"
	"testing"
	"time"

	"github.com/oliveio/olive/bpmn"
	"github.com/oliveio/olive/engine/clock"
	"github.com/oliveio/olive/engine/flow"
	"github.com/oliveio/olive/engine/model"
	"github.com/oliveio/olive/engine/tracing"
	"github.com/oliveio/olive/test"
	"github.com/stretchr/testify/require"
)

var testTimerStartEventInstantiation bpmn.Definitions

func init() {
	test.LoadTestFile("sample/model/instantiate_timer_start_event.bpmn", &testTimerStartEventInstantiation)
}

func TestTimerStartEventInstantiation(t *testing.T) {
	c := clock.NewMock()
	ctx := clock.ToContext(context.Background(), c)
	tracer := tracing.NewTracer(ctx)
	traces := tracer.SubscribeChannel(make(chan tracing.Trace, 128))
	m := model.New(&testTimerStartEventInstantiation, model.WithContext(ctx), model.WithTracer(tracer))
	err := m.Run(ctx)
	require.Nil(t, err)
loop:
	for {
		select {
		case trace := <-traces:
			_, ok := trace.(flow.Trace)
			// Should not flow
			require.False(t, ok)
		default:
			break loop
		}
	}
	// Advance clock by 1M
	c.Add(1 * time.Minute)
loop1:
	for {
		trace := tracing.Unwrap(<-traces)
		switch trace := trace.(type) {
		case flow.VisitTrace:
			if idPtr, present := trace.Node.Id(); present {
				if *idPtr == "end" {
					// we've reached the desired outcome
					break loop1
				}
			}
		default:
			t.Logf("%#v", trace)
		}
	}
}

var testRecurringTimerStartEventInstantiation bpmn.Definitions

func init() {
	test.LoadTestFile("sample/model/instantiate_recurring_timer_start_event.bpmn", &testRecurringTimerStartEventInstantiation)
}

func TestRecurringTimerStartEventInstantiation(t *testing.T) {
	c := clock.NewMock()
	ctx := clock.ToContext(context.Background(), c)
	tracer := tracing.NewTracer(ctx)
	traces := tracer.SubscribeChannel(make(chan tracing.Trace, 128))
	m := model.New(&testRecurringTimerStartEventInstantiation, model.WithContext(ctx), model.WithTracer(tracer))
	err := m.Run(ctx)
	require.Nil(t, err)
loop:
	for {
		select {
		case trace := <-traces:
			trace = tracing.Unwrap(trace)
			_, ok := trace.(flow.Trace)
			// Should not flow
			require.False(t, ok)
		default:
			break loop
		}
	}
	// Test for some arbitrary number of recurrences (say, 10?)
	for i := 0; i < 10; i++ {
		// Advance clock by 1M
		c.Add(1 * time.Minute)
	loop1:
		for {
			trace := tracing.Unwrap(<-traces)
			switch trace := trace.(type) {
			case flow.VisitTrace:
				if idPtr, present := trace.Node.Id(); present {
					if *idPtr == "end" {
						// we've reached the desired outcome
						break loop1
					}
				}
			default:
				t.Logf("%#v", trace)
			}
		}
	}
}
