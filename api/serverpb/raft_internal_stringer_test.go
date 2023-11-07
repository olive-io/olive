package serverpb_test

import (
	"testing"

	"github.com/olive-io/olive/api/serverpb"
)

// TestInvalidGoTypeIntPanic tests conditions that caused
func TestInvalidGoTypeIntPanic(t *testing.T) {
	result := serverpb.NewLoggablePutRequest(&serverpb.PutRequest{}).String()
	if result != "" {
		t.Errorf("Got result: %s, expected empty string", result)
	}
}
