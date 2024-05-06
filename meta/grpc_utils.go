/*
   Copyright 2024 The olive Authors

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

package meta

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"path"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/pkg/runtime"
)

func (s *Server) getRunner(ctx context.Context, id uint64) (runner *pb.Runner, err error) {
	key := path.Join(runtime.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrKeyNotFound
	}
	runner = new(pb.Runner)
	_ = proto.Unmarshal(rsp.Kvs[0].Value, runner)
	return
}

func (s *Server) getRegion(ctx context.Context, id uint64) (region *pb.Region, err error) {
	key := path.Join(runtime.DefaultRunnerRegion, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return nil, err
	}
	if len(rsp.Kvs) == 0 {
		return nil, rpctypes.ErrKeyNotFound
	}
	region = new(pb.Region)
	_ = proto.Unmarshal(rsp.Kvs[0].Value, region)
	return
}

func (s *Server) reqPrepare(ctx context.Context) error {
	if !s.notifier.IsLeader() {
		return rpctypes.ErrGRPCNotLeader
	}
	if s.etcd.Server.Leader() == 0 {
		return rpctypes.ErrGRPCNoLeader
	}

	return nil
}

func (s *Server) responseHeader() *pb.ResponseHeader {
	es := s.etcd.Server
	header := &pb.ResponseHeader{
		ClusterId: uint64(es.Cluster().ID()),
		MemberId:  uint64(es.ID()),
		RaftTerm:  es.Term(),
	}
	return header
}

func (s *Server) buildGRPCConn(ctx context.Context, targetURL string) (*grpc.ClientConn, error) {
	url, err := urlpkg.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	host := url.Host

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	return grpc.DialContext(ctx, host, options...)
}
