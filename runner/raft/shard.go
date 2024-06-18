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
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/traceutil"
	v3wait "go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"

	"github.com/olive-io/olive/pkg/bytesutil"
	"github.com/olive-io/olive/pkg/proxy"
	"github.com/olive-io/olive/pkg/queue"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
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

type ProcessInfo struct {
	Stat *corev1.ProcessStat
}

func NewProcessInfo(stat *corev1.ProcessStat) *ProcessInfo {
	return &ProcessInfo{Stat: stat}
}

func (pi *ProcessInfo) UID() string {
	return pi.Stat.Name
}

func (pi *ProcessInfo) Score() int64 {
	return 1
}

type IShardRaftKV interface {
	Range(ctx context.Context, r *pb.ShardRangeRequest) (*pb.ShardRangeResponse, error)
	Put(ctx context.Context, r *pb.ShardPutRequest) (*pb.ShardPutResponse, error)
	Delete(ctx context.Context, r *pb.ShardDeleteRequest) (*pb.ShardDeleteResponse, error)
}

type Shard struct {
	lg  *zap.Logger
	cfg ShardConfig

	id       uint64
	memberId uint64

	applyW   v3wait.Wait
	commitW  v3wait.Wait
	openWait v3wait.Wait

	tracer tracing.ITracer
	proxy  proxy.IProxy

	reqIDGen *idutil.Generator

	be backend.IBackend

	metric *regionMetrics

	processQ *queue.SyncPriorityQueue

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

func (c *Controller) initDiskStateMachine(shardId, nodeId uint64) sm.IOnDiskStateMachine {
	reqIDGen := idutil.NewGenerator(uint16(nodeId), time.Now())
	processQ := queue.NewSync()

	tracer := c.tracer
	region := &Shard{
		id:       shardId,
		memberId: nodeId,
		lg:       c.cfg.Logger,
		openWait: c.regionW,
		tracer:   tracer,
		proxy:    c.proxy,
		applyW:   v3wait.New(),
		commitW:  v3wait.New(),
		reqIDGen: reqIDGen,
		be:       c.be,
		processQ: processQ,
		changeC:  make(chan struct{}),
		readyC:   make(chan struct{}),
		stopc:    make(<-chan struct{}, 1),
	}

	applyBase := region.newApplier()
	region.applyBase = applyBase

	c.setShard(region)

	return region
}

func (s *Shard) Range(ctx context.Context, req *pb.ShardRangeRequest) (*pb.ShardRangeResponse, error) {
	trace := traceutil.New("range",
		s.lg,
		traceutil.Field{Key: "range_begin", Value: string(req.Key)},
		traceutil.Field{Key: "range_end", Value: string(req.RangeEnd)},
	)
	ctx = context.WithValue(ctx, traceutil.TraceKey, trace)

	var resp *pb.ShardRangeResponse
	var err error
	defer func(start time.Time) {
		warnOfExpensiveReadOnlyRangeRequest(s.lg, s.metric.slowApplies, s.cfg.WarningApplyDuration, start, req, resp, err)
		if resp != nil {
			trace.AddField(
				traceutil.Field{Key: "response_count", Value: len(resp.Kvs)},
			)
		}
		trace.LogIfLong(traceThreshold)
	}(time.Now())

	result, err := s.raftQuery(ctx, &pb.RaftInternalRequest{Range: req})
	if err != nil {
		return nil, err
	}
	resp = result.(*pb.ShardRangeResponse)
	return resp, err
}

func (s *Shard) Put(ctx context.Context, req *pb.ShardPutRequest) (*pb.ShardPutResponse, error) {
	ctx = context.WithValue(ctx, traceutil.StartTimeKey, time.Now())
	resp, err := s.raftRequestOnce(ctx, &pb.RaftInternalRequest{Put: req})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ShardPutResponse), nil
}

func (s *Shard) Delete(ctx context.Context, req *pb.ShardDeleteRequest) (*pb.ShardDeleteResponse, error) {
	resp, err := s.raftRequestOnce(ctx, &pb.RaftInternalRequest{Delete: req})
	if err != nil {
		return nil, err
	}
	return resp.(*pb.ShardDeleteResponse), nil
}

