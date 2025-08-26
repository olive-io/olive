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

package scheduler

import (
	"context"
	"embed"
	"encoding/xml"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

//go:embed testdata
var embedFS embed.FS

//go:embed testdata
var testdata embed.FS

func LoadTestFile(filename string, definitions any) {
	var err error
	src, err := testdata.ReadFile(filename)
	if err != nil {
		log.Fatalf("Can't read file %s: %v", filename, err)
	}
	err = xml.Unmarshal(src, definitions)
	if err != nil {
		log.Fatalf("XML unmarshalling error in %s: %v", filename, err)
	}
}

func TestScheduler(t *testing.T) {
	lg := zap.NewExample()
	options := NewOptions(lg)
	ctx := context.Background()
	sche, err := NewScheduler(ctx, options)
	if !assert.NoError(t, err) {
		return
	}
	wch := sche.Watch(ctx, "1")

	content, err := testdata.ReadFile("testdata/task.bpmn")
	if !assert.NoError(t, err) {
		return
	}

	process := &types.Process{
		Name:     "test",
		Uid:      "pa",
		Metadata: map[string]string{},
		Priority: 1,
		Args: &types.BpmnArgs{
			Headers:     map[string]string{},
			Properties:  map[string]string{},
			DataObjects: map[string]string{},
		},
		DefinitionsId:      1,
		DefinitionsVersion: 1,
		DefinitionsProcess: "test",
		Context: &types.ProcessContext{
			Variables:   map[string]string{},
			DataObjects: map[string]string{},
		},
		Attempts: 0,
		Status:   types.Process_Unknown,
	}

	stat := &ProcessStat{
		Process:     process,
		Definitions: string(content),
		FlowNodes:   []*types.FlowNode{},
	}

	err = sche.Execute(stat)
	if !assert.NoError(t, err) {
		return
	}

	for {
		rsp := wch.Next()
		if rsp.Err != nil {
			t.Fatalf("rsp err: %v", rsp.Err)
		}

		if rsp.Process != nil {
			t.Logf("process: %#v", rsp.Process)
			if rsp.Process.Stage == types.Process_Finish {
				break
			}
		}
		if rsp.FlowNode != nil {
			t.Logf("flow node: %#v", rsp.FlowNode)
		}
	}
}
