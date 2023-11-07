package mvcc

import (
	"fmt"
	"testing"

	"github.com/olive-io/olive/server/lease"
	betesting "github.com/olive-io/olive/server/mvcc/backend/testing"
	"go.uber.org/zap"
)

func BenchmarkKVWatcherMemoryUsage(b *testing.B) {
	be, tmpPath := betesting.NewDefaultTmpBackend(b)
	watchable := newWatchableStore(zap.NewExample(), be, &lease.FakeLessor{}, StoreConfig{})

	defer cleanup(watchable, be, tmpPath)

	w := watchable.NewWatchStream()

	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w.Watch(0, []byte(fmt.Sprint("foo", i)), nil, 0)
	}
}
