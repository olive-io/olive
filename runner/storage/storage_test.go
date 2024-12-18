package storage_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/olive-io/olive/api"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
	"github.com/olive-io/olive/runner/storage"
	"github.com/olive-io/olive/runner/storage/backend"
)

func newStorage(t *testing.T) (*storage.Storage, func()) {
	cfg := backend.NewConfig("tmp")
	b, err := backend.NewBackend(cfg)
	if err != nil {
		t.Fatalf("error creating backend: %v", err)
	}

	cancel := func() {
		b.Close()

		os.RemoveAll("tmp")
	}

	scheme := api.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	bs := storage.New(scheme, b)

	return bs, cancel
}

func TestStorage(t *testing.T) {
	bs, cancel := newStorage(t)
	defer cancel()

	rt1 := &corev1.Runner{}
	rt1.SetName("r1")
	rt1.SetUID(uuid.New().String())
	key1 := "/runner/r1"

	rt2 := &corev1.Runner{}
	rt2.SetName("r2")
	rt2.SetUID(uuid.New().String())
	key2 := "/runner/r2"

	ctx := context.Background()

	if err := bs.Create(ctx, key1, rt1, 0); err != nil {
		t.Fatal(err)
	}
	if err := bs.Create(ctx, key2, rt2, 0); err != nil {
		t.Fatal(err)
	}

	outs := new(corev1.RunnerList)
	if err := bs.GetList(ctx, "/runner", outs); err != nil {
		t.Fatal(err)
	}

	t.Logf("%#v", outs)
}
