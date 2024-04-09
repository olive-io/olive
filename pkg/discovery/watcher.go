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

func newEtcdWatcher(ctx context.Context, r *etcdRegistrar, opts ...WatchOption) (Watcher, error) {
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
