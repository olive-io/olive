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

package idutil

import (
	"context"
	"encoding/binary"
	"sync/atomic"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Generator struct {
	key    string
	client *clientv3.Client
	lg     *zap.Logger
	id     uint64
}

func NewGenerator(key string, client *clientv3.Client) (*Generator, error) {
	id, err := get(context.TODO(), client, key)
	if err != nil {
		return nil, err
	}
	g := &Generator{
		key:    key,
		client: client,
		lg:     client.GetLogger(),
		id:     id,
	}

	return g, nil
}

func (g *Generator) Next(ctx context.Context) uint64 {
	next := atomic.AddUint64(&g.id, 1)
	err := set(ctx, g.client, g.key, next)
	if err != nil {
		g.lg.Error("generate next id", zap.Error(err))
	}
	return next
}

func set(ctx context.Context, client *clientv3.Client, key string, id uint64) error {
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, id)
	_, err := client.Put(ctx, key, string(val))
	return err
}

func get(ctx context.Context, client *clientv3.Client, key string) (uint64, error) {
	rsp, err := client.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	if len(rsp.Kvs) == 0 {
		return 0, nil
	}
	val := rsp.Kvs[0].Value[0:8]
	id := binary.LittleEndian.Uint64(val)
	return id, nil
}
