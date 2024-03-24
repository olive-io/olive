// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"fmt"
	"reflect"
	"time"

	json "github.com/json-iterator/go"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
)

var noPrefixEnd = []byte{0}

func warnOfExpensiveRequest(lg *zap.Logger, slowApplies prometheus.Counter, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, respMsg proto.Message, err error) {
	if time.Since(now) <= warningApplyDuration {
		return
	}
	var resp string
	if !isNil(respMsg) {
		resp = fmt.Sprintf("size:%d", proto.Size(respMsg))
	}
	warnOfExpensiveGenericRequest(lg, slowApplies, warningApplyDuration, now, reqStringer, "", resp, err)
}

func warnOfFailedRequest(lg *zap.Logger, now time.Time, reqStringer fmt.Stringer, respMsg proto.Message, err error) {
	var resp string
	if !isNil(respMsg) {
		resp = fmt.Sprintf("size:%d", proto.Size(respMsg))
	}
	d := time.Since(now)
	lg.Warn(
		"failed to apply request",
		zap.Duration("took", d),
		zap.String("request", reqStringer.String()),
		zap.String("response", resp),
		zap.Error(err),
	)
}

func warnOfExpensiveReadOnlyRangeRequest(lg *zap.Logger, slowApplies prometheus.Counter, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, rangeResponse *pb.RegionRangeResponse, err error) {
	if time.Since(now) <= warningApplyDuration {
		return
	}
	var resp string
	if !isNil(rangeResponse) {
		resp = fmt.Sprintf("range_response_count:%d size:%d", len(rangeResponse.Kvs), proto.Size(rangeResponse))
	}
	warnOfExpensiveGenericRequest(lg, slowApplies, warningApplyDuration, now, reqStringer, "read-only range ", resp, err)
}

// callers need make sure time has passed warningApplyDuration
func warnOfExpensiveGenericRequest(lg *zap.Logger, slowApplies prometheus.Counter, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, prefix string, resp string, err error) {
	lg.Warn(
		"apply request took too long",
		zap.Duration("took", time.Since(now)),
		zap.Duration("expected-duration", warningApplyDuration),
		zap.String("prefix", prefix),
		zap.String("request", reqStringer.String()),
		zap.String("response", resp),
		zap.Error(err),
	)
	slowApplies.Inc()
}

func isNil(msg proto.Message) bool {
	return msg == nil || reflect.ValueOf(msg).IsNil()
}

func getPrefix(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] < 0xff {
			end[i] = end[i] + 1
			end = end[:i+1]
			return end
		}
	}
	// next prefix does not exist (e.g., 0xffff);
	// default to WithFromKey policy
	return noPrefixEnd
}

func rangePrefix(r *pb.RegionRangeRequest) {
	if len(r.Key) == 0 {
		r.Key, r.RangeEnd = []byte{0}, []byte{0}
		return
	}
	r.RangeEnd = getPrefix(r.Key)
}

type SV interface {
	[]byte | string
}

func toGenericMap[V SV](in map[string]any) map[string]V {
	out := make(map[string]V)
	for key, value := range in {
		var vv V
		switch tv := value.(type) {
		case string:
			vv = V(tv)
		case []byte:
			vv = V(tv)
		case *[]byte:
			vv = V(*tv)
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64:
			vv = V(fmt.Sprintf("%d", tv))
		case float32, float64:
			vv = V(fmt.Sprintf("%f", tv))
		case bool:
			vv = V("true")
			if !tv {
				vv = V("false")
			}
		default:
			data, _ := json.Marshal(tv)
			vv = V(data)
		}
		out[key] = vv
	}
	return out
}

func toAnyMap[V any](in map[string]V) map[string]any {
	out := make(map[string]any)
	for key, value := range in {
		out[key] = value
	}
	return out
}
