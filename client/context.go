package client

import (
	"context"

	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/pkg/version"
	"google.golang.org/grpc/metadata"
)

// WithRequireLeader requires client requests to only succeed
// when the cluster has a leader.
func WithRequireLeader(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok { // no outgoing metadata ctx key, create one
		md = metadata.Pairs(rpctypes.MetadataRequireLeaderKey, rpctypes.MetadataHasLeader)
		return metadata.NewOutgoingContext(ctx, md)
	}
	copied := md.Copy() // avoid racey updates
	// overwrite/add 'hasleader' key/value
	copied.Set(rpctypes.MetadataRequireLeaderKey, rpctypes.MetadataHasLeader)
	return metadata.NewOutgoingContext(ctx, copied)
}

// embeds client version
func withVersion(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok { // no outgoing metadata ctx key, create one
		md = metadata.Pairs(rpctypes.MetadataClientAPIVersionKey, version.APIVersion)
		return metadata.NewOutgoingContext(ctx, md)
	}
	copied := md.Copy() // avoid racey updates
	// overwrite/add version key/value
	copied.Set(rpctypes.MetadataClientAPIVersionKey, version.APIVersion)
	return metadata.NewOutgoingContext(ctx, copied)
}
