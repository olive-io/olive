/*
   Copyright 2023 The olive Authors

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

package backend

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/cockroachdb/pebble"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/apis/pb/olive"
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

	values := make([]*pb.InternalKV, 0)

	for iter.SeekPrefixGE(prefix); iteratorIsValid(iter); iter.Next() {
		key := iter.Key()
		val := iter.Value()
		rkv := &pb.InternalKV{
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
		data, err := proto.Marshal(rkv)
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
