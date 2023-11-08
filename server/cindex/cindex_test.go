package cindex

import (
	"math/rand"
	"testing"
	"time"

	"github.com/olive-io/olive/server/mvcc/backend"
	"github.com/olive-io/olive/server/mvcc/backend/testing"
	"github.com/stretchr/testify/assert"
)

// TestConsistentIndex ensures that LoadConsistentIndex/Save/ConsistentIndex and backend.IBatchTx can work well together.
func TestConsistentIndex(t *testing.T) {

	be, tmpPath := betesting.NewTmpBackend(t, time.Microsecond, 10)
	ci := NewConsistentIndex(be, 0, 0)

	tx := be.BatchTx()
	if tx == nil {
		t.Fatal("batch tx is nil")
	}
	tx.Lock()

	UnsafeCreateMetaBucket(tx)
	tx.Unlock()
	be.ForceCommit()
	r := uint64(7890123)
	term := uint64(234)
	ci.SetConsistentIndex(r, term)
	index := ci.ConsistentIndex()
	if index != r {
		t.Errorf("expected %d, got %d", r, index)
	}
	tx.Lock()
	ci.UnsafeSave(tx)
	tx.Unlock()
	be.ForceCommit()
	be.Close()

	b := backend.NewDefaultBackend(tmpPath)
	defer b.Close()
	ci.SetBackend(b)
	index = ci.ConsistentIndex()
	assert.Equal(t, r, index)

	ci = NewConsistentIndex(b, 0, 0)
	index = ci.ConsistentIndex()
	assert.Equal(t, r, index)
}

func TestConsistentIndexDecrease(t *testing.T) {
	initIndex := uint64(100)
	initTerm := uint64(10)

	tcs := []struct {
		name  string
		index uint64
		term  uint64
	}{
		{
			name:  "Decrease term",
			index: initIndex + 1,
			term:  initTerm - 1,
		},
		{
			name:  "Decrease CI",
			index: initIndex - 1,
			term:  initTerm + 1,
		},
		{
			name:  "Decrease CI and term",
			index: initIndex - 1,
			term:  initTerm - 1,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			be, tmpPath := betesting.NewTmpBackend(t, time.Microsecond, 10)
			tx := be.BatchTx()
			tx.Lock()
			UnsafeCreateMetaBucket(tx)
			UnsafeUpdateConsistentIndex(tx, initIndex, initTerm, 0, 0)
			tx.Unlock()
			be.ForceCommit()
			be.Close()

			be = backend.NewDefaultBackend(tmpPath)
			defer be.Close()
			ci := NewConsistentIndex(be, 0, 0)
			ci.SetConsistentIndex(tc.index, tc.term)
			tx = be.BatchTx()
			tx.Lock()
			ci.UnsafeSave(tx)
			tx.Unlock()
			assert.Equal(t, tc.index, ci.ConsistentIndex())

			ci = NewConsistentIndex(be, 0, 0)
			assert.Equal(t, tc.index, ci.ConsistentIndex())
		})
	}
}

func TestFakeConsistentIndex(t *testing.T) {

	r := rand.Uint64()
	ci := NewFakeConsistentIndex(r)
	index := ci.ConsistentIndex()
	if index != r {
		t.Errorf("expected %d, got %d", r, index)
	}
	r = rand.Uint64()
	ci.SetConsistentIndex(r, 5)
	index = ci.ConsistentIndex()
	if index != r {
		t.Errorf("expected %d, got %d", r, index)
	}

}
