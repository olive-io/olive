// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/olive-io/bpmn/flow"
	"github.com/olive-io/bpmn/flow_node/activity"
	bp "github.com/olive-io/bpmn/process"
	"github.com/olive-io/bpmn/process/instance"
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/tracing"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
)

var (
	definitionPrefix = []byte("definitions")
	processPrefix    = []byte("processes")
)

func (r *Region) DeployDefinition(ctx context.Context, req *pb.RegionDeployDefinitionRequest) (*pb.RegionDeployDefinitionResponse, error) {
	resp := &pb.RegionDeployDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{DeployDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionDeployDefinitionResponse)
	return resp, nil
}

func (r *Region) ExecuteDefinition(ctx context.Context, req *pb.RegionExecuteDefinitionRequest) (*pb.RegionExecuteDefinitionResponse, error) {
	resp := &pb.RegionExecuteDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{ExecuteDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionExecuteDefinitionResponse)
	return resp, nil
}

func (r *Region) scheduleCycle() {
	duration := 100 * time.Millisecond
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		if !r.waitUtilLeader() {
			select {
			case <-r.stopc:
				return
			default:
			}
			continue
		}

		timer.Reset(duration)

	LOOP:
		for {
			select {
			case <-r.stopc:
				return
			case <-r.changeC:
				break LOOP
			case <-timer.C:
				timer.Reset(duration)

				x, ok := r.processQ.Pop()
				if !ok {
					break
				}

				processInstance := x.(*pb.ProcessInstance)
				go r.scheduleDefinition(processInstance)
			}
		}
	}
}

func (r *Region) scheduleDefinition(process *pb.ProcessInstance) {
	if len(process.DefinitionContent) == 0 {
		definitionKey := bytesutil.PathJoin(definitionPrefix,
			[]byte(process.DefinitionId), []byte(fmt.Sprintf("%d", process.DefinitionVersion)))

		kv, err := r.get(definitionKey)
		if err != nil {
			if errors.Is(err, pebble.ErrNotFound) {

			}
			return
		}
		definition := new(pb.Definition)
		_ = definition.Unmarshal(kv.Value)
		process.DefinitionContent = definition.Content
	}

	var definitions *schema.Definitions
	err := xml.Unmarshal(process.DefinitionContent, &definitions)
	if err != nil {
		return
	}
	r.metric.runningDefinition.Inc()
	defer r.metric.runningDefinition.Dec()

	logger := r.lg.Sugar()

	variables := make(map[string]any)
	for key, value := range process.Properties {
		variables[key] = value
	}

	options := []instance.Option{
		instance.WithVariables(variables),
		instance.WithDataObjects(make(map[string]any)),
	}
	for _, processElement := range *definitions.Processes() {
		proc := bp.New(&processElement, definitions)
		if instance, err := proc.Instantiate(options...); err == nil {
			traces := instance.Tracer.Subscribe()
			ctx, cancel := context.WithCancel(context.Background())
			err = instance.StartAll(ctx)
			if err != nil {
				cancel()
				logger.Errorf("failed to run the instance: %s", err)
			}
			go func() {
			LOOP:
				for {
					trace := tracing.Unwrap(<-traces)
					switch trace := trace.(type) {
					case activity.ActiveTaskTrace:

						//switch tt := trace.(type) {
						//case *service.ActiveTrace:
						//	//ctx := tt.Context
						//	//headers := tt.Headers
						//	//properties := tt.Properties
						//	//dataObjects := tt.DataObjects
						//}

						logger.Infof("%v", trace)
						trace.Execute()
					case tracing.ErrorTrace:
						logger.Errorf("%v", trace)
					case flow.CeaseFlowTrace:
						break LOOP
					default:
						logger.Infof("%v", trace)
					}
				}
			}()
			instance.WaitUntilComplete(ctx)
			instance.Tracer.Unsubscribe(traces)
			cancel()
		} else {
			logger.Errorf("failed to instantiate the process: %s", err)
		}
	}
}
