package runner

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync/atomic"

	"github.com/cockroachdb/pebble"
	sm "github.com/lni/dragonboat/v4/statemachine"
)

type BrainStorage struct {
	Option

	clusterID int64
	nodeID    int64

	db *pebble.DB

	applied atomic.Uint64

	done  chan struct{}
	close chan struct{}
}

func (s *BrainStorage) Open(stopc <-chan struct{}) (uint64, error) {
	options := pebble.Options{}
	region := fmt.Sprintf("%d_%d", s.clusterID, s.nodeID)
	dataDir := filepath.Join(s.Dir, region)
	db, err := pebble.Open(dataDir, &options)
	if err != nil {
		return 0, err
	}

	val, closer, err := db.Get([]byte(AppliedIndex))
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return 0, err
	}

	defer func() {
		if closer != nil {
			closer.Close()
		}
	}()

	if len(val) == 0 {
		return 0, nil
	}

	applied := binary.LittleEndian.Uint64(val)
	s.applied.Store(applied)

	return applied, nil
}

func (s *BrainStorage) Update(entries []sm.Entry) ([]sm.Entry, error) {
	for _, en := range entries {
		_ = en.Index
		//en.Result.
	}

	iter := s.db.NewIter(&pebble.IterOptions{})
	for iter.First(); iter.Valid(); iter.Next() {

	}

	return entries, nil
}

func (s *BrainStorage) Lookup(in interface{}) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (s *BrainStorage) Sync() error {
	return nil
}

func (s *BrainStorage) PrepareSnapshot() (interface{}, error) {
	snapshot := s.db.NewSnapshot()
	return snapshot, nil
}

func (s *BrainStorage) SaveSnapshot(in interface{}, writer io.Writer, done <-chan struct{}) error {
	//snapshot := in.(*pebble.Snapshot)
	//snapshot.

	return nil
}

func (s *BrainStorage) RecoverFromSnapshot(reader io.Reader, done <-chan struct{}) error {
	//TODO implement me
	//panic("implement me")
	//sn := s.db.NewSnapshot()
	return nil
}

func (s *BrainStorage) Close() error {
	if s.isClosed() {
		return nil
	}

	close(s.close)
	return s.db.Close()
}

func (s *BrainStorage) isClosed() bool {
	select {
	case <-s.close:
		return true
	default:
		return false
	}
}
