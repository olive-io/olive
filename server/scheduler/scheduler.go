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
	"fmt"
	"sync"
	"time"

	"github.com/olive-io/bpmn/schema"
	bpmn "github.com/olive-io/bpmn/v2"
	"github.com/olive-io/bpmn/v2/pkg/tracing"
	ants "github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

type antLogger struct {
	lg *zap.SugaredLogger
}

func (l *antLogger) Printf(format string, args ...any) {
	l.lg.Debugf(format, args...)
}

type ProcessStat struct {
	*types.Process `json:",inline"`

	FlowNodes []*types.FlowNode `json:"flowNodes"`
}

func (p *ProcessStat) ID() int64 {
	return p.Id
}

type Scheduler struct {
	ctx     context.Context
	cancel  context.CancelFunc
	options *Options

	lg *zap.Logger

	queue *SyncPriorityQueue[*ProcessStat]

	executePool *ants.Pool

	engine *bpmn.Engine

	wmu      sync.RWMutex
	watchers map[string]*WatchChan
}

func NewScheduler(pctx context.Context, options *Options) (*Scheduler, error) {
	lg := options.Logger
	if lg == nil {
		lg = zap.NewNop()
	}

	poolOpts := []ants.Option{
		ants.WithPreAlloc(true),
		ants.WithLogger(&antLogger{lg: lg.Sugar()}),
	}
	executePool, err := ants.NewPool(options.ExecutePoolSize, poolOpts...)
	if err != nil {
		return nil, fmt.Errorf("create execute pool: %w", err)
	}

	ctx, cancel := context.WithCancel(pctx)
	queue := NewSync[*ProcessStat](func(v *ProcessStat) int64 {
		score := v.Priority
		if v.Stage == types.Process_Rollback ||
			v.Stage == types.Process_Destroy {
			score += 300
		}
		if v.Stage == types.Process_Commit {
			score += 200
		}
		return score
	})

	engine := bpmn.NewEngine(bpmn.WithEngineContext(ctx))

	scheduler := &Scheduler{
		ctx:         ctx,
		cancel:      cancel,
		lg:          lg,
		options:     options,
		queue:       queue,
		executePool: executePool,
		engine:      engine,
		watchers:    make(map[string]*WatchChan),
	}

	go scheduler.process(ctx)
	return scheduler, nil
}

func (sch *Scheduler) Execute(stat *ProcessStat) error {
	lg := sch.lg

	lg.Info("push new process",
		zap.Int64("pid", stat.Id),
		zap.Int64("priority", stat.Priority),
		zap.String("status", stat.Status.String()),
		zap.String("stage", stat.Stage.String()))

	sch.queue.Push(stat)
	return nil
}

func (sch *Scheduler) Watch(ctx context.Context, id string) *WatchChan {
	sch.lg.Sugar().Infof("add new watcher [%s]", id)

	wch := newWatchChan(ctx, id, sch.lg, sch.ctx.Done())

	sch.wmu.Lock()
	sch.watchers[id] = wch
	sch.wmu.Unlock()

	return wch
}

func (sch *Scheduler) process(ctx context.Context) {

	tick := time.Microsecond * 100
	timer := time.NewTimer(tick)
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-timer.C:
			timer.Reset(tick)

			sch.tick(ctx)
		}
	}

	sch.destroy()
}

func (sch *Scheduler) tick(ctx context.Context) {
	free := sch.executePool.Free()
	if free == 0 {
		return
	}
	value, ok := sch.queue.Pop()
	if !ok {
		return
	}
	stat := value.(*ProcessStat)
	err := sch.executePool.Submit(func() {
		err := sch.execute(ctx, stat)
		if err != nil {
			//TODO: handle error
		}
	})
	if err != nil {
		sch.queue.Push(stat)
	}
}

