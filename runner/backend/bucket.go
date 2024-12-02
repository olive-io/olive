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
	"github.com/cockroachdb/pebble"
	json "github.com/json-iterator/go"

	xpath "github.com/olive-io/olive/x/path"
)

var (
	defaultBucketPrefix = []byte("_bucket")
)

type BucketID int

type IBucket interface {
	// ID returns a unique identifier of a bucket.
	// The id must NOT be persisted and can be used as lightweight identificator
	// in the in-memory maps.
	ID() BucketID
	Name() []byte
	// String implements Stringer (human readable name).
	String() string

	// IsSafeRangeBucket is a hack to avoid inadvertently reading duplicate keys;
	// overwrites on a bucket should only fetch with limit=1, but safeRangeBucket
	// is known to never overwrite any key so range is safe.
	IsSafeRangeBucket() bool
}

type localBucket struct {
	Name_  string   `json:"name"`
	ID_    BucketID `json:"id"`
	IsSafe bool     `json:"is-safe"`
}

func (b *localBucket) ID() BucketID {
	return b.ID_
}

func (b *localBucket) Name() []byte {
	return []byte(b.Name_)
}

func (b *localBucket) String() string {
	return b.Name_
}

func (b *localBucket) IsSafeRangeBucket() bool {
	return b.IsSafe
}

func readBucket(tx *pebble.Batch, name []byte) (*localBucket, error) {
	key := xpath.Join(defaultBucketPrefix, name)
	value, closer, err := tx.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	bucket := &localBucket{}
	err = json.Unmarshal(value, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func createBucket(tx *pebble.Batch, bucket IBucket) error {
	key := xpath.Join(defaultBucketPrefix, bucket.Name())
	lb := &localBucket{
		Name_:  string(bucket.Name()),
		ID_:    bucket.ID(),
		IsSafe: bucket.IsSafeRangeBucket(),
	}

	value, _ := json.Marshal(lb)
	wo := &pebble.WriteOptions{Sync: true}
	return tx.Set(key, value, wo)
}

func deleteBucket(tx *pebble.Batch, bucket IBucket) error {
	key := xpath.Join(defaultBucketPrefix, bucket.Name())
	wo := &pebble.WriteOptions{Sync: true}
	return tx.Delete(key, wo)
}
