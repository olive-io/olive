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
	"encoding/base64"
	"fmt"
	urlpkg "net/url"
	"path"
	"strings"

	"github.com/cockroachdb/errors"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/olive-io/olive/apis/pb/auth"
	pb "github.com/olive-io/olive/apis/pb/olive"
	"github.com/olive-io/olive/apis/rpctypes"

	"github.com/olive-io/olive/meta/pagation"
	mdutil "github.com/olive-io/olive/pkg/context/metadata"
	"github.com/olive-io/olive/pkg/crypto"
	"github.com/olive-io/olive/pkg/jwt"
	ort "github.com/olive-io/olive/pkg/runtime"
)

var reqEnd = "\x00"

func withKeyEnd(key string) string {
	return key + reqEnd
}

type decodeFunc func(kv *mvccpb.KeyValue) error

func (s *Server) get(ctx context.Context, key string, target proto.Message, opts ...clientv3.OpOption) (*etcdserverpb.ResponseHeader, *mvccpb.KeyValue, error) {
	rsp, err := s.v3cli.Get(ctx, key, opts...)
	if err != nil {
		return nil, nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, nil, rpctypes.ErrKeyNotFound
	}
	kv := rsp.Kvs[0]

	if target != nil {
		err = proto.Unmarshal(kv.Value, target)
		if err != nil {
			return nil, nil, err
		}
	}
	return rsp.Header, kv, nil
}

func (s *Server) put(ctx context.Context, key string, value proto.Message, opts ...clientv3.OpOption) (*etcdserverpb.ResponseHeader, error) {
	data, _ := proto.Marshal(value)
	rsp, err := s.v3cli.Put(ctx, key, string(data), opts...)
	if err != nil {
		return nil, err
	}

	return rsp.Header, nil
}

func (s *Server) del(ctx context.Context, key string, opts ...clientv3.OpOption) (*etcdserverpb.ResponseHeader, error) {
	rsp, err := s.v3cli.Delete(ctx, key, opts...)
	if err != nil {
		return nil, err
	}

	return rsp.Header, nil
}

func (s *Server) list(ctx context.Context, key string, fn decodeFunc) error {
	if fn == nil {
		return nil
	}

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
	}

	rsp, err := s.v3cli.Get(ctx, key, options...)
	if err != nil {
		return err
	}

	for _, kv := range rsp.Kvs {
		if err = fn(kv); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) pageList(ctx context.Context, preparedKey string, reqLimit int64, continueToken string, fn decodeFunc) (next string, err error) {
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
	key := path.Join(ort.DefaultMetaRunnerRegistrar, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
		clientv3.WithSerializable(),
	}
	runner = &pb.Runner{}
	_, _, err = s.get(ctx, key, runner, options...)
	return
}

func (s *Server) getRegion(ctx context.Context, id uint64) (region *pb.Region, err error) {
	key := path.Join(ort.DefaultRunnerRegion, fmt.Sprintf("%d", id))
	options := []clientv3.OpOption{
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
	_, _, err = s.get(ctx, key, region, options...)
	return
}

func (s *Server) mustBeRoot(ctx context.Context) (*authv1.User, error) {
	user, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}

	if user.Role != ort.DefaultRootRole {
		return nil, errors.Wrap(rpctypes.ErrGRPCInvalidAuthMgmt, "must be root role")
	}
	return user, nil
}

func (s *Server) currentUser(ctx context.Context) (*authv1.User, error) {
	user := &authv1.User{}
	ok, err := mdutil.GetMsg(ctx, rpctypes.UserNameGRPC, user)
	if err != nil {
		return nil, errors.Wrap(rpctypes.ErrGRPCInvalidAuthMgmt, err.Error())
	}
	if !ok {
		return nil, errors.Wrap(rpctypes.ErrGRPCInvalidAuthMgmt, "user not found")
	}
	return user, nil
}

func (s *Server) unaryInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

		if !s.notifier.IsLeader() {
			return nil, rpctypes.ErrGRPCNotLeader
		}
		if s.etcd.Server.Leader() == 0 {
			return nil, rpctypes.ErrGRPCNoLeader
		}

		method := info.FullMethod
		if !strings.HasPrefix(method, "/olivepb.") ||
			method == "/olivepb.AuthRPC/Authenticate" {
			return handler(ctx, req)
		}

		md, _ := metadata.FromIncomingContext(ctx)
		if md == nil {
			md = metadata.New(map[string]string{})
		}

		vals := md.Get(rpctypes.TokenNameGRPC)
		if len(vals) == 0 {
			return nil, errors.Wrap(rpctypes.ErrGRPCInvalidAuthToken, "token not found")
		}
		token := vals[0]

		var user *authv1.User
		authKind, token, ok := strings.Cut(token, " ")
		if !ok {
			authKind = rpctypes.TokenBearer
		}

		switch authKind {
		case rpctypes.TokenBearer:
			user, err = s.bearerAuth(ctx, token)
		case rpctypes.TokenBasic:
			user, err = s.basicAuth(ctx, token)
		default:
			user, err = s.bearerAuth(ctx, token)
		}
		if err != nil {
			return nil, err
		}

		ctx = mdutil.SetMsg(ctx, rpctypes.UserNameGRPC, user)
		resp, err = handler(ctx, req)
		return resp, err
	}
}

func (s *Server) prepareReq(ctx context.Context, scopes ...*authv1.Scope) error {

	return nil
}

func (s *Server) bearerAuth(ctx context.Context, token string) (*authv1.User, error) {
	user, err := jwt.ParseToken(token)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Server) basicAuth(ctx context.Context, token string) (*authv1.User, error) {
	encodeText, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, rpctypes.ErrGRPCInvalidAuthToken
	}
	username, password, ok := strings.Cut(string(encodeText), ":")
	if !ok {
		return nil, rpctypes.ErrGRPCAuthFailed
	}

	key := path.Join(ort.DefaultUserPrefix, username)
	user := &authv1.User{}
	_, _, err = s.get(ctx, key, user, clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	decodedPasswd := crypto.NewSha256().Hash([]byte(password))
	if decodedPasswd != user.Password {
		return nil, rpctypes.ErrGRPCAuthFailed
	}
	return user, nil
}

func (s *Server) admit(ctx context.Context, role, user string, scope *authv1.Scope) error {
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
