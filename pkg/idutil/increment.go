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

package idutil

import (
	"context"
	"encoding/binary"
	"errors"
	"path"
	"sync/atomic"

	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

// Incrementer defines an auto-increment id generator
type Incrementer struct {
	key    string
	client *clientv3.Client
	lg     *zap.Logger
	id     uint64
	rev    int64

	stopCh <-chan struct{}
}

// NewIncrementer returns an Incrementer
func NewIncrementer(key string, client *clientv3.Client, stopCh <-chan struct{}) (*Incrementer, error) {
	in := &Incrementer{
		key:    key,
		client: client,
		lg:     client.GetLogger(),
		stopCh: stopCh,
	}
	err := in.load(context.TODO())
	if err != nil {
		return nil, err
	}
	return in, nil
}

// Current returns the current id number
func (in *Incrementer) Current() uint64 {
	return atomic.LoadUint64(&in.id)
}

// Next returns next identify
func (in *Incrementer) Next(ctx context.Context) uint64 {
	next := atomic.AddUint64(&in.id, 1)
	err := in.set(ctx, next)
	if err != nil {
		in.lg.Error("generate next id", zap.Error(err))
	}
	return next
}

func (in *Incrementer) Start() {
	go in.watching()
}

func (in *Incrementer) watching() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wch := in.client.Watch(ctx, in.key, clientv3.WithSerializable())
	for {
		select {
		case <-in.stopCh:
			return
		case ch := <-wch:
			if ch.IsProgressNotify() || ch.Canceled {
				break
			}

			for _, event := range ch.Events {
				if event.Kv.ModRevision > atomic.LoadInt64(&in.rev) {
					val := event.Kv.Value[0:8]
					in.id = binary.LittleEndian.Uint64(val)
					atomic.StoreInt64(&in.rev, event.Kv.ModRevision)
				}
			}
		}
	}
}

func (in *Incrementer) set(ctx context.Context, id uint64) error {
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, id)
	session, err := concurrency.NewSession(in.client)
	if err != nil {
		return err
	}
	defer session.Close()
	mu := concurrency.NewMutex(session, path.Join(in.key, "lock"))
	if err = mu.Lock(ctx); err != nil {
		return err
	}
	defer mu.Unlock(ctx)
	rsp, err := in.client.Put(ctx, in.key, string(val))
	if err != nil {
		return err
	}
	atomic.StoreInt64(&in.rev, rsp.Header.Revision)

	return nil
}

func (in *Incrementer) load(ctx context.Context) error {
	rsp, err := in.client.Get(ctx, in.key, clientv3.WithSerializable())
	if err != nil && !errors.Is(err, rpctypes.ErrKeyNotFound) {
		return err
	}
	if len(rsp.Kvs) == 0 {
		return nil
	}
	val := rsp.Kvs[0].Value[0:8]
	in.id = binary.LittleEndian.Uint64(val)
	atomic.StoreInt64(&in.rev, rsp.Header.Revision)
	return nil
}
