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
	"errors"
	"sync"
	"time"

	pb "github.com/olive-io/olive/api/olivepb"
	"go.etcd.io/etcd/pkg/v3/idutil"
)

type RegionStatWatcher interface {
	Watch(ctx context.Context) RegionStatWatchChan
	close() error
}

type RegionStatWatchChan <-chan pb.RegionStat

type regionStatWatcher struct {
	mu sync.Mutex

	idReq *idutil.Generator
	// streams holds all the active regionStat streams keyed by ctx value.
	streams map[uint64]*regionStatWatchStream
}

func newRegionStatWatcher() *regionStatWatcher {
	return &regionStatWatcher{
		mu:      sync.Mutex{},
		idReq:   idutil.NewGenerator(0, time.Now()),
		streams: make(map[uint64]*regionStatWatchStream),
	}
}

func (w *regionStatWatcher) Trigger(rs *pb.RegionStat) {
	w.mu.Lock()
	for _, stream := range w.streams {
		stream.ch <- *rs
	}
	w.mu.Unlock()
}

func (w *regionStatWatcher) Watch(ctx context.Context) RegionStatWatchChan {
	id := w.idReq.Next()
	ch := make(chan pb.RegionStat, 1)
	ctx, cancel := context.WithCancel(ctx)
	stream := &regionStatWatchStream{
		ctx:    ctx,
		cancel: cancel,
		ch:     ch,
	}

	w.mu.Lock()
	w.streams[id] = stream
	w.mu.Unlock()

	go func() {
		<-ctx.Done()
		close(ch)
		w.mu.Lock()
		delete(w.streams, id)
		w.mu.Unlock()
	}()

	return ch
}

func (w *regionStatWatcher) close() (err error) {
	w.mu.Lock()
	streams := w.streams
	w.streams = nil
	w.mu.Unlock()
	for _, wgs := range streams {
		if werr := wgs.close(); werr != nil {
			err = werr
		}
	}
	// Consider context.Canceled as a successful close
	if errors.Is(err, context.Canceled) {
		err = nil
	}
	return err
}

type regionStatWatchStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan pb.RegionStat
}

func (s *regionStatWatchStream) close() error {
	select {
	case <-s.ctx.Done():
		return context.Canceled
	default:
		s.cancel()
	}

	return nil
}
