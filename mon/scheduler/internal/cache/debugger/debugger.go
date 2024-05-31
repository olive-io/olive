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

package debugger

import (
	"context"
	"os"
	"os/signal"

	"k8s.io/klog/v2"

	corelisters "github.com/olive-io/olive/client-go/generated/listers/core/v1"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
)

// CacheDebugger provides ways to check and write cache information for debugging.
type CacheDebugger struct {
	Comparer CacheComparer
	Dumper   CacheDumper
}

// New creates a CacheDebugger.
func New(
	runnerLister corelisters.RunnerLister,
	regionLister corelisters.RegionLister,
	cache internalcache.Cache,
	regionQueue internalqueue.SchedulingQueue,
) *CacheDebugger {
	return &CacheDebugger{
		Comparer: CacheComparer{
			RunnerLister: runnerLister,
			RegionLister: regionLister,
			Cache:        cache,
			RegionQueue:  regionQueue,
		},
		Dumper: CacheDumper{
			cache:       cache,
			regionQueue: regionQueue,
		},
	}
}

// ListenForSignal starts a goroutine that will trigger the CacheDebugger's
// behavior when the process receives SIGINT (Windows) or SIGUSER2 (non-Windows).
func (d *CacheDebugger) ListenForSignal(ctx context.Context) {
	logger := klog.FromContext(ctx)
	stopCh := ctx.Done()
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, compareSignal)

	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-ch:
				d.Comparer.Compare(logger)
				d.Dumper.DumpAll(logger)
			}
		}
	}()
}
