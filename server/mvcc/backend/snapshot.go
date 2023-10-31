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

package backend

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/cockroachdb/pebble"
	"github.com/olive-io/olive/api"
)

type ISnapshot interface {
	// WriteTo writes the snapshot into the given writer.
	WriteTo(prefix []byte, w io.Writer) (n int64, err error)
	// Close closes the snapshot.
	Close() error
}

type snapshot struct {
	sn    *pebble.Snapshot
	stopc chan struct{}
	donec chan struct{}
}

func (s *snapshot) WriteTo(prefix []byte, w io.Writer) (n int64, err error) {
	ro := &pebble.IterOptions{}
	iter := s.sn.NewIter(ro)
	defer iter.Close()

	values := make([]*api.RaftInternalKV, 0)

	if len(prefix) != 0 {
		iter.SetBounds(prefix, nil)
	}
	for iter.First(); iteratorIsValid(iter); iter.Next() {
		key := iter.Key()
		if len(prefix) != 0 && !bytes.HasPrefix(key, prefix) {
			continue
		}
		val := iter.Value()
		rkv := &api.RaftInternalKV{
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
