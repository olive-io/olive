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
	"encoding/binary"
	"fmt"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/gogo/protobuf/proto"
	sm "github.com/lni/dragonboat/v4/statemachine"
	"github.com/olive-io/bpmn/tracing"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

var (
	appliedIndex = []byte("applied_index")
)

const (
	// In the health case, there might be a small gap (10s of entries) between
	// the applied index and committed index.
	// However, if the committed entries are very heavy to apply, the gap might grow.
	// We should stop accepting new proposals if the gap growing to a certain point.
	maxGapBetweenApplyAndCommitIndex = 5000
	traceThreshold                   = 100 * time.Millisecond
)

func processInstanceStoreFn(process *pb.ProcessInstance) int64 {
	return 1
}

type IRegionRaftKV interface {
	Range(ctx context.Context, r *pb.RegionRangeRequest) (*pb.RegionRangeResponse, error)
	Put(ctx context.Context, r *pb.RegionPutRequest) (*pb.RegionPutResponse, error)
	Delete(ctx context.Context, r *pb.RegionDeleteRequest) (*pb.RegionDeleteResponse, error)
}

type Region struct {
	RegionConfig

	id       uint64
	memberId uint64

	lg *zap.Logger

	applyW   wait.Wait
	commitW  wait.Wait
	openWait wait.Wait

	tracer tracing.ITracer

	reqIDGen *idutil.Generator

	be backend.IBackend

	rimu       sync.RWMutex
	regionInfo *pb.Region
	metric     *regionMetrics

	processQ *queue.SyncPriorityQueue[*pb.ProcessInstance]

	applyBase Applier

	applied   uint64
	committed uint64
	term      uint64
	leader    uint64

	changeCMu sync.RWMutex
	changeC   chan struct{}

	leadTimeMu      sync.RWMutex
	leadElectedTime time.Time

	readyCMu sync.RWMutex
	readyC   chan struct{}

	stopc <-chan struct{}
}

func (c *Controller) InitDiskStateMachine(shardId, nodeId uint64) sm.IOnDiskStateMachine {
	reqIDGen := idutil.NewGenerator(uint16(nodeId), time.Now())
	processQ := queue.NewSync[*pb.ProcessInstance](processInstanceStoreFn)

	tracer := c.tracer
	region := &Region{
		id:         shardId,
		memberId:   nodeId,
		lg:         c.Logger,
		openWait:   c.regionW,
		tracer:     tracer,
		applyW:     wait.New(),
		commitW:    wait.New(),
		reqIDGen:   reqIDGen,
		be:         c.be,
		processQ:   processQ,
		regionInfo: &pb.Region{},
		changeC:    make(chan struct{}),
		readyC:     make(chan struct{}),
		stopc:      make(<-chan struct{}, 1),
	}

	applyBase := region.newApplier()
	region.applyBase = applyBase

	c.setRegion(region)

	return region
}

func (r *Region) Range(ctx context.Context, req *pb.RegionRangeRequest) (*pb.RegionRangeResponse, error) {
	trace := traceutil.New("range",
		r.lg,
		traceutil.Field{Key: "range_begin", Value: string(req.Key)},
		traceutil.Field{Key: "range_end", Value: string(req.RangeEnd)},
	)
	ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

	var resp *pb.RegionRangeResponse
	var err error
	defer func(start time.Time) {
		warnOfExpensiveReadOnlyRangeRequest(r.lg, r.metric.slowApplies, r.WarningApplyDuration, start, req, resp, err)
		if resp != nil {
			trace.AddField(
				traceutil.Field{Key: "response_count", Value: len(resp.Kvs)},
			)
		}
		trace.LogIfLong(traceThreshold)
	}(time.Now())

	result, err := r.raftQuery(ctx, pb.RaftInternalRequest{Range: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.RegionRangeResponse)
	return resp, err
}

func (r *Region) Put(ctx context.Context, req *pb.RegionPutRequest) (*pb.RegionPutResponse, error) {
	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{Put: req})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.RegionPutResponse), nil
}

