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
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/data"
	"github.com/olive-io/bpmn/flow"
	"github.com/olive-io/bpmn/flow_node/activity"
	bp "github.com/olive-io/bpmn/process"
	bpi "github.com/olive-io/bpmn/process/instance"
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/tracing"
	"github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/pkg/discovery/execute"
	"go.uber.org/zap"
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

	locator := data.NewFlowDataLocator()
	if len(*definitions.Processes()) == 0 {
		return
	}
	processElement := (*definitions.Processes())[0]
	properties := process.Properties
	dataObjects := process.DataObjects
	var variables map[string][]byte
	if state := process.RunningState; state != nil {
		for key, value := range state.Properties {
			properties[key] = value
		}
		for key, value := range state.DataObjects {
			dataObjects[key] = value
		}
		variables = state.Variables
	}

	ctx := context.Background()
	options := []bpi.Option{
		bpi.WithLocator(locator),
		bpi.WithVariables(toAnyMap[[]byte](properties)),
		bpi.WithDataObjects(toAnyMap[[]byte](dataObjects)),
		bpi.WithVariables(toAnyMap[[]byte](variables)),
	}
	r.scheduleProcess(ctx, process, definitions, &processElement, options...)
}

func (r *Region) scheduleProcess(ctx context.Context, process *pb.ProcessInstance, definitions *schema.Definitions, processElement *schema.Process, options ...bpi.Option) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r.metric.process.Inc()
	defer r.metric.process.Dec()

	fields := []zap.Field{
		zap.String("definition", process.DefinitionId),
		zap.Uint64("version", process.DefinitionVersion),
		zap.Uint64("process", process.Id),
	}

	r.lg.Info("start process instance", fields...)

	var err error
	if process.Status == pb.ProcessInstance_Prepare {
		process.Status = pb.ProcessInstance_Running
		process.StartTime = time.Now().Unix()
		err = saveProcess(ctx, r, process)
		if err != nil {
			r.lg.Error("save process instance", append(fields, zap.Error(err))...)
		}
	}

	finish := true
	defer func() {
		if !finish {
			return
		}

		process.EndTime = time.Now().Unix()
		process.Status = pb.ProcessInstance_Ok
		if err != nil {
			process.Status = pb.ProcessInstance_Fail
			process.Message = err.Error()
		}

		err = saveProcess(ctx, r, process)
		if err != nil {
			r.lg.Error("finish process instance", append(fields, zap.Error(err))...)
		} else {
			r.lg.Info("finish process instance", fields...)
		}
	}()

	proc := bp.New(processElement, definitions)
	var inst *bpi.Instance
	inst, err = proc.Instantiate(options...)
	if err != nil {
		r.lg.Error("failed to instantiate the process", append(fields, zap.Error(err))...)
		return
	}
	traces := inst.Tracer.Subscribe()
	err = inst.StartAll(ctx)
	if err != nil {
		cancel()
		r.lg.Error("failed to run the instance", append(fields, zap.Error(err))...)
		return
	}

	finish = false
	completed := map[string]struct{}{}

LOOP:
	for {
		select {
		case <-r.changeNotify():
			r.processQ.Push(process)
			break LOOP
		case trace := <-traces:
			trace = tracing.Unwrap(trace)
			switch trace := trace.(type) {
			case flow.VisitTrace:
				id, _ := trace.Node.Id()

				switch trace.Node.(type) {
				case schema.EndEventInterface:
				case schema.EventInterface:
					r.metric.event.Inc()
				}

				if process.FlowNodes == nil {
					process.FlowNodes = map[string]*pb.FlowNodeStat{}
				}
				flowNode, ok := process.FlowNodes[*id]
				if ok {
					completed[*id] = struct{}{}
				} else {
					flowNode = &pb.FlowNodeStat{
						Id:        *id,
						StartTime: time.Now().Unix(),
					}
					if name, has := trace.Node.Name(); has {
						flowNode.Name = *name
					}
					process.FlowNodes[*id] = flowNode
				}

			case flow.LeaveTrace:
				id, _ := trace.Node.Id()

				flowNode, ok := process.FlowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().Unix()
				}

				switch trace.Node.(type) {
				case schema.EventInterface:
					r.metric.event.Dec()
				}

			case activity.ActiveBoundaryTrace:
				if trace.Start {
					r.metric.task.Inc()
				} else {
					r.metric.task.Dec()
					process.RunningState.DataObjects = toTMap[[]byte](inst.Locator.CloneItems(data.LocatorObject))
					process.RunningState.Properties = toTMap[[]byte](inst.Locator.CloneItems(data.LocatorProperty))
					process.RunningState.Variables = toTMap[[]byte](inst.Locator.CloneVariables())
				}
			case *activity.Trace:
				act := trace.GetActivity()
				id, _ := act.Element().Id()

				flowNode, ok := process.FlowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().Unix()
					headers, dataSets, dataObjects := activity.FetchTaskDataInput(inst.Locator, act.Element())
					flowNode.Headers = toTMap[string](headers)
					flowNode.Properties = toTMap[[]byte](dataSets)
					flowNode.DataObjects = toTMap[[]byte](dataObjects)
				}

				_, ok = completed[*id]
				if ok {
					trace.Do()
				} else {
					r.handleActivity(ctx, trace, inst, process)
				}
			case tracing.ErrorTrace:
				finish = true
				err = trace.Error
				r.lg.Error("process error occurred", append(fields, zap.Error(err))...)
			case flow.CeaseFlowTrace:
				finish = true
				break LOOP
			default:
			}

			_ = saveProcess(ctx, r, process)
		}
	}
	inst.Tracer.Unsubscribe(traces)
}

