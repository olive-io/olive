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

package runner

import (
	"context"
	"encoding/binary"
	"net/url"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/gofrs/flock"
	"github.com/olive-io/bpmn/tracing"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/runtime"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"github.com/olive-io/olive/runner/raft"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Runner struct {
	Config

	ctx    context.Context
	cancel context.CancelFunc

	gLock *flock.Flock

	oct *client.Client

	be backend.IBackend

	controller *raft.Controller
	traces     <-chan tracing.ITrace
	pr         *pb.Runner

	stopping chan struct{}
	done     chan struct{}
	stop     chan struct{}

	wgMu sync.RWMutex
	wg   sync.WaitGroup
}

func NewRunner(cfg Config) (*Runner, error) {
	lg := cfg.Logger

	gLock, err := cfg.LockDataDir()
	if err != nil {
		return nil, err
	}
	lg.Debug("protected directory: " + cfg.DataDir)

	oct, err := client.New(cfg.Config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	be := newBackend(&cfg)

	runner := &Runner{
		Config: cfg,

		ctx:    ctx,
		cancel: cancel,

		gLock: gLock,

		oct: oct,
		be:  be,

		stopping: make(chan struct{}, 1),
		done:     make(chan struct{}, 1),
		stop:     make(chan struct{}, 1),
	}

	return runner, nil
}

func (r *Runner) Start() error {
	if err := r.start(); err != nil {
		return err
	}

	r.GoAttach(r.registry)
	r.GoAttach(r.watching)

	return nil
}

func (r *Runner) start() error {
	var err error

	defer r.be.ForceCommit()
	tx := r.be.BatchTx()
	tx.Lock()
	tx.UnsafeCreateBucket(buckets.Meta)
	tx.UnsafeCreateBucket(buckets.Region)
	tx.UnsafeCreateBucket(buckets.Key)
	tx.Unlock()
	if err = tx.Commit(); err != nil {
		r.Logger.Error("pebble commit", zap.Error(err))
	}

	r.pr, err = r.register()
	if err != nil {
		return err
	}

	r.controller, r.traces, err = r.startRaftController()
	if err != nil {
		return err
	}

	go r.run()
	return nil
}

func (r *Runner) startRaftController() (*raft.Controller, <-chan tracing.ITrace, error) {
	listenPeerURL, err := url.Parse(r.ListenPeerURL)
	if err != nil {
		return nil, nil, err
	}
	raftAddr := listenPeerURL.Host

	cc := raft.NewConfig()
	cc.DataDir = r.RegionDir()
	cc.RaftAddress = raftAddr
	cc.HeartbeatMs = r.HeartbeatMs
	cc.RaftRTTMillisecond = r.RaftRTTMillisecond
	cc.Logger = r.Logger

	discovery, err := dsy.NewDiscovery(r.Logger, r.oct.Client, r.StoppingNotify())
	if err != nil {
		return nil, nil, err
	}

	controller, err := raft.NewController(r.ctx, cc, r.be, discovery, r.pr)
	if err != nil {
		return nil, nil, err
	}

	r.Logger.Info("start raft container")
	if err = controller.Start(r.StoppingNotify()); err != nil {
		return nil, nil, err
	}

	ctx := r.ctx
	prefix := runtime.DefaultRunnerRegion
	rev := r.getRev()

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
		clientv3.WithRev(rev),
	}

	rsp, err := r.oct.Get(ctx, prefix, append(options, clientv3.WithMinModRev(rev+1))...)
	if err != nil {
		return nil, nil, err
	}

	regions := make([]*pb.Region, 0)
	definitions := make([]*pb.Definition, 0)
	processes := make([]*pb.ProcessInstance, 0)
	for _, kv := range rsp.Kvs {
		rev = kv.ModRevision
		key := string(kv.Value)
		switch {
		case strings.HasPrefix(key, runtime.DefaultRunnerRegion):
			region, match, err := parseRegionKv(kv, r.pr.Id)
			if err != nil || !match {
				continue
			}
			regions = append(regions, region)
		case strings.HasPrefix(key, runtime.DefaultRunnerDefinitions):
			definition, match, err := parseDefinitionKv(kv)
			if err != nil || !match {
				continue
			}
			definitions = append(definitions, definition)
		case strings.HasPrefix(key, runtime.DefaultRunnerProcessInstance):
			proc, match, err := parseProcessInstanceKv(kv)
			if err != nil || !match {
				continue
			}
			processes = append(processes, proc)
		}
	}
	for _, region := range regions {
		if err = controller.SyncRegion(ctx, region); err != nil {
			return nil, nil, errors.Wrap(err, "sync region")
		}
	}
	for _, definition := range definitions {
		if err = controller.DeployDefinition(ctx, definition); err != nil {
			r.Logger.Error("deploy definition", zap.Error(err))
		}
	}
	for _, process := range processes {
		if err = controller.ExecuteDefinition(ctx, process); err != nil {
			r.Logger.Error("execute definition", zap.Error(err))
		} else {
			commitProcessInstance(ctx, r.Logger, r.oct, process)
		}
	}

	r.setRev(rev)
	return controller, controller.SubscribeTrace(), nil
}

