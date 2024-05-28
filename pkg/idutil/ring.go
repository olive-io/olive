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

package idutil

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"sync"
	"sync/atomic"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

type store struct {
	Index uint64   `json:"index"`
	Frees []uint64 `json:"frees"`
}

type Ring struct {
	key    string
	client *clientv3.Client
	lg     *zap.Logger

	mu  sync.RWMutex
	s   store
	rev int64

	stopCh <-chan struct{}
}

func NewRing(key string, client *clientv3.Client, stopCh <-chan struct{}) (*Ring, error) {
	r := &Ring{
		key:    key,
		client: client,
		lg:     client.GetLogger(),
		stopCh: stopCh,
	}
	err := r.load(context.TODO())
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Current returns the current id number
func (r *Ring) Current() uint64 {
	return atomic.LoadUint64(&r.s.Index)
}

// Next returns next identify
func (r *Ring) Next(ctx context.Context) uint64 {
	var next uint64
	if r.frees() > 0 {
		r.mu.Lock()
		next = r.s.Frees[0]
		r.s.Frees = r.s.Frees[1:]
		r.mu.Unlock()
	} else {
		r.mu.Lock()
		r.s.Index += 1
		next = r.s.Index
		r.mu.RUnlock()
	}
	err := r.set(ctx)
	if err != nil {
		r.lg.Error("generate next id", zap.Error(err))
	}
	return next
}

func (r *Ring) Recycle(ctx context.Context, id uint64) {
	r.mu.Lock()
	r.s.Frees = append(r.s.Frees, id)
	r.mu.Unlock()
	err := r.set(ctx)
	if err != nil {
		r.lg.Error("recycle id", zap.Error(err))
	}
}

func (r *Ring) frees() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.s.Frees)
}

func (r *Ring) Start() {
	go r.watching()
}

func (r *Ring) watching() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wch := r.client.Watch(ctx, r.key, clientv3.WithSerializable())
	for {
		select {
		case <-r.stopCh:
			return
		case ch := <-wch:
			if ch.IsProgressNotify() || ch.Canceled {
				break
			}

			for _, event := range ch.Events {
				if event.Kv.ModRevision > atomic.LoadInt64(&r.rev) {
					var s store
					if err := json.Unmarshal(event.Kv.Value, &s); err == nil {
						r.s = s
					}
					atomic.StoreInt64(&r.rev, event.Kv.ModRevision)
				}
			}
		}
	}
}

func (r *Ring) set(ctx context.Context) error {
	r.mu.Lock()
	data, _ := json.Marshal(r.s)
	r.mu.Unlock()

	session, err := concurrency.NewSession(r.client)
	if err != nil {
		return err
	}
	defer session.Close()
	mu := concurrency.NewMutex(session, path.Join(r.key, "lock"))
	if err = mu.Lock(ctx); err != nil {
		return err
	}
	defer mu.Unlock(ctx)
	rsp, err := r.client.Put(ctx, r.key, string(data))
	if err != nil {
		return err
	}
	atomic.StoreInt64(&r.rev, rsp.Header.Revision)

	return nil
}

func (r *Ring) load(ctx context.Context) error {
	rsp, err := r.client.Get(ctx, r.key, clientv3.WithSerializable())
	if err != nil && !errors.Is(err, rpctypes.ErrKeyNotFound) {
		return err
	}
	if len(rsp.Kvs) == 0 {
		return nil
	}
	return json.Unmarshal(rsp.Kvs[0].Value, &r.s)
}
