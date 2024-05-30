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

package parallelize

import (
	"context"
	"math"

	"k8s.io/client-go/util/workqueue"

	"github.com/olive-io/olive/mon/scheduler/metrics"
)

// DefaultParallelism is the default parallelism used in scheduler.
const DefaultParallelism int = 16

// Parallelizer holds the parallelism for scheduler.
type Parallelizer struct {
	parallelism int
}

// NewParallelizer returns an object holding the parallelism.
func NewParallelizer(p int) Parallelizer {
	return Parallelizer{parallelism: p}
}

// chunkSizeFor returns a chunk size for the given number of items to use for
// parallel work. The size aims to produce good CPU utilization.
// returns max(1, min(sqrt(n), n/Parallelism))
func chunkSizeFor(n, parallelism int) int {
	s := int(math.Sqrt(float64(n)))

	if r := n/parallelism + 1; s > r {
		s = r
	} else if s < 1 {
		s = 1
	}
	return s
}

// Until is a wrapper around workqueue.ParallelizeUntil to use in scheduling algorithms.
// A given operation will be a label that is recorded in the goroutine metric.
func (p Parallelizer) Until(ctx context.Context, pieces int, doWorkPiece workqueue.DoWorkPieceFunc, operation string) {
	goroutinesMetric := metrics.Goroutines.WithLabelValues(operation)
	withMetrics := func(piece int) {
		goroutinesMetric.Inc()
		doWorkPiece(piece)
		goroutinesMetric.Dec()
	}

	workqueue.ParallelizeUntil(ctx, p.parallelism, pieces, withMetrics, workqueue.WithChunkSize(chunkSizeFor(pieces, p.parallelism)))
}
