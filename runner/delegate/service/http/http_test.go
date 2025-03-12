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
		"ov:method":    "GET",
		"ov:url":       "http://localhost:8000/auth/users",
	}

	properties := map[string]any{}

	req := &delegate.Request{
		Headers:     headers,
		Properties:  properties,
		DataObjects: map[string]any{},
		Timeout:     time.Second * 30,
	}

	ctx := context.TODO()
	resp, err := http.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("result: %v\n", resp.Result)
}
