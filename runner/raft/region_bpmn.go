/*
   Copyright 2023 The olive Authors

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

package raft

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
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
	"github.com/olive-io/olive/api/gatewaypb"
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

func (r *Region) GetProcessInstance(ctx context.Context, definitionId string, definitionVersion uint64, id string) (*pb.ProcessInstance, error) {
	prefix := bytesutil.PathJoin(processPrefix,
		[]byte(definitionId),
		[]byte(fmt.Sprintf("%d", definitionVersion)))
	key := bytesutil.PathJoin(prefix, []byte(id))

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
	if len(process.DefinitionsContent) == 0 {
		definitionKey := bytesutil.PathJoin(definitionPrefix,
			[]byte(process.DefinitionsId), []byte(fmt.Sprintf("%d", process.DefinitionsVersion)))

		kv, err := r.get(definitionKey)
		if err != nil {
			if errors.Is(err, pebble.ErrNotFound) {

			}
			return
		}
		definition := new(pb.Definition)
		_ = proto.Unmarshal(kv.Value, definition)
		process.DefinitionsContent = definition.Content
	}

	var definitions *schema.Definitions
	err := xml.Unmarshal([]byte(process.DefinitionsContent), &definitions)
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
	if id, ok := processElement.Id(); ok {
		process.DefinitionsProcess = *id
	}
	var dataObjects map[string][]byte
	for name, value := range process.DataObjects {
		dataObjects[name] = []byte(value)
	}

	variables := make(map[string][]byte)
	for name, value := range process.Properties {
		box, _ := json.Marshal(value)
		variables[name] = box
	}
	if state := process.RunningState; state != nil {
		for key, value := range state.Properties {
			variables[key] = []byte(value)
		}
		for key, value := range state.DataObjects {
			dataObjects[key] = []byte(value)
		}
		for key, value := range state.Variables {
			variables[key] = []byte(value)
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
		zap.String("definition", process.DefinitionsId),
		zap.Uint64("version", process.DefinitionsVersion),
		zap.String("process", process.Id),
	}

	r.lg.Info("start process instance", fields...)

	var err error
	if process.Status == pb.ProcessInstance_Prepare {
		process.Status = pb.ProcessInstance_Running
		process.StartTime = time.Now().UnixNano()
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

		process.EndTime = time.Now().UnixNano()
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
			r.lg.Sugar().Debugf("%#v", trace)
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
						StartTime: time.Now().UnixNano(),
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
					flowNode.EndTime = time.Now().UnixNano()
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
					process.RunningState.DataObjects = toGenericMap[string](inst.Locator.CloneItems(data.LocatorObject))
					process.RunningState.Properties = toGenericMap[string](inst.Locator.CloneItems(data.LocatorProperty))
					process.RunningState.Variables = toGenericMap[string](inst.Locator.CloneVariables())
				}
			case *bact.Trace:
				act := tt.GetActivity()
				id, _ := act.Element().Id()

				flowNode, ok := process.FlowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().UnixNano()
					headers, dataSets, dataObjects := bact.FetchTaskDataInput(inst.Locator, act.Element())
					flowNode.Headers = toGenericMap[string](headers)
					flowNode.Properties = toGenericMap[string](dataSets)
					flowNode.DataObjects = toGenericMap[string](dataObjects)
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
		Type:               actCov(actType),
		Definitions:        process.DefinitionsId,
		DefinitionsVersion: process.DefinitionsVersion,
	}
	if id, ok := taskAct.Element().Id(); ok {
		dsyAct.Id = *id
	}
	if name, ok := taskAct.Element().Name(); ok {
		dsyAct.Name = *name
	}
	if id, ok := inst.Process().Id(); ok {
		dsyAct.Process = *id
	}
	if extension, ok := taskAct.Element().ExtensionElements(); ok {
		if td := extension.TaskDefinitionField; td != nil {
			dsyAct.TaskType = td.Type
		}
	}

	fields := []zap.Field{
		zap.String("definition", process.DefinitionsId),
		zap.Uint64("version", process.DefinitionsVersion),
		zap.String("process", process.Id),
		zap.String("task", dsyAct.Id),
	}

	headers := toGenericMap[string](trace.GetHeaders())
	properties := trace.GetProperties()
	dataObjects := trace.GetDataObjects()

	req := &gatewaypb.TransmitRequest{
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
		err = fmt.Errorf("%s", err.Error())
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
		[]byte(process.DefinitionsId), []byte(fmt.Sprintf("%d", process.DefinitionsVersion)),
		[]byte(process.Id))

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