func (sch *Scheduler) execute(ctx context.Context, stat *ProcessStat) error {
	lg := sch.lg.Sugar()

	var err error
	defer func() {
		stat.EndAt = time.Now().UnixNano()
		stat.Stage = types.Process_Finish

		stat.Status = types.Process_Success
		if err != nil {
			stat.ErrMsg = err.Error()
			stat.Status = types.Process_Failed
		}

		sch.setProcess(stat.Process)
	}()

	var definitions *schema.Definitions
	definitions, err = schema.Parse([]byte(stat.DefinitionsContent))
	if err != nil {
		return err
	}

	options := make([]bpmn.Option, 0)
	if pctx := stat.Context; pctx != nil {
		dataObjects := map[string]interface{}{}
		for name, dataObject := range pctx.DataObjects {
			dataObjects[name] = dataObject
		}
		options = append(options, bpmn.WithDataObjects(dataObjects))
		variables := map[string]interface{}{}
		for name, variable := range dataObjects {
			variables[name] = variable
		}
		options = append(options, bpmn.WithVariables(variables))
	}

	var bp *bpmn.Process
	bp, err = sch.engine.NewProcess(definitions, options...)
	if err != nil {
		return err
	}

	pid := bp.Id().String()
	nodeMapping := make(map[string]*types.FlowNode)
	for _, node := range stat.FlowNodes {
		nodeMapping[node.Name] = node
	}

	traces := bp.Tracer().Subscribe()
	defer bp.Tracer().Unsubscribe(traces)
	if err = bp.StartAll(ctx); err != nil {
		return err
	}

	stack := make([]*types.FlowNode, 0)
	stat.StartAt = time.Now().UnixNano()
	stat.Status = types.Process_Running
	stat.Stage = types.Process_Commit
	sch.setProcess(stat.Process)

	ech := make(chan error, 1)
	go func(ech chan<- error) {
		for {
			var trace tracing.ITrace
			select {
			case trace = <-traces:
			case <-ctx.Done():
			}

			trace = tracing.Unwrap(trace)
			switch tt := trace.(type) {
			case bpmn.VisitTrace:
				elem := tt.Node
				var fid string
				id, ok := elem.Id()
				if ok {
					fid = *id
				}
				var fname string
				name, ok := elem.Name()
				if ok {
					fname = *name
				}

				node, exists := nodeMapping[fid]
				if !exists {
					node = &types.FlowNode{
						Name:      fname,
						FlowId:    fid,
						FlowType:  parseElementType(elem),
						StartTime: time.Now().UnixNano(),
						ProcessId: stat.Id,
						Stage:     types.FlowNode_Ready,
					}
					nodeMapping[fid] = node
					stat.FlowNodes = append(stat.FlowNodes, node)
					sch.setFlowNode(node)
				}

			case bpmn.LeaveTrace:
				elem := tt.Node
				var fid string
				id, ok := elem.Id()
				if ok {
					fid = *id
				}
				node, exists := nodeMapping[fid]
				if exists {
					node.EndTime = time.Now().UnixNano()
					node.Stage = types.FlowNode_Finish
					sch.setFlowNode(node)
				}

			case bpmn.TaskTrace:
				act := tt.GetActivity()
				elem := act.Element()

				var fid string
				id, ok := elem.Id()
				if ok {
					fid = *id
				}
				var tname string
				name, ok := elem.Name()
				if ok {
					tname = *name
				}
				lg.Info("commit task [%s][%s] ", pid, tname)

				flowNode, exists := nodeMapping[fid]
				if !exists {
					flowNode = &types.FlowNode{
						Name:      tname,
						FlowId:    fid,
						FlowType:  parseTaskType(act.Type()),
						StartTime: time.Now().UnixNano(),
						ProcessId: stat.Id,
						Stage:     types.FlowNode_Commit,
						Status:    types.FlowNode_Running,
					}
					nodeMapping[fid] = flowNode
					stat.FlowNodes = append(stat.FlowNodes, flowNode)
					sch.setFlowNode(flowNode)
				}

				if flowNode.Stage != types.FlowNode_Commit &&
					flowNode.Stage != types.FlowNode_Rollback &&
					flowNode.Stage != types.FlowNode_Destroy {

					headers := tt.GetHeaders()
					properties := map[string]string{}
					dataObjects := map[string]string{}
					for k, v := range tt.GetProperties() {
						properties[k] = jsonValueToString(v)
					}
					for k, v := range tt.GetDataObjects() {
						dataObjects[k] = jsonValueToString(v)
					}

					flowNode.Headers = headers
					flowNode.Properties = properties
					flowNode.DataObjects = dataObjects
					flowNode.Stage = types.FlowNode_Commit
					sch.setFlowNode(flowNode)

					var taskErr error
					//TODO: handle task commit

					doOptions := make([]bpmn.DoOption, 0)
					flowNode.Status = types.FlowNode_Success
					if taskErr != nil {
						flowNode.Status = types.FlowNode_Failed
						flowNode.ErrMsg = taskErr.Error()
						doOptions = append(doOptions, bpmn.DoWithErr(err))
					}

					tt.Do(doOptions...)

					sch.setFlowNode(flowNode)

					stack = append(stack, flowNode)
				} else {
					tt.Do()
				}

			case bpmn.ActiveBoundaryTrace:

			case bpmn.ErrorTrace:
				ech <- tt.Error
				return
			case bpmn.CeaseFlowTrace:
				return
			default:
			}
		}
	}(ech)

	bp.WaitUntilComplete(ctx)
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return err
	case err = <-ech:
	}

	if err != nil {
		stat.Stage = types.Process_Rollback
		sch.setProcess(stat.Process)

		for i := len(stack) - 1; i >= 0; i-- {
			node := stack[i]
			node.Stage = types.FlowNode_Rollback
			sch.setFlowNode(node)

			lg.Info("rollback task [%s][%s] ", pid, node.Name)

			//TODO: do task rollback stage
		}
	}

	stat.Stage = types.Process_Destroy
	sch.setProcess(stat.Process)

	for i := len(stack) - 1; i >= 0; i-- {
		node := stack[i]
		node.Stage = types.FlowNode_Destroy
		sch.setFlowNode(node)

		lg.Info("destroy task [%s][%s]", pid, node.Name)

		//TODO: do task commit stage

		node.Stage = types.FlowNode_Finish
		sch.setFlowNode(node)
	}

	return nil
}

func (sch *Scheduler) destroy() {
	sch.lg.Debug("destroying scheduler")

	sch.cancel()

	// release goroutines pool
	sch.lg.Debug("release scheduler execute pool")
	sch.executePool.Release()
	sch.lg.Debug("released scheduler execute pool")
}

func (sch *Scheduler) setProcess(process *types.Process) {
	sch.sendWatchMsg(&WatchMessage{Process: process.DeepCopy()})
}

func (sch *Scheduler) setFlowNode(flowNode *types.FlowNode) {
	sch.sendWatchMsg(&WatchMessage{FlowNode: flowNode.DeepCopy()})
}

func (sch *Scheduler) sendWatchMsg(msg *WatchMessage) {
	deleted := make([]string, 0)
	sch.wmu.RLock()
	for name, w := range sch.watchers {
		if w.IsClosed() {
			deleted = append(deleted, name)
			continue
		}
		w.send(msg)
	}
	sch.wmu.RUnlock()

	for _, name := range deleted {
		sch.deleteWatcher(name)
	}
}

func (sch *Scheduler) deleteWatcher(name string) {
	sch.wmu.Lock()
	delete(sch.watchers, name)
	sch.wmu.Unlock()
}
