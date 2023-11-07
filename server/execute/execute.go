package execute

import (
	"context"

	pb "github.com/olive-io/olive/api/serverpb"
)

// IExecutor executes jobs at the replica which is leader of shard
type IExecutor interface {
	Execute(ctx context.Context, r *pb.ExecuteRequest) (*pb.ExecuteResponse, error)
}
