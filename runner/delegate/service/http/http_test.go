package http

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/olive-io/olive/runner/delegate"
)

func TestDelegateForHttp(t *testing.T) {
	http := &DelegateForHttp{}

	headers := map[string]string{
		"Content-Type": "application/json",
		"ov:method":    "POST",
		"ov:url":       "http://localhost:5050/",
	}

	properties := map[string]any{
		"a": 1,
		"b": "hello",
		"c": map[string]interface{}{
			"name": "lack",
		},
	}

	req := &delegate.Request{
		Headers:     headers,
		Properties:  properties,
		DataObjects: map[string]any{},
		Timeout:     time.Second * 30,
	}

	ctx := context.TODO()
	rsp, err := http.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	log.Fatalf("result: %v", rsp.Result)
}