func (s *Shard) initial(stopc <-chan struct{}) (uint64, error) {
	s.stopc = stopc
	applyIndex, err := s.readApplyIndex()
	if err != nil {
		s.openWait.Trigger(s.id, err)
		return 0, err
	}

	s.metric, err = newShardMetrics(s.id, s.memberId)
	if err != nil {
		s.openWait.Trigger(s.id, err)
		return 0, err
	}

	s.setApplied(applyIndex)
	s.openWait.Trigger(s.id, s)

	dm := make(map[string]struct{})
	kvs, _ := s.getRange(definitionPrefix, getPrefix(definitionPrefix), 0)
	for _, kv := range kvs {
		definitionId := path.Dir(string(kv.Key))
		if _, ok := dm[definitionId]; !ok {
			dm[definitionId] = struct{}{}
		}
	}
	s.metric.definition.Set(float64(len(dm)))

	kvs, _ = s.getRange(processPrefix, getPrefix(processPrefix), 0)
	for _, kv := range kvs {
		ps := new(corev1.ProcessStat)
		err = ps.Unmarshal(kv.Value)
		if err != nil {
			s.lg.Error("unmarshal process stat", zap.Error(err))
			_ = s.del(kv.Key, true)
			continue
		}

		if ps.Status.FlowNodes == nil {
			ps.Status.FlowNodes = []*corev1.FlowNodeStat{}
		}
		s.processQ.Push(NewProcessInfo(ps))
	}

	return applyIndex, nil
}

func (s *Shard) waitUtilLeader() bool {
	for {
		if s.isLeader() {
			return true
		}

		select {
		case <-s.stopc:
			return false
		case <-s.changeNotify():
		}
	}
}

func (s *Shard) waitLeader(ctx context.Context) (bool, error) {
	for {
		if s.isLeader() {
			return true, nil
		}
		if s.getLeader() != 0 {
			return false, nil
		}

		select {
		case <-ctx.Done():
			return false, context.Canceled
		case <-s.stopc:
			return false, ErrStopped
		case <-s.changeNotify():
		}
	}
}

