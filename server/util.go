// Copyright 2023 Lack (xingyys@gmail.com).
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

package server

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/oliveio/olive/api"
	"go.uber.org/zap"
)

func warnOfExpensiveRequest(lg *zap.Logger, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, respMsg proto.Message, err error) {
	if time.Since(now) <= warningApplyDuration {
		return
	}
	var resp string
	if !isNil(respMsg) {
		resp = fmt.Sprintf("size:%d", proto.Size(respMsg))
	}
	warnOfExpensiveGenericRequest(lg, warningApplyDuration, now, reqStringer, "", resp, err)
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

func warnOfExpensiveReadOnlyRangeRequest(lg *zap.Logger, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, rangeResponse *api.RangeResponse, err error) {
	if time.Since(now) <= warningApplyDuration {
		return
	}
	var resp string
	if !isNil(rangeResponse) {
		resp = fmt.Sprintf("range_response_count:%d size:%d", len(rangeResponse.Kvs), rangeResponse.XSize())
	}
	warnOfExpensiveGenericRequest(lg, warningApplyDuration, now, reqStringer, "read-only range ", resp, err)
}

func warnOfExpensiveReadOnlyTxnRequest(lg *zap.Logger, warningApplyDuration time.Duration, now time.Time, r *api.TxnRequest, txnResponse *api.TxnResponse, err error) {
	if time.Since(now) <= warningApplyDuration {
		return
	}
	reqStringer := api.NewLoggableTxnRequest(r)
	var resp string
	if !isNil(txnResponse) {
		var resps []string
		for _, r := range txnResponse.Responses {
			switch op := r.Response.(type) {
			case *api.ResponseOp_ResponseRange:
				if op.ResponseRange != nil {
					resps = append(resps, fmt.Sprintf("range_response_count:%d", len(op.ResponseRange.Kvs)))
				} else {
					resps = append(resps, "range_response:nil")
				}
			default:
				// only range responses should be in a read only txn request
			}
		}
		resp = fmt.Sprintf("responses:<%s> size:%d", strings.Join(resps, " "), txnResponse.XSize())
	}
	warnOfExpensiveGenericRequest(lg, warningApplyDuration, now, reqStringer, "read-only txn ", resp, err)
}

// callers need make sure time has passed warningApplyDuration
func warnOfExpensiveGenericRequest(lg *zap.Logger, warningApplyDuration time.Duration, now time.Time, reqStringer fmt.Stringer, prefix string, resp string, err error) {
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
