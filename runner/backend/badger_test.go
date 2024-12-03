package backend_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/olive-io/olive/runner/backend"
)

func newBackend(t *testing.T) (backend.IBackend, func()) {
	cfg := backend.NewConfig("tmp")
	b, err := backend.NewBackend(cfg)
	if err != nil {
		t.Fatalf("error creating backend: %v", err)
	}

	cancel := func() {
		b.Close()

		os.RemoveAll("tmp")
	}

	return b, cancel
}

func TestBackend(t *testing.T) {
	b, cancel := newBackend(t)
	defer cancel()

	if err := b.ForceSync(); err != nil {
		t.Fatalf("error syncing backend: %v", err)
	}
}

func TestBackendOp(t *testing.T) {
	b, cancel := newBackend(t)
	defer cancel()

	ctx := context.TODO()
	if err := b.Set(ctx, "foo", []byte("bar")); err != nil {
		t.Fatalf("error setting value: %v", err)
	}

	val, err := b.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("error getting value: %v", err)
	}
	assert.Equal(t, val.Kvs[0].Value, []byte("bar"))

	if err = b.Del(ctx, "foo"); err != nil {
		t.Fatalf("error deleting value: %v", err)
	}

	val, err = b.Get(ctx, "foo")
	assert.Nil(t, val)
}

type TestData struct {
	Field1 string `json:"field1"`
	Field2 int32  `json:"field2"`
}

func TestBackendMarshal(t *testing.T) {
	b, cancel := newBackend(t)
	defer cancel()

	td1 := &TestData{
		Field1: "foo",
		Field2: 1,
	}

	td2 := &TestData{
		Field1: "bar",
		Field2: 1,
	}

	ctx := context.Background()
	if err := b.Set(ctx, "foo1", td1); err != nil {
		t.Fatalf("error setting value: %v", err)
	}
	if err := b.Set(ctx, "foo2", td2); err != nil {
		t.Fatalf("error setting value: %v", err)
	}

	resp, err := b.Get(ctx, "foo1")
	if err != nil {
		t.Fatalf("error getting value: %v", err)
	}

	var td TestData
	if err = backend.NewUnmarshaler[TestData](resp).To(&td); err != nil {
		t.Fatalf("error unmarshaling value: %v", err)
	}
	assert.Equal(t, td.Field1, td1.Field1)

	resp, err = b.Get(ctx, "foo", backend.GetPrefix())
	if err != nil {
		t.Fatalf("error getting value: %v", err)
	}

	ts := make([]TestData, 0)
	if err = backend.NewUnmarshaler[TestData](resp).SliceTo(&ts); err != nil {
		t.Fatalf("error unmarshaling value: %v", err)
	}

	assert.Equal(t, len(ts), 2)
	assert.Equal(t, ts[0].Field1, td1.Field1)
	assert.Equal(t, ts[1].Field1, td2.Field1)
}
