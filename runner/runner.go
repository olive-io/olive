// Copyright 2023 Lack (xingyys@gmail.com).
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
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/runner/backend"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Runner struct {
	Config

	ctx    context.Context
	cancel context.CancelFunc

	ec *clientv3.Client

	be backend.IBackend

	stopping chan struct{}
	done     chan struct{}
	stop     chan struct{}

	raftGroup *MultiRaftGroup

	wgMu sync.RWMutex
	wg   sync.WaitGroup
}

func NewRunner(cfg Config) (*Runner, error) {
	ec, err := clientv3.New(*cfg.Config)
	if err != nil {
		return nil, err
	}

	be := newBackend(&cfg)
	ctx, cancel := context.WithCancel(context.Background())

	runner := &Runner{
		Config: cfg,

		ctx:    ctx,
		cancel: cancel,

		ec: ec,
		be: be,

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

	r.GoAttach(r.heartbeat)
	r.GoAttach(r.watching)

	return nil
}

func (r *Runner) start() error {
	var err error
	r.raftGroup, err = r.newMultiRaftGroup()
	if err != nil {
		return err
	}

	if err = r.registry(); err != nil {
		return err
	}

	go r.run()
	return nil
}

func (r *Runner) registry() error {
	conn := r.ec.ActiveConnection()
	rc := pb.NewRunnerRPCClient(conn)

	pr := &pb.Runner{
		AdvertiseListen: r.AdvertiseListen,
		PeerListen:      r.PeerListen,
		HeartbeatMs:     r.HeartbeatMs,
	}
	req := &pb.RegistryRunnerRequest{Runner: pr}

	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()
	rsp, err := rc.RegistryRunner(ctx, req)
	if err != nil {
		return err
	}
	pr.Id = rsp.Id

	return nil
}

func (r *Runner) heartbeat() {
	ticker := time.NewTicker(time.Duration(r.HeartbeatMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopping:
			return
		case <-ticker.C:

		}
	}
}

func (r *Runner) watching() {
	ctx, cancel := context.WithCancel(r.ctx)
	defer cancel()

	wopts := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}
	wch := r.ec.Watch(ctx, "/", wopts...)
	for {
		select {
		case <-r.stopping:
			return
		case resp := <-wch:
			for _, event := range resp.Events {
				switch {
				case event.IsModify():
				}
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
	defer func() {
		r.wgMu.Lock() // block concurrent waitgroup adds in GoAttach while stopping
		close(r.stopping)
		r.wgMu.Unlock()
		r.cancel()

		// wait for goroutines before closing raft so wal stays open
		r.wg.Wait()

		close(r.done)
	}()

	for {
		select {
		case <-r.stop:
			return
		}
	}
}
