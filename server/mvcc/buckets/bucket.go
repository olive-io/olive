package buckets

import (
	"bytes"

	"github.com/olive-io/olive/server/mvcc/backend"
)

var (
	keyBucketName   = []byte("key")
	metaBucketName  = []byte("meta")
	leaseBucketName = []byte("lease")

	testBucketName = []byte("test")
)

var (
	Key   = backend.IBucket(bucket{id: 1, name: keyBucketName, safeRangeBucket: true})
	Meta  = backend.IBucket(bucket{id: 2, name: metaBucketName, safeRangeBucket: false})
	Lease = backend.IBucket(bucket{id: 3, name: leaseBucketName, safeRangeBucket: false})

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