func (r *Region) Delete(ctx context.Context, req *pb.RegionDeleteRequest) (*pb.RegionDeleteResponse, error) {
	resp, err := r.raftRequestOnce(ctx, pb.RaftInternalRequest{Delete: req})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.RegionDeleteResponse), nil
}

func (r *Region) initial(stopc <-chan struct{}) (uint64, error) {
	r.stopc = stopc
	applyIndex, err := r.readApplyIndex()
	if err != nil {
		r.openWait.Trigger(r.id, err)
		return 0, err
	}

	r.metric, err = newRegionMetrics(r.id)
	if err != nil {
		r.openWait.Trigger(r.id, err)
		return 0, err
	}

	r.setApplied(applyIndex)
	r.openWait.Trigger(r.id, r)

	dm := make(map[string]struct{})
	kvs, _ := r.getRange(definitionPrefix, getPrefix(definitionPrefix), 0)
	for _, kv := range kvs {
		definitionId := path.Dir(string(kv.Key))
		if _, ok := dm[definitionId]; !ok {
			dm[definitionId] = struct{}{}
		}
	}
	r.metric.definition.Add(float64(len(dm)))

	kvs, _ = r.getRange(processPrefix, getPrefix(processPrefix), 0)
	for _, kv := range kvs {
		proc := new(pb.ProcessInstance)
		err = proc.Unmarshal(kv.Value)
		if err != nil {
			r.lg.Error("unmarshal process instance", zap.Error(err))
			_ = r.del(kv.Key, true)
			continue
		}
		if proc.Status == pb.ProcessInstance_Unknown ||
			proc.Status == pb.ProcessInstance_Ok ||
			proc.Status == pb.ProcessInstance_Fail ||
			proc.DefinitionId == "" ||
			proc.DefinitionVersion == 0 {
			continue
		}

		if proc.RunningState == nil {
			proc.RunningState = &pb.ProcessRunningState{}
		}
		if proc.FlowNodes == nil {
			proc.FlowNodes = map[string]*pb.FlowNodeStat{}
		}
		if proc.Status == pb.ProcessInstance_Waiting {
			proc.Status = pb.ProcessInstance_Prepare
		}
		r.processQ.Push(proc)
	}

	return applyIndex, nil
}

func (r *Region) waitUtilLeader() bool {
	for {
		if r.isLeader() {
			return true
		}

		select {
		case <-r.stopc:
			return false
		case <-r.changeNotify():
		}
	}
}

func (r *Region) waitLeader(ctx context.Context) (bool, error) {
	for {
		if r.isLeader() {
			return true, nil
		}
		if r.getLeader() != 0 {
			return false, nil
		}

		select {
		case <-ctx.Done():
			return false, context.Canceled
		case <-r.stopc:
			return false, ErrStopped
		case <-r.changeNotify():
		}
	}
}

func (r *Region) raftQuery(ctx context.Context, req pb.RaftInternalRequest) (proto.Message, error) {
	trace := newReadTrace(ctx, r.getID(), &req)
	defer trace.Close()
	r.tracer.Trace(trace)

	arch, ech := trace.Trigger()
	select {
	case err := <-ech:
		return nil, err
	case result := <-arch:
		if result.err != nil {
			return nil, result.err
		}
		if startTime, ok := ctx.Value(traceutil.StartTimeKey).(time.Time); ok && result.trace != nil {
			applyStart := result.trace.GetStartTime()
			// The trace object is created in apply. Here reset the start time to trace
			// the raft request time by the difference between the request start time
			// and apply start time
			result.trace.SetStartTime(startTime)
			result.trace.InsertStep(0, applyStart, "process raft query")
			result.trace.LogIfLong(traceThreshold)
		}
		return result.resp, nil
	}
}

