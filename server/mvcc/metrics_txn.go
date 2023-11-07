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
