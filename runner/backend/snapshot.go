package backend

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/cockroachdb/pebble"
	pb "github.com/olive-io/olive/api/serverpb"
)

type ISnapshot interface {
	// WriteTo writes the snapshot into the given writer.
	WriteTo(w io.Writer, prefix []byte) (n int64, err error)
	// Close closes the snapshot.
	Close() error
}

type snapshot struct {
	sn    *pebble.Snapshot
	stopc chan struct{}
	donec chan struct{}
}

func (s *snapshot) WriteTo(w io.Writer, prefix []byte) (n int64, err error) {
	ro := &pebble.IterOptions{}
	iter := s.sn.NewIter(ro)
	defer iter.Close()

	values := make([]*pb.RaftInternalKV, 0)

	for iter.SeekPrefixGE(prefix); iteratorIsValid(iter); iter.Next() {
		key := iter.Key()
		val := iter.Value()
		rkv := &pb.RaftInternalKV{
			Key:   bytes.Clone(key),
			Value: bytes.Clone(val),
		}
		values = append(values, rkv)
	}
	count := uint64(len(values))
	sz := make([]byte, 8)
	binary.LittleEndian.PutUint64(sz, count)

	var c int
	if c, err = w.Write(sz); err != nil {
		return 0, err
	}

	n += int64(c)
	for _, rkv := range values {
		data, err := rkv.Marshal()
		if err != nil {
			panic(err)
		}
		binary.LittleEndian.PutUint64(sz, uint64(len(data)))
		c, err = w.Write(sz)
		if err != nil {
			return 0, err
		}
		n += int64(c)

		c, err = w.Write(data)
		if err != nil {
			return 0, err
		}
		n += int64(c)
	}

	return n, nil
}

func (s *snapshot) Close() error {
	close(s.stopc)
	<-s.donec
	return s.sn.Close()
}
