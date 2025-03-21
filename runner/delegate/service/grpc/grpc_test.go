package grpc

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/olive-io/olive/runner/delegate"
)

func TestDelegateForGRPC_Call(t *testing.T) {
	grpc := &DelegateForGRPC{}

	headers := map[string]string{
		"ov:host": "localhost:9000",
		"ov:name": "helloworld.v1.UserService/ListUsers",
	}

	properties := map[string]any{
		"name": "gRPCurl",
	}

	req := &delegate.Request{
		Headers:     headers,
		Properties:  properties,
		DataObjects: map[string]any{},
		Timeout:     time.Second * 5,
	}

	ctx := context.TODO()
	resp, err := grpc.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	log.Fatalf("result: %v", resp.Result)
}
