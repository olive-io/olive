package mvcc

import (
	"context"

	"github.com/olive-io/olive/server/lease"
	"go.etcd.io/etcd/pkg/v3/traceutil"
)

type readView struct{ kv IKV }

func (rv *readView) FirstRev() int64 {
	tr := rv.kv.Read(ConcurrentReadTxMode, traceutil.TODO())
	defer tr.End()
	return tr.FirstRev()
}

func (rv *readView) Rev() int64 {
	tr := rv.kv.Read(ConcurrentReadTxMode, traceutil.TODO())
	defer tr.End()
	return tr.Rev()
}

func (rv *readView) Range(ctx context.Context, key, end []byte, ro RangeOptions) (r *RangeResult, err error) {
	tr := rv.kv.Read(ConcurrentReadTxMode, traceutil.TODO())
	defer tr.End()
	return tr.Range(ctx, key, end, ro)
}

func (rv *readView) Versions(ctx context.Context, key []byte) (revs []int64, err error) {
	tr := rv.kv.Read(ConcurrentReadTxMode, traceutil.TODO())
	defer tr.End()
	return tr.Versions(ctx, key)
}

type writeView struct{ kv IKV }

func (wv *writeView) DeleteRange(key, end []byte) (n, rev int64) {
	tw := wv.kv.Write(traceutil.TODO())
	defer tw.End()
	return tw.DeleteRange(key, end)
}

func (wv *writeView) Put(key, value []byte, lease lease.LeaseID) (rev int64) {
	tw := wv.kv.Write(traceutil.TODO())
	defer tw.End()
	return tw.Put(key, value, lease)
}
