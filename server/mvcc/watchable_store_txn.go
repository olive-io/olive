package mvcc

import (
	pb "github.com/olive-io/olive/api/serverpb"
	"go.etcd.io/etcd/pkg/v3/traceutil"
)

func (tw *watchableStoreTxnWrite) End() {
	changes := tw.Changes()
	if len(changes) == 0 {
		tw.ITxnWrite.End()
		return
	}

	rev := tw.Rev() + 1
	evs := make([]pb.Event, len(changes))
	for i, change := range changes {
		evs[i].Kv = &changes[i]
		if change.CreateRevision == 0 {
			evs[i].Type = pb.Event_DELETE
			evs[i].Kv.ModRevision = rev
		} else {
			evs[i].Type = pb.Event_PUT
		}
	}

	// end write txn under watchable store lock so the updates are visible
	// when asynchronous event posting checks the current store revision
	tw.s.mu.Lock()
	tw.s.notify(rev, evs)
	tw.ITxnWrite.End()
	tw.s.mu.Unlock()
}

type watchableStoreTxnWrite struct {
	ITxnWrite
	s *watchableStore
}

func (s *watchableStore) Write(trace *traceutil.Trace) ITxnWrite {
	return &watchableStoreTxnWrite{s.store.Write(trace), s}
}
