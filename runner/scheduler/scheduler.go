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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/data"
	"github.com/olive-io/bpmn/v2/pkg/tracing"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
	metav1 "github.com/olive-io/olive/api/types/meta/v1"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/metrics"
	"github.com/olive-io/olive/runner/storage/backend"
)

var ErrStopped = errors.New("scheduler stopped")

type ProcessItem struct {
	*corev1.ProcessInstance
}

func (item *ProcessItem) Score() int64 {
	return 100
}

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger *zap.Logger
	db     backend.IBackend

	wg sync.WaitGroup

	// inner process queue
	spq *queue.SyncPriorityQueue[*ProcessItem]

	pool *ants.Pool

	done chan struct{}
}

func NewScheduler(cfg *Config) (*Scheduler, error) {

	pq := queue.NewSync[*ProcessItem](func(v *ProcessItem) int64 {
		return v.Score()
	})

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
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
		db:     cfg.DB,
		wg:     sync.WaitGroup{},
		spq:    pq,
		pool:   pool,
		done:   make(chan struct{}, 1),
	}

	return sche, nil
}

func (s *Scheduler) Start() error {
	items := s.undoneTasks(s.ctx)
	for _, item := range items {
		s.spq.Push(item)
	}

	go s.process()
	return nil
}

func (s *Scheduler) Stop() error {
	select {
	case s.done <- struct{}{}:
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
	definitionId string,
	definitionVersion int64,
	content string,
	process string,
	instanceName string,
	headers map[string]string,
	properties map[string][]byte,
	dataObjects map[string][]byte,
) (*corev1.ProcessInstance, error) {

	err := s.saveDefinition(ctx, definitionId, instanceName, definitionVersion, content)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()
	instance := &corev1.ProcessInstance{
		ObjectMeta: metav1.ObjectMeta{
			UID:               id,
			Name:              instanceName,
			CreationTimestamp: time.Now().UnixNano(),
		},
		DefinitionsId:      definitionId,
		DefinitionsVersion: definitionVersion,
		DefinitionsProcess: process,
		DefinitionsContent: content,
		Args: corev1.BpmnArgs{
			Headers:     headers,
			Properties:  properties,
			DataObjects: dataObjects,
		},
		Context: corev1.ProcessContext{
			DataObjects: make(map[string][]byte),
			Variables:   make(map[string][]byte),
		},
		FlowNodes:       make([]corev1.FlowNode, 0),
		FlowNodeStatMap: make(map[string]corev1.FlowNodeStat),
		Status:          corev1.ProcessWaiting,
	}

	if err = s.saveProcess(ctx, instance); err != nil {
		return nil, err
	}

	item := &ProcessItem{instance}
	s.spq.Push(item)

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

		value, ok := s.spq.Pop()
		if !ok {
			continue
		}

		item, ok := value.(*ProcessItem)
		if !ok {
			continue
		}

		if !s.poolAble() {
			s.spq.Push(item)
			continue
		}

		err := s.pool.Submit(s.buildProcessTask(item.ProcessInstance))
		if err != nil {
			s.logger.Sugar().Errorf("submit process: %v", err)
		}
	}
}

func (s *Scheduler) buildProcessTask(instance *corev1.ProcessInstance) func() {
	return func() {
		s.wg.Add(1)

		ctx, cancel := context.WithCancel(s.ctx)

		metrics.ProcessCounter.Inc()

		defer func() {
			s.wg.Done()
			cancel()

			metrics.ProcessCounter.Dec()
		}()

		instance.StartTimestamp = time.Now().UnixNano()
		instance.Status = corev1.ProcessPrepare
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

		var err error
		defer func() {
			_ = s.finishProcess(ctx, instance, err)
		}()

		zapFields := []zap.Field{
			zap.String("definition", instance.DefinitionsId),
			zap.Int64("version", instance.DefinitionsVersion),
			zap.String("id", instance.UID),
			zap.String("name", instance.Name),
		}

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
			flowNodeMap = make(map[string]corev1.FlowNodeStat)
		}

		instance.Status = corev1.ProcessRunning
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
					flowNode := corev1.FlowNode{Type: nodeType, Id: id}
					instance.FlowNodes = append(instance.FlowNodes, flowNode)

					stat = corev1.FlowNodeStat{
						Id: id,
						Context: corev1.ProcessContext{
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

			s.logger.Sugar().Infof("%#v", trace)
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
) (results map[string]any, outs map[string]any, err error) {
	//TODO: execute task

	//taskType := activity.Type()
	//var subType string
	//if extension, found := activity.Element().ExtensionElements(); found {
	//	if field := extension.TaskDefinitionField; field != nil {
	//		subType = field.Type
	//	}
	//}

	results = map[string]any{
		"a": "foo",
	}
	outs = map[string]any{}
	return
}

func (s *Scheduler) poolAble() bool {
	free := s.pool.Free()
	return free > 0 || free == -1
}
