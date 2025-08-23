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
	"fmt"
	"runtime"
	"time"

	ants "github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	"github.com/olive-io/olive/api/types"
)

var (
	DefaultExecutePoolSize = runtime.NumCPU() * 10
)

type antLogger struct {
	lg *zap.SugaredLogger
}

func (l *antLogger) Printf(format string, args ...any) {
	l.lg.Debugf(format, args...)
}

type Scheduler struct {
	ctx     context.Context
	cancel  context.CancelFunc
	options *Options

	lg *zap.Logger

	executePool *ants.Pool
}

func NewScheduler(pctx context.Context, options *Options) (*Scheduler, error) {
	lg := options.Logger
	if lg == nil {
		lg = zap.NewNop()
	}

	poolOpts := []ants.Option{
		ants.WithPreAlloc(true),
		ants.WithLogger(&antLogger{lg: lg.Sugar()}),
	}
	executePool, err := ants.NewPool(options.ExecutePoolSize, poolOpts...)
	if err != nil {
		return nil, fmt.Errorf("create execute pool: %w", err)
	}

	ctx, cancel := context.WithCancel(pctx)
	scheduler := &Scheduler{
		ctx:         ctx,
		cancel:      cancel,
		lg:          lg,
		options:     options,
		executePool: executePool,
	}

	go scheduler.process(ctx)
	return scheduler, nil
}

func (sch *Scheduler) Execute(ctx context.Context, process *types.Process) error {
	return nil
}

func (sch *Scheduler) process(ctx context.Context) {

	tick := time.Microsecond * 100
	timer := time.NewTimer(tick)
LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case <-timer.C:
			timer.Reset(tick)
		}
	}

	sch.destroy()
}

func (sch *Scheduler) destroy() {
	sch.lg.Debug("destroying scheduler")

	sch.cancel()

	// release goroutines pool
	sch.lg.Debug("release scheduler execute pool")
	sch.executePool.Release()
	sch.lg.Debug("released scheduler execute pool")
}