func (s *Shard) raftQuery(ctx context.Context, req *pb.RaftInternalRequest) (proto.Message, error) {
	trace := newReadTrace(ctx, s.getID(), req)
	defer trace.Close()
	s.tracer.Trace(trace)

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

func (s *Shard) raftRequestOnce(ctx context.Context, req *pb.RaftInternalRequest) (proto.Message, error) {
	result, err := s.processInternalRaftRequestOnce(ctx, req)
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

func (s *Shard) processInternalRaftRequestOnce(ctx context.Context, req *pb.RaftInternalRequest) (*applyResult, error) {
	ai := s.getApplied()
	ci := s.getCommitted()
	if ci > ai+maxGapBetweenApplyAndCommitIndex {
		return nil, ErrTooManyRequests
	}

	req.Header = &pb.RaftHeader{
		ID: s.reqIDGen.Next(),
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	id := req.Header.ID
	ch := s.applyW.Register(id)

	cctx, cancel := context.WithTimeout(ctx, s.cfg.ReqTimeout())
	defer cancel()

	start := time.Now()
	ech := make(chan error, 1)
	s.tracer.Trace(newProposeTrace(cctx, s.getID(), data, ech))
	if err = <-ech; err != nil {
		s.applyW.Trigger(id, nil)
		return nil, err
	}

	select {
	case x := <-ch:
		return x.(*applyResult), nil
	case <-cctx.Done():
		s.applyW.Trigger(id, nil)
		return nil, s.parseProposeCtxErr(err, start)
	case <-s.stopc:
		return nil, ErrStopped
	}
}

func (s *Shard) parseProposeCtxErr(err error, start time.Time) error {
	switch err {
	case context.Canceled:
		return ErrCanceled

	case context.DeadlineExceeded:
		s.leadTimeMu.RLock()
		curLeadElected := s.leadElectedTime
		s.leadTimeMu.RUnlock()
		prevLeadLost := curLeadElected.Add(-2 * s.cfg.ElectionDuration())
		if start.After(prevLeadLost) && start.Before(curLeadElected) {
			return ErrTimeoutDueToLeaderFail
		}
		return ErrTimeout

	default:
		return err
	}
}

func (s *Shard) applyEntry(entry sm.Entry) {
	index := entry.Index
	s.writeApplyIndex(index)
	if index == s.getCommitted() {
		s.commitW.Trigger(s.id, nil)
	}

	if len(entry.Cmd) == 0 {
		return
	}

	raftReq := &pb.RaftInternalRequest{}
	if err := raftReq.Unmarshal(entry.Cmd); err != nil {
		s.lg.Warn("unmarshal entry cmd", zap.Uint64("index", entry.Index), zap.Error(err))
		return
	}

	var ar *applyResult

	ctx := context.Background()
	id := raftReq.Header.ID
	need := s.applyW.IsRegistered(id)
	if need {
		ar = s.applyBase.Apply(ctx, raftReq)
	}

	if ar == nil {
		return
	}

	s.applyW.Trigger(id, ar)
}

func (s *Shard) Start() {
	go s.heartbeat()
	go s.scheduleCycle()
	go s.run()
}

func (s *Shard) run() {
	for {
		select {
		case <-s.stopc:
			return
		}
	}
}

func (s *Shard) readApplyIndex() (uint64, error) {
	kv, err := s.get(appliedIndex)
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

func (s *Shard) writeApplyIndex(index uint64) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, index)
	s.put(appliedIndex, data, true)
	s.setApplied(index)
}

func (s *Shard) get(key []byte) (*pb.KeyValue, error) {
	tx := s.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	key = bytesutil.PathJoin(s.putPrefix(), key)
	value, err := tx.UnsafeGet(buckets.Key, key)
	if err != nil {
		return nil, err
	}
	kv := &pb.KeyValue{
		Key:   bytes.TrimPrefix(key, s.putPrefix()),
		Value: value,
	}

	return kv, err
}

func (s *Shard) getRange(startKey, endKey []byte, limit int64) ([]*pb.KeyValue, error) {
	tx := s.be.ReadTx()
	tx.RLock()
	defer tx.RUnlock()

	startKey = bytesutil.PathJoin(s.putPrefix(), startKey)
	if len(endKey) > 0 {
		endKey = bytesutil.PathJoin(s.putPrefix(), endKey)
	}
	keys, values, err := tx.UnsafeRange(buckets.Key, startKey, endKey, limit)
	if err != nil {
		return nil, err
	}
	kvs := make([]*pb.KeyValue, len(keys))
	for i, key := range keys {
		kvs[i] = &pb.KeyValue{
			Key:   bytes.TrimPrefix(key, s.putPrefix()),
			Value: values[i],
		}
	}
	return kvs, nil
}

func (s *Shard) put(key, value []byte, isSync bool) error {
	key = bytesutil.PathJoin(s.putPrefix(), key)
	tx := s.be.BatchTx()
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

func (s *Shard) del(key []byte, isSync bool) error {
	key = bytesutil.PathJoin(s.putPrefix(), key)
	tx := s.be.BatchTx()
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

func (s *Shard) notifyAboutChange() {
	s.changeCMu.Lock()
	changeClose := s.changeC
	s.changeC = make(chan struct{})
	s.changeCMu.Unlock()
	close(changeClose)
}

func (s *Shard) changeNotify() <-chan struct{} {
	s.changeCMu.RLock()
	defer s.changeCMu.RUnlock()
	return s.changeC
}

func (s *Shard) notifyAboutReady() {
	s.readyCMu.Lock()
	readyClose := s.readyC
	s.readyC = make(chan struct{})
	s.readyCMu.Unlock()
	close(readyClose)
}

func (s *Shard) ReadyNotify() <-chan struct{} {
	s.readyCMu.RLock()
	defer s.readyCMu.RUnlock()
	return s.readyC
}

func (s *Shard) putPrefix() []byte {
	sb := []byte(fmt.Sprintf("%d", s.id))
	return sb
}

func (s *Shard) updateConfig(config ShardConfig) {
	s.cfg = config
}

func (s *Shard) getID() uint64 {
	return s.id
}

func (s *Shard) getMember() uint64 {
	return s.memberId
}

func (s *Shard) setApplied(applied uint64) {
	atomic.StoreUint64(&s.applied, applied)
}

func (s *Shard) getApplied() uint64 {
	return atomic.LoadUint64(&s.applied)
}

func (s *Shard) setCommitted(committed uint64) {
	atomic.StoreUint64(&s.committed, committed)
}

func (s *Shard) getCommitted() uint64 {
	return atomic.LoadUint64(&s.committed)
}

func (s *Shard) setTerm(term uint64) {
	atomic.StoreUint64(&s.term, term)
}

func (s *Shard) getTerm() uint64 {
	return atomic.LoadUint64(&s.term)
}

func (s *Shard) setLeader(leader uint64, newLead bool) {
	atomic.StoreUint64(&s.leader, leader)
	if newLead && s.isLeader() {
		t := time.Now()
		s.leadTimeMu.Lock()
		s.leadElectedTime = t
		s.leadTimeMu.Unlock()
	}
}

func (s *Shard) getLeader() uint64 {
	return atomic.LoadUint64(&s.leader)
}

func (s *Shard) isLeader() bool {
	lead := s.getLeader()
	return lead != 0 && lead == s.getMember()
}
