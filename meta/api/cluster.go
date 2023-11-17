package api

import "github.com/olive-io/olive/server/membership"

// Cluster is an interface representing a collection of members in one olive cluster.
type Cluster interface {
	// ID returns the cluster ID
	ID() uint64
	// ClientURLs returns an aggregate set of all URLs on which this
	// cluster is listening for client requests
	ClientURLs() []string
	// Members returns a slice of members sorted by their ID
	Members() []*membership.Member
	// Member retrieves a particular member based on ID, or nil if the
	// member does not exist in the cluster
	Member(id uint64) *membership.Member
}
