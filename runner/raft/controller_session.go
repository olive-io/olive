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

package raft

import (
	"context"

	pb "github.com/olive-io/olive/api/olivepb"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
)

type chunk interface {
	unwrap() (context.Context, any)
	write(msg any)
	failure(err error)
}

type chunkImpl struct {
	ctx   context.Context
	value any
	next  uint64
	w     wait.Wait
}

func newChunk(ctx context.Context, value any, next uint64, w wait.Wait) *chunkImpl {
	return &chunkImpl{
		ctx:   ctx,
		value: value,
		next:  next,
		w:     w,
	}
}

func (c *chunkImpl) unwrap() (context.Context, any) {
	return c.ctx, c.value
}

func (c *chunkImpl) write(msg any) {
	c.w.Trigger(c.next, msg)
}

func (c *chunkImpl) failure(err error) {
	c.w.Trigger(c.next, err)
}

type Client struct {
	ch    chan chunk
	reqId *idutil.Generator
	w     wait.Wait
}

func (c *Controller) NewClient() *Client {
	cc := &Client{
		ch:    c.reqCh,
		reqId: c.reqId,
		w:     c.reqW,
	}

	return cc
}

func (c *Client) newCall() (uint64, <-chan any) {
	next := c.reqId.Next()
	out := c.w.Register(next)
	return next, out
}

type regionStatC struct {
	*chunkImpl
}

func (c *Client) regionStat(ctx context.Context, rs *pb.RegionStat) error {
	id, result := c.newCall()
	impl := newChunk(ctx, rs, id, c.w)
	chk := &regionStatC{
		chunkImpl: impl,
	}
	c.ch <- chk

	_, err := response(ctx, result)
	return err
}

type raftReadC struct {
	*chunkImpl
}

type raftQuery struct {
	region uint64
	query  []byte
}

func (c *Client) Read(ctx context.Context, region uint64, query []byte) (any, error) {
	value := &raftQuery{region: region, query: query}
	id, result := c.newCall()
	impl := newChunk(ctx, value, id, c.w)
	chk := &raftReadC{
		chunkImpl: impl,
	}
	c.ch <- chk

	out, err := response(ctx, result)
	return out, err
}

type raftProposeC struct {
	*chunkImpl
}

type raftPropose struct {
	region uint64
	cmd    []byte
}

func (c *Client) Propose(ctx context.Context, region uint64, cmd []byte) (any, error) {
	value := &raftPropose{region: region, cmd: cmd}
	id, result := c.newCall()
	impl := newChunk(ctx, value, id, c.w)
	chk := &raftProposeC{
		chunkImpl: impl,
	}
	c.ch <- chk

	out, err := response(ctx, result)
	return out, err
}

func response(ctx context.Context, result <-chan any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-result:
		switch t := out.(type) {
		case error:
			return nil, t
		default:
			return out, nil
		}
	}
}
