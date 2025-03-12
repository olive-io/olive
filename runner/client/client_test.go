/*
Copyright 2025 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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

func TestListProcess(t *testing.T) {
	cc := newClient(t)
	ctx := context.Background()

	definition := int64(1)
	id := uint64(0)
	r, err := cc.ListProcess(ctx, definition, id)
	if err != nil {
		t.Fatalf("could not list process instances: %v", err)
	}

	t.Logf("%#v", r[0])
}

func TestRunProcess(t *testing.T) {
	cc := newClient(t)

	ctx := context.Background()
	content := getFile("testdata1/task.bpmn")
	process, err := cc.BuildProcess().
		SetDefinition(1, 5).
		SetBpmn(string(content)).
		SetName("test process").
		Do(ctx)

	if err != nil {
		t.Fatalf("could not build process instance: %v", err)
	}

	t.Logf("process: %#v", process)
}
