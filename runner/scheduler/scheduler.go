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

package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/data"
	"github.com/olive-io/bpmn/v2/pkg/tracing"
	"github.com/panjf2000/ants/v2"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/delegate"
	"github.com/olive-io/olive/runner/metrics"
	"github.com/olive-io/olive/runner/storage"
)

var ErrStopped = errors.New("scheduler stopped")

type ProcessItem struct {
	*types.ProcessInstance
}

func (item *ProcessItem) ID() int64 {
	return item.Id
}

func (item *ProcessItem) Score() int64 {
	return 100
}

func ProcessItemStore(v *ProcessItem) int64 {
	return v.Score()
}

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *zap.Logger

	wg sync.WaitGroup

	idGen *idutil.Generator
	bs    storage.Storage

	// inner process queue
	processQ *queue.SyncPriorityQueue[*ProcessItem]

	pool *ants.Pool

	done chan struct{}
}

func NewScheduler(cfg *Config) (*Scheduler, error) {

	pq := queue.NewSync[*ProcessItem](ProcessItemStore)

	pool, err := ants.NewPool(cfg.PoolSize)
	if err != nil {
		return nil, err
	}

	parentCtx := cfg.Context
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)

	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewExample()
	}
	sche := &Scheduler{
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		wg:       sync.WaitGroup{},
		idGen:    cfg.idGen,
		bs:       cfg.Storage,
		processQ: pq,
		pool:     pool,
		done:     make(chan struct{}, 1),
	}

	return sche, nil
}

func (s *Scheduler) Start() error {
	items := s.undoneTasks(s.ctx)
	for _, item := range items {
		s.processQ.Push(item)
	}

	go s.process()
	return nil
}

func (s *Scheduler) Stop() error {
	select {
	case <-s.done:
		return ErrStopped
	default:
	}

	s.cancel()
	s.wg.Wait()
	close(s.done)

	return nil
}

func (s *Scheduler) RunProcess(
	ctx context.Context,
	definitionId int64,
	definitionVersion uint64,
	content string,
	process string,
	instanceName string,
	headers map[string]string,
	properties map[string][]byte,
	dataObjects map[string][]byte,
) (*types.ProcessInstance, error) {

	err := s.saveDefinition(ctx, definitionId, instanceName, definitionVersion, content)
	if err != nil {
		return nil, err
	}

	instance := &types.ProcessInstance{
		Name: instanceName,
		Args: &types.BpmnArgs{
			Headers:     headers,
			Properties:  properties,
			DataObjects: dataObjects,
		},
		DefinitionsId:      definitionId,
		DefinitionsVersion: definitionVersion,
		DefinitionsProcess: process,
		DefinitionsContent: content,
		Context: &types.ProcessContext{
			DataObjects: make(map[string][]byte),
			Variables:   make(map[string][]byte),
		},
		FlowNodes:       make([]*types.FlowNode, 0),
		FlowNodeStatMap: make(map[string]*types.FlowNodeStat),
		Attempts:        0,
		StartAt:         0,
		EndAt:           0,
		Status:          types.ProcessStatus_Ready,
		Message:         "",
	}

	if err = s.saveProcess(ctx, instance); err != nil {
		return nil, err
	}

	item := &ProcessItem{instance}

	s.logger.Info("process push to scheduler queue", zap.String("name", instance.Name))
	s.processQ.Push(item)

	return instance, nil
}

func (s *Scheduler) process() {
	timer := time.NewTicker(time.Millisecond * 10)
	defer timer.Stop()

	ctx := s.ctx
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		value, ok := s.processQ.Pop()
		if !ok {
			continue
		}

		item, ok := value.(*ProcessItem)
		if !ok {
			continue
		}

		if !s.poolAble() {
			s.processQ.Push(item)
			continue
		}

		err := s.pool.Submit(s.buildProcessTask(item.ProcessInstance))
		if err != nil {
			s.logger.Sugar().Errorf("submit process: %v", err)
		}
	}
}