func (r *Region) handleActivity(ctx context.Context, trace *activity.Trace, inst *bpi.Instance, process *pb.ProcessInstance) {

	taskAct := trace.GetActivity()
	id, _ := taskAct.Element().Id()
	fields := []zap.Field{
		zap.String("definition", process.DefinitionId),
		zap.Uint64("version", process.DefinitionVersion),
		zap.Uint64("process", process.Id),
		zap.String("task", *id),
	}

	actType := taskAct.Type()
	headers := toTMap[string](trace.GetHeaders())
	properties := trace.GetProperties()
	dataObjects := trace.GetDataObjects()

	var act discoverypb.Activity
	switch actType {
	case activity.TaskType:
		act = discoverypb.Activity_Task
	case activity.ServiceType:
		act = discoverypb.Activity_ServiceTask
	case activity.ScriptType:
		act = discoverypb.Activity_ScriptTask
	case activity.UserType:
		act = discoverypb.Activity_UserTask
	case activity.SendType:
		act = discoverypb.Activity_SendTask
	case activity.ReceiveType:
		act = discoverypb.Activity_ReceiveTask
	case activity.CallType:
		act = discoverypb.Activity_CallActivity
	}

	urlText := headers[execute.URLKey]
	if urlText == "" {
		urlText = execute.DefaultTaskURL
	}
	method := headers[execute.MethodKey]
	if method == "" {
		method = "POST"
	}
	protocol := headers[execute.ProtocolKey]

	hurl, err := url.Parse(urlText)
	if err != nil {
		r.lg.Error("parse task url", append(fields, zap.Error(err))...)
		trace.Do(activity.WithErr(err))
		return
	}

	in := &discoverypb.ExecuteRequest{
		Activity:    act,
		Headers:     headers,
		Properties:  map[string]*discoverypb.Box{},
		DataObjects: map[string]*discoverypb.Box{},
	}
	for key, value := range properties {
		in.Properties[key] = discoverypb.BoxFromT(value)
	}
	for key, value := range dataObjects {
		in.DataObjects[key] = discoverypb.BoxFromT(value)
	}

	buf, _ := json.Marshal(in)

	header := http.Header{}
	for key, value := range headers {
		if strings.HasPrefix(key, execute.HeaderKeyPrefix) {
			continue
		}
		header.Set(key, value)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	header.Set(execute.RequestActivityKey, act.String())

	hreq := &http.Request{
		Method: method,
		URL:    hurl,
		Proto:  protocol,
		Header: header,
		Body:   io.NopCloser(bytes.NewBuffer(buf)),
	}
	hreq = hreq.WithContext(ctx)

	doOpts := make([]activity.DoOption, 0)
	result, err := r.gw.Handle(hreq)
	if err != nil {
		r.lg.Error("handle task", append(fields, zap.Error(err))...)
		doOpts = append(doOpts, activity.WithErr(err))
	} else {
		resp := new(discoverypb.Response)
		if err = json.Unmarshal(result, resp); err != nil {
			r.lg.Error("unmarshal gateway response", append(fields, zap.Error(err))...)
		}
		dp := map[string]any{}
		ddo := map[string]any{}
		for key, value := range resp.Properties {
			dp[key] = value.Value()
		}
		for key, value := range resp.DataObjects {
			ddo[key] = value.Value()
		}
		doOpts = append(doOpts, activity.WithProperties(dp), activity.WithObjects(ddo))
	}
	trace.Do(doOpts...)
}
