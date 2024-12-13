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

type Generator struct {
	ctx    context.Context
	key    string
	client *clientv3.Client
	lg     *zap.Logger
	id     uint64
	rev    int64
}

func NewGenerator(ctx context.Context, key string, client *clientv3.Client) (*Generator, error) {
	g := &Generator{
		ctx:    ctx,
		key:    key,
		client: client,
		lg:     client.GetLogger(),
	}
	err := g.load()
	if err != nil {
		return nil, err
	}
	return g, nil
}

// Current returns the current id number
func (g *Generator) Current() uint64 {
	return atomic.LoadUint64(&g.id)
}

func (g *Generator) Next() uint64 {
	next := atomic.AddUint64(&g.id, 1)
	err := g.set(next)
	if err != nil {
		g.lg.Error("generate next id", zap.Error(err))
	}
	return next
}

func (g *Generator) Start() {
	go g.watching()
}

func (g *Generator) watching() {
	wch := g.client.Watch(g.ctx, g.key)
	for {
		select {
		case <-g.ctx.Done():
			return
		case ch := <-wch:
			if ch.IsProgressNotify() || ch.Canceled {
				break
			}

			for _, event := range ch.Events {
				if event.Kv.ModRevision > atomic.LoadInt64(&g.rev) {
					val := event.Kv.Value[0:8]
					g.id = binary.LittleEndian.Uint64(val)
					atomic.StoreInt64(&g.rev, event.Kv.ModRevision)
				}
			}
		}
	}
}

func (g *Generator) set(id uint64) error {
	ctx := g.ctx
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, id)
	session, err := concurrency.NewSession(g.client)
	if err != nil {
		return err
	}
	defer session.Close()
	mu := concurrency.NewMutex(session, path.Join(g.key, "lock"))
	if err = mu.Lock(ctx); err != nil {
		return err
	}
	defer mu.Unlock(ctx)
	rsp, err := g.client.Put(ctx, g.key, string(val))
	if err != nil {
		return err
	}
	atomic.StoreInt64(&g.rev, rsp.Header.Revision)

	return nil
}

func (g *Generator) load() error {
	rsp, err := g.client.Get(g.ctx, g.key)
	if err != nil && !errors.Is(err, rpctypes.ErrKeyNotFound) {
		return err
	}
	if len(rsp.Kvs) == 0 {
		return nil
	}
	val := rsp.Kvs[0].Value[0:8]
	g.id = binary.LittleEndian.Uint64(val)
	atomic.StoreInt64(&g.rev, rsp.Header.Revision)
	return nil
}
