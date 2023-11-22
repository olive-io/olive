package backend

import (
	"github.com/cockroachdb/pebble"
)

func DbFromBackendForTest(b IBackend) *pebble.DB {
	return b.(*backend).db
}

func CommitsForTest(b IBackend) int64 {
	return b.(*backend).Commits()
}
