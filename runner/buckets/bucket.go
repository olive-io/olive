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

package buckets

import (
	"bytes"

	"github.com/olive-io/olive/runner/backend"
)

var (
	keyBucketName    = []byte("key")
	metaBucketName   = []byte("meta")
	regionBucketName = []byte("region")

	testBucketName = []byte("test")
)

var (
	Key    = backend.IBucket(bucket{id: 1, name: keyBucketName, safeRangeBucket: true})
	Meta   = backend.IBucket(bucket{id: 2, name: metaBucketName, safeRangeBucket: false})
	Region = backend.IBucket(bucket{id: 3, name: regionBucketName, safeRangeBucket: true})

	Test = backend.IBucket(bucket{id: 100, name: testBucketName, safeRangeBucket: false})
)

type bucket struct {
	id              backend.BucketID
	name            []byte
	safeRangeBucket bool
}

func (b bucket) ID() backend.BucketID    { return b.id }
func (b bucket) Name() []byte            { return b.name }
func (b bucket) String() string          { return string(b.Name()) }
func (b bucket) IsSafeRangeBucket() bool { return b.safeRangeBucket }

var (
	MetaConsistentIndexKeyName = []byte("consistent_index")
	MetaTermKeyName            = []byte("term")
)

// DefaultIgnores defines buckets & keys to ignore in hash checking.
func DefaultIgnores(bucket, key []byte) bool {
	// consistent index & term might be changed due to v2 internal sync, which
	// is not controllable by the user.
	return bytes.Compare(bucket, Meta.Name()) == 0 &&
		(bytes.Compare(key, MetaTermKeyName) == 0 || bytes.Compare(key, MetaConsistentIndexKeyName) == 0)
}
