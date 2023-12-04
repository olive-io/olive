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
	"net/url"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/gofrs/flock"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
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
	tx.Unlock()

	r.pr, err = r.register()
	if err != nil {
		return err
	}

	r.controller, err = r.startRaftController()
	if err != nil {
		return err
	}

	go r.run()
	return nil
}

func (r *Runner) startRaftController() (*raft.Controller, error) {
	listenPeerURL, err := url.Parse(r.ListenPeerURL)
	if err != nil {
		return nil, err
	}
	raftAddr := listenPeerURL.Host

	cc := raft.Config{
		DataDir:            r.RegionDir(),
		RaftAddress:        raftAddr,
		HeartbeatMs:        r.HeartbeatMs,
		RaftRTTMillisecond: r.RaftRTTMillisecond,
		Logger:             r.Logger,
	}

	controller, err := raft.NewController(cc, r.be, r.pr)
	if err != nil {
		return nil, err
	}

	r.Logger.Info("start raft container")
	if err = controller.Start(r.StoppingNotify()); err != nil {
		return nil, err
	}

	ctx := r.ctx
	regionPrefix := runtime.DefaultRunnerRegion
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := r.oct.Get(ctx, regionPrefix, options...)
	if err != nil {
		return nil, errors.Wrap(err, "fetch regions from olive-meta")
	}

	regions := make([]*pb.Region, rsp.Count)
	for i, kv := range rsp.Kvs {
		region := new(pb.Region)
		err = region.Unmarshal(kv.Value)
		if err != nil {
			continue
		}
		regions[i] = region
	}
	for _, region := range regions {
		if err = controller.SyncRegion(ctx, region); err != nil {
			return nil, errors.Wrap(err, "sync region")
		}
	}

	return controller, nil
}

func (r *Runner) watching() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	wopts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}
	wch := r.oct.Watch(ctx, runtime.DefaultRunnerPrefix, wopts...)
	for {
		select {
		case <-r.stopping:
			return
		case resp := <-wch:
			for _, event := range resp.Events {
				r.processEvent(r.ctx, event)
			}
		}
	}
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
	switch {
	case strings.HasPrefix(key, runtime.DefaultRunnerDefinitions):
		r.processBpmnDefinition(ctx, event)
	case strings.HasPrefix(key, runtime.DefaultRunnerRegion):
		r.processRegion(ctx, event)
	}
}

func (r *Runner) processBpmnDefinition(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	definition := new(pb.Definition)
	if err := definition.Unmarshal(kv.Value); err != nil {
		r.Logger.Error("unmarshal definition data", zap.Error(err))
		return
	}
}

func (r *Runner) processRegion(ctx context.Context, event *clientv3.Event) {
	kv := event.Kv
	if event.Type == clientv3.EventTypeDelete {
		return
	}

	region := new(pb.Region)
	err := region.Unmarshal(kv.Value)
	if err != nil {
		r.Logger.Error("unmarshal region data", zap.Error(err))
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
