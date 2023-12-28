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

package discovery

import (
	"context"
	"errors"
	"path"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

// Watcher is an interface that returns updates
// about services within the registry.
type Watcher interface {
	// Next is a blocking call
	Next() (*dsypb.Result, error)
	Stop()
}

type etcdWatcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	stop   chan bool
	w      clientv3.WatchChan
	client *clientv3.Client
}

func newEtcdWatcher(ctx context.Context, r *etcdRegistry, opts ...WatchOption) (Watcher, error) {
	var wo WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	ctx, cancel := context.WithCancel(ctx)
	stop := make(chan bool, 1)

	namespace := r.options.Namespace
	if wo.Namespace != "" {
		namespace = wo.Namespace
	}

	prefix := r.options.Prefix
	watchPath := prefix
	if len(wo.Service) > 0 {
		watchPath = servicePath(prefix, namespace, wo.Service) + "/"
	} else {
		watchPath = path.Join(prefix, namespace) + "/"
	}

	wopts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithPrevKV(),
	}

	watcher := &etcdWatcher{
		ctx:    ctx,
		cancel: cancel,
		stop:   stop,
		w:      r.client.Watch(ctx, watchPath, wopts...),
		client: r.client,
	}
	go watcher.run()
	return watcher, nil
}

func (ew *etcdWatcher) run() {
	defer ew.cancel()
	for {
		select {
		case <-ew.stop:
			return
		}
	}
}

func (ew *etcdWatcher) Next() (*dsypb.Result, error) {
	for wresp := range ew.w {
		if wresp.Err() != nil {
			return nil, wresp.Err()
		}
		if wresp.Canceled {
			return nil, errors.New("could not get next, watch is canceled")
		}
		for _, ev := range wresp.Events {
			service := decode(ev.Kv.Value)
			var action string

			switch ev.Type {
			case clientv3.EventTypePut:
				if ev.IsCreate() {
					action = "create"
				} else if ev.IsModify() {
					action = "update"
				}
			case clientv3.EventTypeDelete:
				action = "delete"

				// get service from prevKv
				service = decode(ev.PrevKv.Value)
			}

			if service == nil {
				continue
			}
			return &dsypb.Result{
				Action:    action,
				Service:   service,
				Timestamp: time.Now().Unix(),
			}, nil
		}
	}

	return nil, errors.New("could not get next")
}

func (ew *etcdWatcher) Stop() {
	select {
	case <-ew.stop:
		return
	default:
		close(ew.stop)
	}
}
