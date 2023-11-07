package hooks

import pb "github.com/olive-io/olive/api/serverpb"

type IExecuteHook interface {
	OnPreExecute(r *pb.ExecuteRequest, done <-chan struct{})
}
