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
	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/data"
	"github.com/olive-io/bpmn/flow"
	bact "github.com/olive-io/bpmn/flow_node/activity"
	bp "github.com/olive-io/bpmn/process"
	bpi "github.com/olive-io/bpmn/process/instance"
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/tracing"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
)

var (
	definitionPrefix = []byte("definitions")
	processPrefix    = []byte("processes")
)

func (r *Region) DeployDefinition(ctx context.Context, req *pb.RegionDeployDefinitionRequest) (*pb.RegionDeployDefinitionResponse, error) {
	resp := &pb.RegionDeployDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, &pb.RaftInternalRequest{DeployDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionDeployDefinitionResponse)
	return resp, nil
}

func (r *Region) ExecuteDefinition(ctx context.Context, req *pb.RegionExecuteDefinitionRequest) (*pb.RegionExecuteDefinitionResponse, error) {
	resp := &pb.RegionExecuteDefinitionResponse{}
	result, err := r.raftRequestOnce(ctx, &pb.RaftInternalRequest{ExecuteDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionExecuteDefinitionResponse)
	return resp, nil
}

func (r *Region) GetProcessInstance(ctx context.Context, definitionId string, definitionVersion, id uint64) (*pb.ProcessInstance, error) {
	prefix := bytesutil.PathJoin(processPrefix,
		[]byte(definitionId),
		[]byte(fmt.Sprintf("%d", definitionVersion)))
	key := bytesutil.PathJoin(prefix, []byte(fmt.Sprintf("%d", id)))

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.cfg.ReqTimeout())
		defer cancel()
	}

	req := &pb.RegionRangeRequest{
		Key:          key,
		Limit:        1,
		Serializable: true,
	}
	rangePrefix(req)
	rsp, err := r.Range(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	instance := new(pb.ProcessInstance)
	_ = proto.Unmarshal(rsp.Kvs[0].Value, instance)

	return instance, nil
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
		_ = proto.Unmarshal(kv.Value, definition)
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
	dataObjects := process.DataObjects

	variables := make(map[string][]byte)
	for name, value := range process.Properties {
		box, _ := json.Marshal(value)
		variables[name] = box
	}
	if state := process.RunningState; state != nil {
		for key, value := range state.Properties {
			variables[key] = value
		}
		for key, value := range state.DataObjects {
			dataObjects[key] = value
		}
		for key, value := range state.Variables {
			variables[key] = value
		}
	}

	ctx := context.Background()
	options := []bpi.Option{
		bpi.WithLocator(locator),
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
			switch tt := trace.(type) {
			case flow.VisitTrace:
				id, _ := tt.Node.Id()

				switch tt.Node.(type) {
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
					if name, has := tt.Node.Name(); has {
						flowNode.Name = *name
					}
					process.FlowNodes[*id] = flowNode
				}

			case flow.LeaveTrace:
				id, _ := tt.Node.Id()

				flowNode, ok := process.FlowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().Unix()
				}

				switch tt.Node.(type) {
				case schema.EventInterface:
					r.metric.event.Dec()
				}

			case bact.ActiveBoundaryTrace:
				if tt.Start {
					r.metric.task.Inc()
				} else {
					r.metric.task.Dec()
					process.RunningState.DataObjects = toGenericMap[[]byte](inst.Locator.CloneItems(data.LocatorObject))
					process.RunningState.Properties = toGenericMap[[]byte](inst.Locator.CloneItems(data.LocatorProperty))
					process.RunningState.Variables = toGenericMap[[]byte](inst.Locator.CloneVariables())
				}
			case *bact.Trace:
				act := tt.GetActivity()
				id, _ := act.Element().Id()

				flowNode, ok := process.FlowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().Unix()
					headers, dataSets, dataObjects := bact.FetchTaskDataInput(inst.Locator, act.Element())
					flowNode.Headers = toGenericMap[string](headers)
					flowNode.Properties = toGenericMap[[]byte](dataSets)
					flowNode.DataObjects = toGenericMap[[]byte](dataObjects)
				}

				_, ok = completed[*id]
				if ok {
					tt.Do()
				} else {
					r.handleActivity(ctx, tt, inst, process)
				}
			case tracing.ErrorTrace:
				finish = true
				err = tt.Error
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

func (r *Region) handleActivity(ctx context.Context, trace *bact.Trace, inst *bpi.Instance, process *pb.ProcessInstance) {

	taskAct := trace.GetActivity()
	actType := taskAct.Type()
	dsyAct := &dsypb.Activity{
		Type: actCov(actType),
	}
	if id, ok := taskAct.Element().Id(); ok {
		dsyAct.Id = *id
	}
	if name, ok := taskAct.Element().Name(); ok {
		dsyAct.Name = *name
	}

	fields := []zap.Field{
		zap.String("definition", process.DefinitionId),
		zap.Uint64("version", process.DefinitionVersion),
		zap.Uint64("process", process.Id),
		zap.String("task", dsyAct.Id),
	}

	headers := toGenericMap[string](trace.GetHeaders())
	properties := trace.GetProperties()
	dataObjects := trace.GetDataObjects()

	req := &dsypb.TransmitRequest{
		Activity:    dsyAct,
		Headers:     headers,
		Properties:  map[string]*dsypb.Box{},
		DataObjects: map[string]*dsypb.Box{},
	}
	for key, value := range properties {
		req.Properties[key] = dsypb.BoxFromAny(value)
	}
	for key, value := range dataObjects {
		req.DataObjects[key] = dsypb.BoxFromAny(value)
	}

	doOpts := make([]bact.DoOption, 0)
	resp, err := r.proxy.Handle(ctx, req)
	if err != nil {
		r.lg.Error("handle task", append(fields, zap.Error(err))...)
		doOpts = append(doOpts, bact.WithErr(err))
	} else {
		dp := map[string]any{}
		ddo := map[string]any{}
		for key, value := range resp.Properties {
			dp[key] = value.Value()
		}
		for key, value := range resp.DataObjects {
			ddo[key] = value.Value()
		}
		doOpts = append(doOpts, bact.WithProperties(dp), bact.WithObjects(ddo))
	}
	trace.Do(doOpts...)
}

func saveProcess(ctx context.Context, kv IRegionRaftKV, process *pb.ProcessInstance) error {
	pkey := bytesutil.PathJoin(processPrefix,
		[]byte(process.DefinitionId), []byte(fmt.Sprintf("%d", process.DefinitionVersion)),
		[]byte(fmt.Sprintf("%d", process.Id)))

	value, err := proto.Marshal(process)
	if err != nil {
		return err
	}
	_, err = kv.Put(ctx, &pb.RegionPutRequest{Key: pkey, Value: value})
	return err
}

// actCov converts bact.Type to discoverypb.ActivityType
func actCov(actType bact.Type) dsypb.ActivityType {
	var act dsypb.ActivityType
	switch actType {
	case bact.TaskType:
		act = dsypb.ActivityType_Task
	case bact.ServiceType:
		act = dsypb.ActivityType_ServiceTask
	case bact.ScriptType:
		act = dsypb.ActivityType_ScriptTask
	case bact.UserType:
		act = dsypb.ActivityType_UserTask
	case bact.SendType:
		act = dsypb.ActivityType_SendTask
	case bact.ReceiveType:
		act = dsypb.ActivityType_ReceiveTask
	case bact.CallType:
		act = dsypb.ActivityType_CallActivity
	default:
		act = dsypb.ActivityType_Task
	}
	return act
}
