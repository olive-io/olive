package client

import (
	"time"

	pb "github.com/olive-io/olive/api/serverpb"
)

const (
	EventTypeDelete = pb.Event_DELETE
	EventTypePut    = pb.Event_PUT

	closeSendErrTimeout = 250 * time.Millisecond

	// AutoWatchID is the watcher ID passed in WatchStream.Watch when no
	// user-provided ID is available. If pass, an ID will automatically be assigned.
	AutoWatchID = 0

	// InvalidWatchID represents an invalid watch ID and prevents duplication with an existing watch.
	InvalidWatchID = -1
)
