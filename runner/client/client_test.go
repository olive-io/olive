package client_test

import (
	"context"
	"embed"
	"testing"

	"github.com/olive-io/olive/runner/client"
)

//go:embed testdata1/task.bpmn
var fs embed.FS

func getFile(name string) []byte {
	data, _ := fs.ReadFile(name)
	return data
}

func newClient(t *testing.T) *client.Client {
	cfg := &client.Config{
		Address: "127.0.0.1:15280",
	}

	cc, err := client.NewClient(cfg)
	if err != nil {
		t.Fatalf("could not create client: %v", err)
	}
	return cc
}

func TestNewClient(t *testing.T) {
	newClient(t)
}

func TestGetRunner(t *testing.T) {
	cc := newClient(t)
	ctx := context.Background()
	r, err := cc.GetRunner(ctx)
	if err != nil {
		t.Fatalf("could not get runner: %v", err)
	}

	t.Log(r)
}

func TestListProcessInstances(t *testing.T) {
	cc := newClient(t)
	ctx := context.Background()

	definition := ""
	id := int64(0)
	r, err := cc.ListProcessInstances(ctx, definition, id)
	if err != nil {
		t.Fatalf("could not list process instances: %v", err)
	}

	t.Logf("%#v", r[0])

}

func TestRunProcessInstance(t *testing.T) {
	cc := newClient(t)

	ctx := context.Background()
	content := getFile("testdata1/task.bpmn")
	instance, err := cc.BuildProcessInstance().
		SetDefinition("test", 1).
		SetBpmn(string(content)).
		SetName("test process").
		Do(ctx)

	if err != nil {
		t.Fatalf("could not build process instance: %v", err)
	}

	t.Logf("instance: %#v", instance)
}
