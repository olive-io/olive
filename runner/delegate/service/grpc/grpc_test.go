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
		"ov:host": "localhost:50051",
		"ov:name": "helloworld.Greeter.SayHello",
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
	rsp, err := grpc.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	log.Fatalf("result: %v", rsp.Result)
}
