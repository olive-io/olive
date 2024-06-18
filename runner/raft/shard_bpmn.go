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
	"sort"
	"time"

	json "github.com/json-iterator/go"
	"github.com/olive-io/bpmn/data"
	"github.com/olive-io/bpmn/flow"
	bact "github.com/olive-io/bpmn/flow_node/activity"
	bp "github.com/olive-io/bpmn/process"
	bpi "github.com/olive-io/bpmn/process/instance"
	"github.com/olive-io/bpmn/schema"
	"github.com/olive-io/bpmn/tracing"
	"go.uber.org/zap"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	dsypb "github.com/olive-io/olive/apis/pb/discovery"
	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/pkg/bytesutil"
)

var (
	definitionPrefix = []byte("definitions")
	processPrefix    = []byte("processes")
)

func (s *Shard) GetDefinitionArchive(ctx context.Context, namespace, name string) ([]*corev1.Definition, error) {
	prefix := bytesutil.PathJoin(definitionPrefix, []byte(namespace), []byte(name))

	definitions := make([]*corev1.Definition, 0)
	kvs, err := s.getRange(prefix, getPrefix(prefix), 0)
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		definition := &corev1.Definition{}
		_ = definition.Unmarshal(kv.Value)
		definitions = append(definitions, definition)
	}

	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Spec.Version < definitions[j].Spec.Version
	})

	return definitions, nil
}

func (s *Shard) DeployDefinition(ctx context.Context, req *pb.ShardDeployDefinitionRequest) (*pb.ShardDeployDefinitionResponse, error) {
	resp := &pb.ShardDeployDefinitionResponse{}
	result, err := s.raftRequestOnce(ctx, &pb.RaftInternalRequest{DeployDefinition: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.ShardDeployDefinitionResponse)
	return resp, nil
}

func (s *Shard) RunBpmnProcess(ctx context.Context, req *pb.ShardRunBpmnProcessRequest) (*pb.ShardRunBpmnProcessResponse, error) {
	resp := &pb.ShardRunBpmnProcessResponse{}
	result, err := s.raftRequestOnce(ctx, &pb.RaftInternalRequest{RunBpmnProcess: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.ShardRunBpmnProcessResponse)
	return resp, nil
}

func (s *Shard) GetProcessStat(ctx context.Context, definitionId string, definitionVersion uint64, id string) (*corev1.ProcessStat, error) {
	prefix := bytesutil.PathJoin(processPrefix,
		[]byte(definitionId),
		[]byte(fmt.Sprintf("%d", definitionVersion)))
	key := bytesutil.PathJoin(prefix, []byte(id))

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.cfg.ReqTimeout())
		defer cancel()
	}

	req := &pb.ShardRangeRequest{
		Key:          key,
		Limit:        1,
		Serializable: true,
	}
	rangePrefix(req)
	rsp, err := s.Range(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, ErrNotFound
	}
	ps := new(corev1.ProcessStat)
	if err = ps.Unmarshal(rsp.Kvs[0].Value); err != nil {
		return nil, err
	}

	return ps, nil
}

func (s *Shard) scheduleCycle() {
	duration := 100 * time.Millisecond
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		if !s.waitUtilLeader() {
			select {
			case <-s.stopc:
				return
			default:
			}
			continue
		}

		timer.Reset(duration)

	LOOP:
		for {
			select {
			case <-s.stopc:
				return
			case <-s.changeC:
				break LOOP
			case <-timer.C:
				timer.Reset(duration)

				x, ok := s.processQ.Pop()
				if !ok {
					break
				}
				pi := x.(*ProcessInfo)

				go s.runProcess(pi.Stat)
			}
		}
	}
}

func (s *Shard) runProcess(stat *corev1.ProcessStat) {
	var definitions *schema.Definitions
	err := xml.Unmarshal([]byte(stat.Status.DefinitionContent), &definitions)
	if err != nil {
		return
	}
	s.metric.runningDefinition.Inc()
	defer s.metric.runningDefinition.Dec()

	locator := data.NewFlowDataLocator()
	if len(*definitions.Processes()) == 0 {
		return
	}
	processElement := (*definitions.Processes())[0]
	//if id, ok := processElement.Id(); ok {
	//	process.DefinitionsProcess = *id
	//}
	initContext := stat.Spec.InitContext
	var dataObjects map[string][]byte
	for name, value := range initContext.DataObjects {
		dataObjects[name] = []byte(value)
	}

	variables := make(map[string][]byte)
	for name, value := range initContext.Properties {
		box, _ := json.Marshal(value)
		variables[name] = box
	}

	runningContext := stat.Status.Context
	for key, value := range runningContext.Properties {
		variables[key] = []byte(value)
	}
	for key, value := range runningContext.DataObjects {
		dataObjects[key] = []byte(value)
	}
	for key, value := range runningContext.Headers {
		variables[key] = []byte(value)
	}

	ctx := context.Background()
	options := []bpi.Option{
		bpi.WithLocator(locator),
		bpi.WithDataObjects(toAnyMap[[]byte](dataObjects)),
		bpi.WithVariables(toAnyMap[[]byte](variables)),
	}
	s.scheduleProcess(ctx, stat, definitions, &processElement, options...)
}

