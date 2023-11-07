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

package mvcc

import (
	"context"

	"github.com/olive-io/olive/server/lease"
)

type metricsTxnWrite struct {
	ITxnWrite
	ranges  uint
	puts    uint
	deletes uint
	putSize int64
}

func newMetricsTxnRead(tr ITxnRead) ITxnRead {
	return &metricsTxnWrite{&txnReadWrite{tr}, 0, 0, 0, 0}
}

func newMetricsTxnWrite(tw ITxnWrite) ITxnWrite {
	return &metricsTxnWrite{tw, 0, 0, 0, 0}
}

func (tw *metricsTxnWrite) Range(ctx context.Context, key, end []byte, ro RangeOptions) (*RangeResult, error) {
	tw.ranges++
	return tw.ITxnWrite.Range(ctx, key, end, ro)
}

func (tw *metricsTxnWrite) DeleteRange(key, end []byte) (n, rev int64) {
	tw.deletes++
	return tw.ITxnWrite.DeleteRange(key, end)
}

func (tw *metricsTxnWrite) Put(key, value []byte, lease lease.LeaseID) (rev int64) {
	tw.puts++
	size := int64(len(key) + len(value))
	tw.putSize += size
	return tw.ITxnWrite.Put(key, value, lease)
}

func (tw *metricsTxnWrite) End() {
	defer tw.ITxnWrite.End()
	if sum := tw.ranges + tw.puts + tw.deletes; sum > 1 {
		txnCounter.Inc()
	}

	ranges := float64(tw.ranges)
	rangeCounter.Add(ranges)
	rangeCounterDebug.Add(ranges) // TODO: remove in 3.5 release

	puts := float64(tw.puts)
	putCounter.Add(puts)
	totalPutSizeGauge.Add(float64(tw.putSize))

	deletes := float64(tw.deletes)
	deleteCounter.Add(deletes)
}
