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
	"strings"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/api/rpctypes"
	"github.com/olive-io/olive/meta/pagation"
	"github.com/olive-io/olive/pkg/runtime"
)

var reqEnd = "\x00"

func withKeyEnd(key string) string {
	return key + reqEnd
}

type pageDecodeFunc func(kv *mvccpb.KeyValue) error

func (s *Server) pageList(ctx context.Context, preparedKey string, reqLimit int64, continueToken string, fn pageDecodeFunc) (next string, err error) {
	keyPrefix := preparedKey
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}
	if !strings.HasSuffix(keyPrefix, "/") {
		keyPrefix += "/"
	}

	var paging bool
	// set the appropriate clientv3 options to filter the returned data set
	var limitOption *clientv3.OpOption
	limit := reqLimit
	if limit > 0 {
		paging = true
		options = append(options, clientv3.WithLimit(limit))
		limitOption = &options[len(options)-1]
	}

	var returnedRV, continueRV, withRev int64
	var continueKey string
	if len(continueToken) > 0 {
		continueKey, continueRV, err = pagation.DecodeContinue(continueToken, keyPrefix)
		if err != nil {
			return "", fmt.Errorf("invalid continue token: %v", err)
		}

		rangeEnd := clientv3.GetPrefixRangeEnd(keyPrefix)
		options = append(options, clientv3.WithRange(rangeEnd))
		preparedKey = continueKey

		// If continueRV > 0, the LIST req needs a specific resource version.
		// continueRV==0 is invalid.
		// If continueRV < 0, the req is for the latest resource version.
		if continueRV > 0 {
			withRev = continueRV
			returnedRV = continueRV
		}
	}

	if withRev != 0 {
		options = append(options, clientv3.WithRev(withRev))
	}

	var lastKey []byte
	var hasMore bool
	decoded := int64(0)
	for {
		getResp, err := s.v3cli.Get(ctx, preparedKey, options...)
		if err != nil {
			return "", err
		}
		hasMore = getResp.More

		if len(getResp.Kvs) == 0 && getResp.More {
			return "", fmt.Errorf("no results were found, but olive indicated there were more values remaining")
		}

		for _, kv := range getResp.Kvs {
			if paging && decoded >= reqLimit {
				hasMore = true
				break
			}
			lastKey = kv.Key
			err = fn(kv)
			if err != nil {
				return "", fmt.Errorf("%w: decode values", err)
			}

			decoded += 1
		}

		if !hasMore || !paging {
			break
		}

		// indicate to the client which resource version was returned
		if returnedRV == 0 {
			returnedRV = getResp.Header.Revision
		}

		if decoded >= reqLimit {
			break
		}

		if limit < maxLimit {
			limit *= 2
			if limit > maxLimit {
				limit = maxLimit
			}
			*limitOption = clientv3.WithLimit(limit)
		}
		preparedKey = withKeyEnd(string(lastKey))
		if withRev == 0 {
			withRev = returnedRV
			options = append(options, clientv3.WithRev(withRev))
		}
	}

	if hasMore {
		// we want to start immediately after the last key
		next, err = pagation.EncodeContinue(withKeyEnd(string(lastKey)), keyPrefix, returnedRV)
		if err != nil {
			return "", err
		}
	}

	return
}

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