func (s *Shard) scheduleProcess(ctx context.Context, stat *corev1.ProcessStat, definitions *schema.Definitions, processElement *schema.Process, options ...bpi.Option) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.metric.process.Inc()
	defer s.metric.process.Dec()

	fields := []zap.Field{
		zap.String("definition", stat.Spec.DefinitionName),
		zap.Int64("version", stat.Spec.DefinitionVersion),
		zap.String("process", stat.Spec.ProcessName),
	}

	s.lg.Info("start process instance", fields...)

	var err error
	if stat.Status.Phase == corev1.ProcessPrepare {
		stat.Status.Phase = corev1.ProcessRunning
		//process.Status.StartTime = time.Now().UnixNano()
		err = s.saveProcess(ctx, stat)
		if err != nil {
			s.lg.Error("save process instance", append(fields, zap.Error(err))...)
		}
	}

	flowNodes := map[string]*corev1.FlowNodeStat{}
	for _, node := range stat.Status.FlowNodes {
		flowNodes[node.Id] = node
	}

	finish := true
	defer func() {
		if !finish {
			return
		}

		stat.Status.EndTime = time.Now().UnixNano()
		stat.Status.Phase = corev1.ProcessSuccess
		if err != nil {
			stat.Status.Phase = corev1.ProcessFailed
			stat.Status.Message = err.Error()
		}

		err = s.saveProcess(ctx, stat)
		if err != nil {
			s.lg.Error("finish process instance", append(fields, zap.Error(err))...)
		} else {
			s.lg.Info("finish process instance", fields...)
		}

		s.tracer.Trace(&processStatTrace{stat: stat})
	}()

	proc := bp.New(processElement, definitions)
	var inst *bpi.Instance
	inst, err = proc.Instantiate(options...)
	if err != nil {
		s.lg.Error("failed to instantiate the process", append(fields, zap.Error(err))...)
		return
	}
	traces := inst.Tracer.Subscribe()
	err = inst.StartAll(ctx)
	if err != nil {
		cancel()
		s.lg.Error("failed to run the instance", append(fields, zap.Error(err))...)
		return
	}

	finish = false
	completed := map[string]struct{}{}

LOOP:
	for {
		select {
		case <-s.changeNotify():
			s.processQ.Push(NewProcessInfo(stat))
			break LOOP
		case trace := <-traces:
			trace = tracing.Unwrap(trace)
			s.lg.Sugar().Debugf("%#v", trace)
			switch tt := trace.(type) {
			case flow.VisitTrace:
				id, _ := tt.Node.Id()

				switch tt.Node.(type) {
				case schema.EndEventInterface:
				case schema.EventInterface:
					s.metric.event.Inc()
				}
				flowNode, ok := flowNodes[*id]
				if ok {
					completed[*id] = struct{}{}
				} else {
					flowNode = &corev1.FlowNodeStat{
						Id:        *id,
						StartTime: time.Now().UnixNano(),
					}
					if name, has := tt.Node.Name(); has {
						flowNode.Name = *name
					}
					flowNodes[*id] = flowNode
					stat.Status.FlowNodes = append(stat.Status.FlowNodes, flowNode)
				}

			case flow.LeaveTrace:
				id, _ := tt.Node.Id()

				flowNode, ok := flowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().UnixNano()
				}

				switch tt.Node.(type) {
				case schema.EventInterface:
					s.metric.event.Dec()
				}

			case bact.ActiveBoundaryTrace:
				if tt.Start {
					s.metric.task.Inc()
				} else {
					s.metric.task.Dec()
					stat.Status.Context.DataObjects = toGenericMap[string](inst.Locator.CloneItems(data.LocatorObject))
					stat.Status.Context.Properties = toGenericMap[string](inst.Locator.CloneItems(data.LocatorProperty))
					stat.Status.Context.Headers = toGenericMap[string](inst.Locator.CloneVariables())
				}
			case *bact.Trace:
				act := tt.GetActivity()
				id, _ := act.Element().Id()
				var ok bool

				flowNode, ok := flowNodes[*id]
				if ok {
					flowNode.EndTime = time.Now().UnixNano()
					headers, dataSets, dataObjects := bact.FetchTaskDataInput(inst.Locator, act.Element())
					flowNode.Context = &corev1.ProcessContext{
						Headers:     toGenericMap[string](headers),
						Properties:  toGenericMap[string](dataSets),
						DataObjects: toGenericMap[string](dataObjects),
					}
				}

				_, ok = completed[*id]
				if ok {
					tt.Do()
				} else {
					s.handleActivity(ctx, tt, inst, stat)
				}
			case tracing.ErrorTrace:
				finish = true
				err = tt.Error
				s.lg.Error("process error occurred", append(fields, zap.Error(err))...)
			case flow.CeaseFlowTrace:
				finish = true
				break LOOP
			default:
			}

			_ = s.saveProcess(ctx, stat)
		}
	}
	inst.Tracer.Unsubscribe(traces)
}

