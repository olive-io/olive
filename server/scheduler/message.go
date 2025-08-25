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

package scheduler

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

var (
	ErrSchedulerStopped = errors.New("scheduler has been stopped")

	ErrIsClosed = errors.New("watch channel closed")
)

type WatchMessage struct {
	Err      error
	Process  *types.Process
	FlowNode *types.FlowNode
}

type WatchChan struct {
	ctx context.Context

	name string
	lg   *zap.Logger

	ch   chan *WatchMessage
	done chan struct{}

	closed <-chan struct{}
}

func newWatchChan(ctx context.Context, name string, lg *zap.Logger, closed <-chan struct{}) *WatchChan {
	w := &WatchChan{
		ctx:    ctx,
		name:   name,
		lg:     lg,
		ch:     make(chan *WatchMessage, 3),
		done:   make(chan struct{}, 1),
		closed: closed,
	}
	return w
}

func (w *WatchChan) send(msg *WatchMessage) {
	select {
	case <-w.ctx.Done():
	case <-w.done:
	case <-w.closed:
	case w.ch <- msg:
	}
}

func (w *WatchChan) Next() *WatchMessage {
	select {
	case <-w.ctx.Done():
		return &WatchMessage{Err: w.ctx.Err()}
	case <-w.done:
		return &WatchMessage{Err: ErrIsClosed}
	case <-w.closed:
		return &WatchMessage{Err: ErrSchedulerStopped}
	case msg, ok := <-w.ch:
		if !ok {
			return &WatchMessage{Err: ErrIsClosed}
		}
		return msg
	}
}

func (w *WatchChan) IsClosed() bool {
	select {
	case <-w.ctx.Done():
		return true
	case <-w.done:
		return true
	default:
		return false
	}
}

func (w *WatchChan) Close() {
	select {
	case <-w.done:
		return
	default:
		w.lg.Sugar().Infof("close watch [%s]", w.name)
		close(w.done)
	}
}
