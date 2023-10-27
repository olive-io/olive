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
	"github.com/cockroachdb/pebble"
	json "github.com/json-iterator/go"
	"github.com/oliveio/olive/pkg/bytesutil"
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
	key := bytesutil.PathJoin(defaultBucketPrefix, name)
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
	key := bytesutil.PathJoin(defaultBucketPrefix, bucket.Name())
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
	key := bytesutil.PathJoin(defaultBucketPrefix, bucket.Name())
	wo := &pebble.WriteOptions{Sync: true}
	return tx.Delete(key, wo)
}
