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

	"github.com/google/uuid"
	"github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/data"
	"github.com/olive-io/bpmn/v2/pkg/tracing"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/delegate"
	"github.com/olive-io/olive/runner/metrics"
	"github.com/olive-io/olive/runner/storage"
)

var ErrStopped = errors.New("scheduler stopped")

type ProcessItem struct {
	*corev1.Process
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

	bs *storage.Storage

	wg sync.WaitGroup

	// inner process queue
	spq *queue.SyncPriorityQueue[*ProcessItem]

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
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
		bs:     cfg.Storage,
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
	definitionName string,
	definitionVersion int64,
	content string,
	process string,
	instanceName string,
	headers map[string]string,
	properties map[string][]byte,
	dataObjects map[string][]byte,
) (*corev1.Process, error) {

	err := s.saveDefinition(ctx, definitionName, instanceName, definitionVersion, content)
	if err != nil {
		return nil, err
	}

	id := uuid.New().String()
	instance := &corev1.Process{
		ObjectMeta: metav1.ObjectMeta{
			UID:               types.UID(id),
			Name:              instanceName,
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ProcessSpec{
			DefinitionsName:    definitionName,
			DefinitionsVersion: definitionVersion,
			DefinitionsProcess: process,
			DefinitionsContent: content,
			Args: corev1.BpmnArgs{
				Headers:     headers,
				Properties:  properties,
				DataObjects: dataObjects,
			},
		},
		Status: corev1.ProcessStatus{
			Phase: corev1.ProcessPending,
			Context: corev1.ProcessContext{
				DataObjects: make(map[string][]byte),
				Variables:   make(map[string][]byte),
			},
			FlowNodes:       make([]corev1.FlowNode, 0),
			FlowNodeStatMap: make(map[string]corev1.FlowNodeStat),
		},
	}

	if err = s.saveProcess(ctx, instance); err != nil {
		return nil, err
	}

	item := &ProcessItem{instance}

	s.logger.Info("process push to scheduler queue", zap.String("name", instance.Name))
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

		err := s.pool.Submit(s.buildProcessTask(item.Process))
		if err != nil {
			s.logger.Sugar().Errorf("submit process: %v", err)
		}
	}
}

func (s *Scheduler) buildProcessTask(process *corev1.Process) func() {
	return func() {
		s.wg.Add(1)

		ctx, cancel := context.WithCancel(s.ctx)

		metrics.ProcessCounter.Inc()

		defer func() {
			s.wg.Done()
			cancel()

			metrics.ProcessCounter.Dec()
		}()

		process.Status.Phase = corev1.ProcessPrepare
		process.Status.StartTimestamp = time.Now().UnixNano()
		_ = s.saveProcess(ctx, process)

		variables := make(map[string]any)
		dataObjects := make(map[string]any)

		for key, value := range process.Spec.Args.Headers {
			variables[key] = value
		}
		for key, value := range process.Spec.Args.Properties {
			variables[key] = string(value)
		}
		for key, value := range process.Spec.Args.DataObjects {
			dataObjects[key] = string(value)
		}

		for key, value := range process.Status.Context.Variables {
			variables[key] = string(value)
		}
		for key, value := range process.Status.Context.DataObjects {
			dataObjects[key] = string(value)
		}

		zapFields := []zap.Field{
			zap.String("definition", process.Spec.DefinitionsName),
			zap.Int64("version", process.Spec.DefinitionsVersion),
			zap.String("id", string(process.UID)),
			zap.String("name", process.Name),
		}

		s.logger.Info("process instance started", zapFields...)
		var err error
		defer func() {
			s.logger.Info("process instance finished", zapFields...)
			_ = s.finishProcess(ctx, process, err)
		}()

		content := []byte(process.Spec.DefinitionsContent)
		opts := []bpmn.Option{
			bpmn.WithContext(ctx),
			bpmn.WithVariables(variables),
			bpmn.WithDataObjects(dataObjects),
		}

		var pr *Process
		pr, err = NewProcess(ctx, content, opts...)
		if err != nil {
			s.logger.Error("failed to create process", append(zapFields, zap.Error(err))...)
			return
		}

		flowNodeMap := process.Status.FlowNodeStatMap
		if flowNodeMap == nil {
			flowNodeMap = make(map[string]corev1.FlowNodeStat)
		}

		process.Status.Phase = corev1.ProcessRunning
		err = pr.Run(func(trace tracing.ITrace, locator data.IFlowDataLocator) {
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
				process.Status.Message = tr.Error.Error()

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
					process.Status.FlowNodes = append(process.Status.FlowNodes, flowNode)

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

			_ = s.saveProcess(ctx, process)
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