func (r *Region) raftRequestOnce(ctx context.Context, req pb.RaftInternalRequest) (proto.Message, error) {
	result, err := r.processInternalRaftRequestOnce(ctx, req)
	if err != nil {
		return nil, err
	}
	if result.err != nil {
		return nil, result.err
	}
	if startTime, ok := ctx.Value(traceutil.StartTimeKey).(time.Time); ok && result.trace != nil {
		applyStart := result.trace.GetStartTime()
		// The trace object is created in apply. Here reset the start time to trace
		// the raft request time by the difference between the request start time
		// and apply start time
		result.trace.SetStartTime(startTime)
		result.trace.InsertStep(0, applyStart, "process raft request")
		result.trace.LogIfLong(traceThreshold)
	}
	return result.resp, nil
}

func (r *Region) processInternalRaftRequestOnce(ctx context.Context, req pb.RaftInternalRequest) (*applyResult, error) {
	ai := r.getApplied()
	ci := r.getCommitted()
	if ci > ai+maxGapBetweenApplyAndCommitIndex {
		return nil, ErrTooManyRequests
	}

	req.Header = &pb.RaftHeader{
		ID: r.reqIDGen.Next(),
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	id := req.Header.ID
	ch := r.applyW.Register(id)

	cctx, cancel := context.WithTimeout(ctx, r.ReqTimeout())
	defer cancel()

	start := time.Now()
	ech := make(chan error, 1)
	r.tracer.Trace(newProposeTrace(cctx, r.getID(), data, ech))
	if err = <-ech; err != nil {
		r.applyW.Trigger(id, nil)
		return nil, err
	}

	select {
	case x := <-ch:
		return x.(*applyResult), nil
	case <-cctx.Done():
		r.applyW.Trigger(id, nil)
		return nil, r.parseProposeCtxErr(err, start)
	case <-r.stopc:
		return nil, ErrStopped
	}
}

func (r *Region) parseProposeCtxErr(err error, start time.Time) error {
	switch err {
	case context.Canceled:
		return ErrCanceled

	case context.DeadlineExceeded:
		r.leadTimeMu.RLock()
		curLeadElected := r.leadElectedTime
		r.leadTimeMu.RUnlock()
		prevLeadLost := curLeadElected.Add(-2 * r.ElectionDuration())
		if start.After(prevLeadLost) && start.Before(curLeadElected) {
			return ErrTimeoutDueToLeaderFail
		}
		return ErrTimeout

	default:
		return err
	}
}

func (r *Region) applyEntry(entry sm.Entry) {
	index := entry.Index
	r.writeApplyIndex(index)
	if index == r.getCommitted() {
		r.commitW.Trigger(r.id, nil)
	}

	if len(entry.Cmd) == 0 {
		return
	}

	var raftReq pb.RaftInternalRequest
	if err := raftReq.Unmarshal(entry.Cmd); err != nil {
		r.lg.Warn("unmarshal entry cmd", zap.Uint64("index", entry.Index), zap.Error(err))
		return
	}

	var ar *applyResult

	ctx := context.Background()
	id := raftReq.Header.ID
	need := r.applyW.IsRegistered(id)
	if need {
		ar = r.applyBase.Apply(ctx, &raftReq)
	}

	if ar == nil {
		return
	}

	r.applyW.Trigger(id, ar)
}

func (r *Region) Start() {
	go r.heartbeat()
	go r.scheduleCycle()
	go r.run()
}

func (r *Region) run() {
	for {
		select {
		case <-r.stopc:
			return
		}
	}
}

func (r *Region) readApplyIndex() (uint64, error) {
	kv, err := r.get(appliedIndex)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	value := kv.Value[:8]

	applied := binary.LittleEndian.Uint64(value)
	return applied, nil
}

func (r *Region) writeApplyIndex(index uint64) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, index)
	r.put(appliedIndex, data, true)
	r.setApplied(index)
}

func (r *Region) get(key []byte) (*pb.KeyValue, error) {
	tx := r.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	key = bytesutil.PathJoin(r.putPrefix(), key)
	value, err := tx.UnsafeGet(buckets.Key, key)
	if err != nil {
		return nil, err
	}
	kv := &pb.KeyValue{
		Key:   bytes.TrimPrefix(key, r.putPrefix()),
		Value: value,
	}

	return kv, err
}