func (s *Scheduler) buildProcessTask(instance *types.ProcessInstance) func() {
	return func() {
		s.wg.Add(1)

		ctx, cancel := context.WithCancel(s.ctx)

		metrics.ProcessCounter.Inc()

		defer func() {
			s.wg.Done()
			cancel()

			metrics.ProcessCounter.Dec()
		}()

		instance.StartAt = time.Now().UnixNano()
		instance.Status = types.ProcessStatus_Ready
		_ = s.saveProcess(ctx, instance)

		variables := make(map[string]any)
		dataObjects := make(map[string]any)

		for key, value := range instance.Args.Headers {
			variables[key] = value
		}
		for key, value := range instance.Args.Properties {
			variables[key] = string(value)
		}
		for key, value := range instance.Args.DataObjects {
			dataObjects[key] = string(value)
		}

		for key, value := range instance.Context.Variables {
			variables[key] = string(value)
		}
		for key, value := range instance.Context.DataObjects {
			dataObjects[key] = string(value)
		}

		zapFields := []zap.Field{
			zap.Int64("definition", instance.DefinitionsId),
			zap.Uint64("version", instance.DefinitionsVersion),
			zap.Int64("id", instance.Id),
			zap.String("name", instance.Name),
		}

		s.logger.Info("process instance started", zapFields...)
		var err error
		defer func() {
			s.logger.Info("process instance finished", zapFields...)
			_ = s.finishProcess(ctx, instance, err)
		}()

		content := []byte(instance.DefinitionsContent)
		opts := []bpmn.Option{
			bpmn.WithContext(ctx),
			bpmn.WithVariables(variables),
			bpmn.WithDataObjects(dataObjects),
		}

		var process *Process
		process, err = NewProcess(ctx, content, opts...)
		if err != nil {
			s.logger.Error("failed to create process", append(zapFields, zap.Error(err))...)
			return
		}

		flowNodeMap := instance.FlowNodeStatMap
		if flowNodeMap == nil {
			flowNodeMap = make(map[string]*types.FlowNodeStat)
		}

		instance.Status = types.ProcessStatus_Running
		err = process.Run(func(trace tracing.ITrace, locator data.IFlowDataLocator) {
			switch tr := trace.(type) {
			case *bpmn.TaskTrace:
				id := flowId(tr.GetActivity().Element())
				stat, ok := flowNodeMap[id]
				if !ok {
					tr.Do()
					return
				}

				if stat.EndTime != 0 {
					tr.Do()
					return
				}

				taskCtx := tr.Context()
				activity := tr.GetActivity()
				headers := tr.GetHeaders()
				properties := tr.GetProperties()
				objects := tr.GetDataObjects()

				var results map[string]any
				var outs map[string]any
				results, outs, err = s.taskDelegate(taskCtx, activity, headers, properties, objects)

				doOpts := []bpmn.DoOption{}
				if len(results) > 0 {
					doOpts = append(doOpts, bpmn.DoWithResults(results))
				}
				if len(outs) > 0 {
					doOpts = append(doOpts, bpmn.DoWithResults(outs))
				}
				if err != nil {
					doOpts = append(doOpts, bpmn.DoWithErr(err))
					stat.Message = err.Error()
				}
				tr.Do(doOpts...)

				for key, value := range locator.CloneVariables() {
					var vv []byte
					switch tv := value.(type) {
					case string:
						vv = []byte(tv)
					}
					stat.Context.Variables[key] = vv
				}
				flowNodeMap[id] = stat

			case bpmn.ErrorTrace:
				err = tr.Error
				instance.Message = tr.Error.Error()

			case bpmn.ActiveBoundaryTrace:
				id := flowId(tr.Node)
				if stat, ok := flowNodeMap[id]; ok {
					if tr.Start {
						stat.Retries += 1
						metrics.TaskCounter.Inc()
					} else {
						metrics.TaskCounter.Dec()
					}
					flowNodeMap[id] = stat
				}
			case bpmn.VisitTrace:
				id := flowId(tr.Node)

				stat, ok := flowNodeMap[id]
				if !ok {
					nodeType := flowType(tr.Node)
					flowNode := &types.FlowNode{Type: nodeType, Id: id}
					instance.FlowNodes = append(instance.FlowNodes, flowNode)

					stat = &types.FlowNodeStat{
						Id: id,
						Context: &types.ProcessContext{
							Variables:   make(map[string][]byte),
							DataObjects: make(map[string][]byte),
						},
						Retries:   -1,
						StartTime: time.Now().UnixNano(),
					}
					if ptr, ok1 := tr.Node.Name(); ok1 {
						stat.Name = *ptr
					}
					flowNodeMap[id] = stat
				}

				if isEvent(tr.Node) {
					metrics.EventCounter.Inc()
				}

			case bpmn.LeaveTrace:
				id := flowId(tr.Node)
				if stat, ok := flowNodeMap[id]; ok {
					stat.EndTime = time.Now().UnixNano()
					flowNodeMap[id] = stat
				}

				if isEvent(tr.Node) {
					metrics.EventCounter.Dec()
				}

			default:
			}

			_ = s.saveProcess(ctx, instance)
		})

		if err != nil {
			s.logger.Error("failed to run process", append(zapFields, zap.Error(err))...)
			return
		}
	}
}

func (s *Scheduler) taskDelegate(
	ctx context.Context,
	activity bpmn.Activity,
	headers map[string]string,
	properties map[string]any,
	dataObjects map[string]any,
) (map[string]any, map[string]any, error) {

	taskType := flowType(activity.Element())
	var subType string
	if extension, found := activity.Element().ExtensionElements(); found {
		if field := extension.TaskDefinitionField; field != nil {
			subType = field.Type
		}
	}
	theme := delegate.Theme{
		Major: taskType,
		Minor: subType,
	}

	url := theme.String()
	dg, err := delegate.GetDelegate(url)
	if err != nil {
		return nil, nil, fmt.Errorf("match delegate %s: %w", url, err)
	}

	timeout := bpmn.FetchTaskTimeout(activity.Element())
	if timeout <= 0 {
		timeout = delegate.DefaultTimeout
	}
	req := &delegate.Request{
		Headers:     headers,
		Properties:  properties,
		DataObjects: dataObjects,
		Timeout:     timeout,
	}

	resp, err := dg.Call(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	return resp.Result, resp.DataObjects, nil
}

func (s *Scheduler) poolAble() bool {
	free := s.pool.Free()
	return free > 0 || free == -1
}
