package rpctypes

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConvert(t *testing.T) {
	e1 := status.New(codes.InvalidArgument, "oliveserver: key is not provided").Err()
	e2 := ErrGRPCEmptyKey
	e3 := ErrEmptyKey

	if e1.Error() != e2.Error() {
		t.Fatalf("expected %q == %q", e1.Error(), e2.Error())
	}
	if ev1, ok := status.FromError(e1); ok && ev1.Code() != e3.(OliveError).Code() {
		t.Fatalf("expected them to be equal, got %v / %v", ev1.Code(), e3.(OliveError).Code())
	}

	if e1.Error() == e3.Error() {
		t.Fatalf("expected %q != %q", e1.Error(), e3.Error())
	}
	if ev2, ok := status.FromError(e2); ok && ev2.Code() != e3.(OliveError).Code() {
		t.Fatalf("expected them to be equal, got %v / %v", ev2.Code(), e3.(OliveError).Code())
	}
}