func (r *Region) getRange(startKey, endKey []byte, limit int64) ([]*pb.KeyValue, error) {
	tx := r.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	startKey = bytesutil.PathJoin(r.putPrefix(), startKey)
	if len(endKey) > 0 {
		endKey = bytesutil.PathJoin(r.putPrefix(), endKey)
	}
	keys, values, err := tx.UnsafeRange(buckets.Key, startKey, endKey, limit)
	if err != nil {
		return nil, err
	}
	kvs := make([]*pb.KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = &pb.KeyValue{
			Key:   bytes.TrimPrefix(key, r.putPrefix()),
			Value: values[i],
		}
	}
	return kvs, nil
}

func (r *Region) put(key, value []byte, isSync bool) error {
	key = bytesutil.PathJoin(r.putPrefix(), key)
	tx := r.be.BatchTx()
	tx.Lock()
	if err := tx.UnsafePut(buckets.Key, key, value); err != nil {
		tx.Unlock()
		return err
	}
	tx.Unlock()
	if isSync {
		return tx.Commit()
	}
	return nil
}

func (r *Region) del(key []byte, isSync bool) error {
	key = bytesutil.PathJoin(r.putPrefix(), key)
	tx := r.be.BatchTx()
	tx.Lock()
	if err := tx.UnsafeDelete(buckets.Key, key); err != nil {
		tx.Unlock()
		return err
	}
	tx.Unlock()
	if isSync {
		return tx.Commit()
	}
	return nil
}

func (r *Region) notifyAboutChange() {
	r.changeCMu.Lock()
	changeClose := r.changeC
	r.changeC = make(chan struct{})
	r.changeCMu.Unlock()
	close(changeClose)
}

func (r *Region) changeNotify() <-chan struct{} {
	r.changeCMu.RLock()
	defer r.changeCMu.RUnlock()
	return r.changeC
}

func (r *Region) notifyAboutReady() {
	r.readyCMu.Lock()
	readyClose := r.readyC
	r.readyC = make(chan struct{})
	r.readyCMu.Unlock()
	close(readyClose)
}

func (r *Region) ReadyNotify() <-chan struct{} {
	r.readyCMu.RLock()
	defer r.readyCMu.RUnlock()
	return r.readyC
}

func (r *Region) putPrefix() []byte {
	sb := []byte(fmt.Sprintf("%d", r.id))
	return sb
}

func (r *Region) updateInfo(info *pb.Region) {
	r.rimu.Lock()
	defer r.rimu.Unlock()
	r.regionInfo = info
}

func (r *Region) updateConfig(config RegionConfig) {
	r.RegionConfig = config
}

func (r *Region) getInfo() *pb.Region {
	r.rimu.RLock()
	defer r.rimu.RUnlock()
	return r.regionInfo
}

func (r *Region) getID() uint64 {
	return r.id
}

func (r *Region) getMember() uint64 {
	return r.memberId
}

func (r *Region) setApplied(applied uint64) {
	atomic.StoreUint64(&r.applied, applied)
}

func (r *Region) getApplied() uint64 {
	return atomic.LoadUint64(&r.applied)
}

func (r *Region) setCommitted(committed uint64) {
	atomic.StoreUint64(&r.committed, committed)
}

func (r *Region) getCommitted() uint64 {
	return atomic.LoadUint64(&r.committed)
}

func (r *Region) setTerm(term uint64) {
	atomic.StoreUint64(&r.term, term)
}

func (r *Region) getTerm() uint64 {
	return atomic.LoadUint64(&r.term)
}

func (r *Region) setLeader(leader uint64, newLead bool) {
	atomic.StoreUint64(&r.leader, leader)
	if newLead && r.isLeader() {
		t := time.Now()
		r.leadTimeMu.Lock()
		r.leadElectedTime = t
		r.leadTimeMu.Unlock()
	}
}

func (r *Region) getLeader() uint64 {
	return atomic.LoadUint64(&r.leader)
}

func (r *Region) isLeader() bool {
	lead := r.getLeader()
	return lead != 0 && lead == r.getMember()
}