func (s *Shard) handleActivity(ctx context.Context, trace *bact.Trace, inst *bpi.Instance, stat *corev1.ProcessStat) {

	//taskAct := trace.GetActivity()
	//actType := taskAct.Type()
	//dsyAct := &dsypb.Activity{
	//	Type:               actCov(actType),
	//	Definitions:        process.DefinitionsId,
	//	DefinitionsVersion: process.Spec.DefinitionsVersion,
	//}
	//if id, ok := taskAct.Element().Id(); ok {
	//	dsyAct.Id = *id
	//}
	//if name, ok := taskAct.Element().Name(); ok {
	//	dsyAct.Name = *name
	//}
	//if id, ok := inst.Process().Id(); ok {
	//	dsyAct.Process = *id
	//}
	//if extension, ok := taskAct.Element().ExtensionElements(); ok {
	//	if td := extension.TaskDefinitionField; td != nil {
	//		dsyAct.TaskType = td.Type
	//	}
	//}
	//
	//fields := []zap.Field{
	//	zap.String("definition", process.DefinitionsId),
	//	zap.Uint64("version", process.DefinitionsVersion),
	//	zap.String("process", process.Id),
	//	zap.String("task", dsyAct.Id),
	//}
	//
	//headers := toGenericMap[string](trace.GetHeaders())
	//properties := trace.GetProperties()
	//dataObjects := trace.GetDataObjects()
	//
	//req := &gatewaypb.TransmitRequest{
	//	Activity:    dsyAct,
	//	Headers:     headers,
	//	Properties:  map[string]*dsypb.Box{},
	//	DataObjects: map[string]*dsypb.Box{},
	//}
	//for key, value := range properties {
	//	req.Properties[key] = dsypb.BoxFromAny(value)
	//}
	//for key, value := range dataObjects {
	//	req.DataObjects[key] = dsypb.BoxFromAny(value)
	//}

	doOpts := make([]bact.DoOption, 0)
	//resp, err := r.proxy.Handle(ctx, req)
	//if err != nil {
	//	err = fmt.Errorf("%s", err.Error())
	//	r.lg.Error("handle task", append(fields, zap.Error(err))...)
	//	doOpts = append(doOpts, bact.WithErr(err))
	//} else {
	//	dp := map[string]any{}
	//	ddo := map[string]any{}
	//	for key, value := range resp.Properties {
	//		dp[key] = value.Value()
	//	}
	//	for key, value := range resp.DataObjects {
	//		ddo[key] = value.Value()
	//	}
	//	doOpts = append(doOpts, bact.WithProperties(dp), bact.WithObjects(ddo))
	//}
	trace.Do(doOpts...)
}

func (s *Shard) saveProcess(ctx context.Context, stat *corev1.ProcessStat) error {
	var err error
	pkey := bytesutil.PathJoin(processPrefix,
		[]byte(stat.Namespace), []byte(stat.Spec.DefinitionName),
		[]byte(fmt.Sprintf("%d", stat.Spec.DefinitionVersion)), []byte(stat.Spec.ProcessName))

	value, err := stat.Marshal()
	if err != nil {
		return err
	}
	_, err = s.Put(ctx, &pb.ShardPutRequest{Key: pkey, Value: value})

	return err
}

// actCov converts bact.Type to discovery.ActivityType
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
