/*
Copyright 2024 The olive Authors

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

package scheduler_test

import (
	"context"
	"embed"
	"os"
	"testing"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api"
	corev1 "github.com/olive-io/olive/api/types/core/v1"
	"github.com/olive-io/olive/runner/scheduler"
	"github.com/olive-io/olive/runner/storage"
	"github.com/olive-io/olive/runner/storage/backend"
)

//go:embed testdata1/task.bpmn
var fs embed.FS

func getBpmnFile(name string) ([]byte, error) {
	return fs.ReadFile(name)
}

func newScheduler(t *testing.T) (*scheduler.Scheduler, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	logger := zap.NewExample()

	dir := "./testdata"

	db, err := backend.NewBackend(backend.NewConfig(dir))
	if err != nil {
		t.Fatal(err)
	}

	scheme := api.NewScheme()
	bs := storage.New(scheme, db)

	cfg := scheduler.NewConfig(ctx, logger, bs)
	sch, err := scheduler.NewScheduler(cfg)
	if err != nil {
		t.Fatal(err)
	}

	destroy := func() {
		_ = os.RemoveAll(dir)
		cancel()
	}

	return sch, destroy
}

func TestNewScheduler(t *testing.T) {
	newScheduler(t)
}

func TestProcess_Run(t *testing.T) {
	sch, cancel := newScheduler(t)
	defer cancel()

	bpmn, err := getBpmnFile("testdata1/task.bpmn")
	if err != nil {
		t.Fatalf("could not get BPMN file: %v", err)
	}

	ctx := context.TODO()
	headers := map[string]string{}
	properties := map[string][]byte{}
	dataObjects := map[string][]byte{}
	instance, err := sch.RunProcess(ctx, "test", 1, string(bpmn), "", "test process", headers, properties, dataObjects)
	if err != nil {
		t.Fatalf("could not run instance: %v", err)
	}

	assert.Equal(t, instance.DefinitionsId, "test")
	assert.Equal(t, instance.DefinitionsVersion, uint64(1))
	assert.Equal(t, instance.Name, "test process")

	d, err := sch.GetDefinition(ctx, instance.DefinitionsId, instance.DefinitionsVersion)
	assert.Nil(t, err)
	assert.Equal(t, d.Content, bpmn)

	if err = sch.Start(); err != nil {
		t.Fatalf("could not start instance: %v", err)
	}

	time.Sleep(1 * time.Second)
	_ = sch.Stop()

	pi, err := sch.GetProcess(ctx, instance.DefinitionsId, instance.DefinitionsVersion, instance.UID)
	assert.Nil(t, err)

	assert.Equal(t, pi.Status, corev1.ProcessOk)
	assert.True(t, len(pi.FlowNodes) > 0)

	data, _ := json.Marshal(pi.FlowNodes)
	t.Logf("%v", string(data))

	data, _ = json.Marshal(pi.FlowNodeStatMap)
	t.Logf("%v", string(data))
}
