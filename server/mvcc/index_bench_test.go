package mvcc

import (
	"testing"

	"go.uber.org/zap"
)

func BenchmarkIndexCompact1(b *testing.B)       { benchmarkIndexCompact(b, 1) }
func BenchmarkIndexCompact100(b *testing.B)     { benchmarkIndexCompact(b, 100) }
func BenchmarkIndexCompact10000(b *testing.B)   { benchmarkIndexCompact(b, 10000) }
func BenchmarkIndexCompact100000(b *testing.B)  { benchmarkIndexCompact(b, 100000) }
func BenchmarkIndexCompact1000000(b *testing.B) { benchmarkIndexCompact(b, 1000000) }

func benchmarkIndexCompact(b *testing.B, size int) {
	log := zap.NewNop()
	kvindex := newTreeIndex(log)

	bytesN := 64
	keys := createBytesSlice(bytesN, size)
	for i := 1; i < size; i++ {
		kvindex.Put(keys[i], revision{main: int64(i), sub: int64(i)})
	}
	b.ResetTimer()
	for i := 1; i < b.N; i++ {
		kvindex.Compact(int64(i))
	}
}