func (r *Runner) watching() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	rev := r.getRev()
	wopts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithRev(rev + 1),
	}
	wch := r.oct.Watch(ctx, runtime.DefaultRunnerPrefix, wopts...)
	for {
		select {
		case <-r.stopping:
			return
		case resp := <-wch:
			for _, event := range resp.Events {
				if event.Kv.ModRevision > rev {
					rev = event.Kv.ModRevision
					r.setRev(rev)
				}
				r.processEvent(r.ctx, event)
			}
		}
	}
}

func (r *Runner) getRev() int64 {
	key := []byte("cur_rev")
	tx := r.be.ReadTx()
	tx.RLock()
	value, err := tx.UnsafeGet(buckets.Meta, key)
	tx.RUnlock()
	if err != nil || len(value) == 0 {
		return 0
	}
	value = value[:8]
	return int64(binary.LittleEndian.Uint64(value))
}

func (r *Runner) setRev(rev int64) {
	key := []byte("cur_rev")
	value := make([]byte, 8)
	binary.LittleEndian.PutUint64(value, uint64(rev))
	tx := r.be.BatchTx()
	tx.Lock()
	tx.UnsafePut(buckets.Meta, key, value)
	tx.Unlock()
	tx.Commit()
}

// StopNotify returns a channel that receives an empty struct
// when the server is stopped.
func (r *Runner) StopNotify() <-chan struct{} { return r.done }

// StoppingNotify returns a channel that receives an empty struct
// when the server is being stopped.
func (r *Runner) StoppingNotify() <-chan struct{} { return r.stopping }

// GoAttach creates a goroutine on a given function and tracks it using the waitgroup.
// The passed function should interrupt on s.StoppingNotify().
func (r *Runner) GoAttach(f func()) {
	r.wgMu.RLock() // this blocks with ongoing close(s.stopping)
	defer r.wgMu.RUnlock()
	select {
	case <-r.stopping:
		r.Logger.Warn("server has stopped; skipping GoAttach")
		return
	default:
	}

	// now safe to add since waitgroup wait has not started yet
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		f()
	}()
}

func (r *Runner) Stop() {
	select {
	case r.stop <- struct{}{}:
	case <-r.done:
		return
	}
	<-r.done
}

func (r *Runner) run() {
	lg := r.Logger

	defer func() {
		r.wgMu.Lock() // block concurrent waitgroup adds in GoAttach while stopping
		close(r.stopping)
		r.wgMu.Unlock()
		r.cancel()

		// wait for goroutines before closing raft so wal stays open
		r.wg.Wait()
		if err := r.gLock.Unlock(); err != nil {
			lg.Error("released "+r.DataDir, zap.Error(err))
		}

		close(r.done)
	}()

	for {
		select {
		case <-r.stop:
			return
		}
	}
}

func (r *Runner) processEvent(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	key := string(kv.Key)
	if event.Type == clientv3.EventTypeDelete {
		rev := event.Kv.CreateRevision
		options := []clientv3.OpOption{clientv3.WithRev(rev)}
		rsp, err := r.oct.Get(ctx, string(event.Kv.Key), options...)
		if err != nil {
			r.Logger.Error("get key-value")
		}
		if len(rsp.Kvs) > 0 {
			event.Kv = rsp.Kvs[0]
		}
	}
	switch {
	case strings.HasPrefix(key, runtime.DefaultRunnerRegion):
		r.processRegion(ctx, event)
	case strings.HasPrefix(key, runtime.DefaultRunnerDefinitions):
		r.processBpmnDefinition(ctx, event)
	case strings.HasPrefix(key, runtime.DefaultRunnerProcessInstance):
		r.processBpmnProcess(ctx, event)
	}
}

func (r *Runner) processRegion(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	region, match, err := parseRegionKv(kv, r.pr.Id)
	if err != nil {
		r.Logger.Error("parse region data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if event.IsCreate() {
		r.Logger.Info("create region", zap.Stringer("body", region))
		if err = r.controller.CreateRegion(ctx, region); err != nil {
			r.Logger.Error("create region", zap.Error(err))
		}
		return
	}

	if event.IsModify() {
		r.Logger.Info("sync region", zap.Stringer("body", region))
		if err = r.controller.SyncRegion(ctx, region); err != nil {
			r.Logger.Error("sync region", zap.Error(err))
		}
		return
	}
}

func (r *Runner) processBpmnDefinition(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	definition, match, err := parseDefinitionKv(kv)
	if err != nil {
		r.Logger.Error("parse definition data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if err = r.controller.DeployDefinition(ctx, definition); err != nil {
		r.Logger.Error("definition deploy",
			zap.String("id", definition.Id),
			zap.Uint64("version", definition.Version),
			zap.Error(err))
		return
	}
}

func (r *Runner) processBpmnProcess(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	process, match, err := parseProcessInstanceKv(kv)
	if err != nil {
		r.Logger.Error("parse process data", zap.Error(err))
		return
	}
	if !match {
		return
	}

	if err = r.controller.ExecuteDefinition(ctx, process); err != nil {
		r.Logger.Error("execute definition",
			zap.String("id", process.DefinitionId),
			zap.Uint64("version", process.DefinitionVersion),
			zap.Error(err))
		return
	}

	commitProcessInstance(ctx, r.Logger, r.oct, process)
}
