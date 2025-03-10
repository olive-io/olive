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
	"sort"
)

const bucketBufferInitialSize = 512

// txBuffer handles functionality shared between txWriteBuffer and txReadBuffer.
type txBuffer struct {
	buckets map[BucketID]*bucketBuffer
}

func (txb *txBuffer) reset() {
	for k, v := range txb.buckets {
		if v.used == 0 {
			// demote
			delete(txb.buckets, k)
		}
		v.used = 0
	}
}

// txWriteBuffer buffers writes of pending updates that have not yet committed.
type txWriteBuffer struct {
	txBuffer
	// Map from bucket ID into information whether this bucket is edited
	// sequentially (i.e. keys are growing monotonically).
	bucket2seq map[BucketID]bool
}

func (txw *txWriteBuffer) put(bucket IBucket, k, v []byte) {
	txw.bucket2seq[bucket.ID()] = false
	txw.putInternal(bucket, k, v)
}

func (txw *txWriteBuffer) putSeq(bucket IBucket, k, v []byte) {
	// TODO: Add (in tests?) verification whether k>b[len(b)]
	txw.putInternal(bucket, k, v)
}

func (txw *txWriteBuffer) putInternal(bucket IBucket, k, v []byte) {
	b, ok := txw.buckets[bucket.ID()]
	if !ok {
		b = newBucketBuffer()
		txw.buckets[bucket.ID()] = b
	}
	b.add(k, v)
}

func (txw *txWriteBuffer) reset() {
	txw.txBuffer.reset()
	for k := range txw.bucket2seq {
		v, ok := txw.buckets[k]
		if !ok {
			delete(txw.bucket2seq, k)
		} else if v.used == 0 {
			txw.bucket2seq[k] = true
		}
	}
}

func (txw *txWriteBuffer) writeback(txr *txReadBuffer) {
	for k, wb := range txw.buckets {
		rb, ok := txr.buckets[k]
		if !ok {
			delete(txw.buckets, k)
			txr.buckets[k] = wb
			continue
		}
		if seq, ok := txw.bucket2seq[k]; ok && !seq && wb.used > 1 {
			// assume no duplicate keys
			sort.Sort(wb)
		}
		rb.merge(wb)
	}
	txw.reset()
	// increase the buffer version
	txr.bufVersion++
}

// txReadBuffer accesses buffered updates.
type txReadBuffer struct {
	txBuffer
	// bufVersion is used to check if the buffer is modified recently
	bufVersion uint64
}

func (txr *txReadBuffer) Range(bucket IBucket, key, endKey []byte, limit int64) ([][]byte, [][]byte, error) {
	if b := txr.buckets[bucket.ID()]; b != nil {
		return b.Range(key, endKey, limit)
	}
	return nil, nil, nil
}

func (txr *txReadBuffer) ForEach(bucket IBucket, visitor func(k, v []byte) error) error {
	if b := txr.buckets[bucket.ID()]; b != nil {
		return b.ForEach(visitor)
	}
	return nil
}

// unsafeCopy returns a copy of txReadBuffer, caller should acquire backend.readTx.RLock()
func (txr *txReadBuffer) unsafeCopy() txReadBuffer {
	txrCopy := txReadBuffer{
		txBuffer: txBuffer{
			buckets: make(map[BucketID]*bucketBuffer, len(txr.txBuffer.buckets)),
		},
		bufVersion: 0,
	}
	for bucketName, bucket := range txr.txBuffer.buckets {
		txrCopy.txBuffer.buckets[bucketName] = bucket.Copy()
	}
	return txrCopy
}

type kv struct {
	key []byte
	val []byte
}

// bucketBuffer buffers key-value pairs that are pending commit.
type bucketBuffer struct {
	buf []kv
	// used tracks number of elements in use so buf can be reused without reallocation.
	used int
}

func newBucketBuffer() *bucketBuffer {
	return &bucketBuffer{buf: make([]kv, bucketBufferInitialSize), used: 0}
}

func (bb *bucketBuffer) Range(key, endKey []byte, limit int64) (keys [][]byte, vals [][]byte, err error) {
	f := func(i int) bool { return bytes.Compare(bb.buf[i].key, key) >= 0 }
	idx := sort.Search(bb.used, f)
	if idx < 0 {
		return nil, nil, nil
	}
	if len(endKey) == 0 {
		if bytes.Equal(key, bb.buf[idx].key) {
			keys = append(keys, bb.buf[idx].key)
			vals = append(vals, bb.buf[idx].val)
		}
		return keys, vals, nil
	}
	if bytes.Compare(endKey, bb.buf[idx].key) <= 0 {
		return nil, nil, nil
	}
	for i := idx; i < bb.used && int64(len(keys)) < limit; i++ {
		if bytes.Compare(endKey, bb.buf[i].key) <= 0 {
			break
		}
		keys = append(keys, bb.buf[i].key)
		vals = append(vals, bb.buf[i].val)
	}
	return keys, vals, nil
}

func (bb *bucketBuffer) ForEach(visitor func(k, v []byte) error) error {
	for i := 0; i < bb.used; i++ {
		if err := visitor(bb.buf[i].key, bb.buf[i].val); err != nil {
			return err
		}
	}
	return nil
}

func (bb *bucketBuffer) add(k, v []byte) {
	bb.buf[bb.used].key, bb.buf[bb.used].val = k, v
	bb.used++
	if bb.used == len(bb.buf) {
		buf := make([]kv, (3*len(bb.buf))/2)
		copy(buf, bb.buf)
		bb.buf = buf
	}
}

// merge merges data from bbsrc into bb.
func (bb *bucketBuffer) merge(bbsrc *bucketBuffer) {
	for i := 0; i < bbsrc.used; i++ {
		bb.add(bbsrc.buf[i].key, bbsrc.buf[i].val)
	}
	if bb.used == bbsrc.used {
		return
	}
	if bytes.Compare(bb.buf[(bb.used-bbsrc.used)-1].key, bbsrc.buf[0].key) < 0 {
		return
	}

	sort.Stable(bb)

	// remove duplicates, using only newest update
	widx := 0
	for ridx := 1; ridx < bb.used; ridx++ {
		if !bytes.Equal(bb.buf[ridx].key, bb.buf[widx].key) {
			widx++
		}
		bb.buf[widx] = bb.buf[ridx]
	}
	bb.used = widx + 1
}

func (bb *bucketBuffer) Len() int { return bb.used }
func (bb *bucketBuffer) Less(i, j int) bool {
	return bytes.Compare(bb.buf[i].key, bb.buf[j].key) < 0
}
func (bb *bucketBuffer) Swap(i, j int) { bb.buf[i], bb.buf[j] = bb.buf[j], bb.buf[i] }

func (bb *bucketBuffer) Copy() *bucketBuffer {
	bbCopy := bucketBuffer{
		buf:  make([]kv, len(bb.buf)),
		used: bb.used,
	}
	copy(bbCopy.buf, bb.buf)
	return &bbCopy
}
