/*
Copyright 2025 The olive Authors

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

package delegate

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
)

// This is an application-wide global ID allocator.  Unfortunately we need
// to have unique IDs globally to permit certain things to work
// correctly.
type idAllocator struct {
	used map[uint64]struct{}
	next uint64
	lock sync.Mutex
}

func newIDAllocator() *idAllocator {
	b := make([]byte, 8)
	// The following could in theory fail, but in that case
	// we will wind up with IDs starting at zero.  It should
	// not happen unless the platform can't get good entropy.
	_, _ = rand.Read(b)
	used := make(map[uint64]struct{})
	next := binary.BigEndian.Uint64(b)
	alloc := &idAllocator{
		used: used,
		next: next,
	}
	return alloc
}

func (alloc *idAllocator) Get() uint64 {
	alloc.lock.Lock()
	defer alloc.lock.Unlock()
	for {
		id := alloc.next & 0x7fffffff
		alloc.next++
		if id == 0 {
			continue
		}
		if _, ok := alloc.used[id]; ok {
			continue
		}
		alloc.used[id] = struct{}{}
		return id
	}
}

func (alloc *idAllocator) Free(id uint64) {
	alloc.lock.Lock()
	if _, ok := alloc.used[id]; ok {
		delete(alloc.used, id)
	}
	alloc.lock.Unlock()
}

type Delegate interface {
	Call(ctx context.Context, req *pb.CallTaskRequest) (*pb.CallTaskResponse, error)
}

type StreamPipe struct {
	ctx      context.Context
	receiver <-chan *pb.CallTaskResponse

	allocator *idAllocator

	tmu      sync.RWMutex
	transfer map[uint64]chan *pb.CallTaskResponse
}

func NewStreamPipe(ctx context.Context, receiver <-chan *pb.CallTaskResponse) *StreamPipe {
	pipe := &StreamPipe{
		ctx:       ctx,
		receiver:  receiver,
		allocator: newIDAllocator(),
		tmu:       sync.RWMutex{},
		transfer:  map[uint64]chan *pb.CallTaskResponse{},
	}
	return pipe
}

func (s *StreamPipe) Call(ctx context.Context, req *pb.CallTaskRequest) (*pb.CallTaskResponse, error) {
	uid := s.allocator.Get()
	defer s.allocator.Free(uid)

	req.Uid = uid

	ch := make(chan *pb.CallTaskResponse, 1)
	s.tmu.Lock()
	s.transfer[uid] = ch
	s.tmu.Unlock()

	defer func() {
		s.tmu.Lock()
		delete(s.transfer, uid)
		s.tmu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.ctx.Done():
		return nil, fmt.Errorf("stream is closed")
	case rsp := <-ch:
		return rsp, nil
	}
}

func (s *StreamPipe) Start() {
	ctx := s.ctx
	for {
		select {
		case <-ctx.Done():
			return
		case recv := <-s.receiver:
			s.tmu.RLock()
			ch, ok := s.transfer[recv.Uid]
			s.tmu.RUnlock()
			if !ok {
				ch <- recv
			}
		}
	}
}
