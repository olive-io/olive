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

package rpctypes

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server-side error
var (
	ErrGRPCBadDefinition = status.New(codes.InvalidArgument, "olive-meta: content is bad").Err()
	ErrGRPCEmptyKey      = status.New(codes.InvalidArgument, "oliveserver: key is not provided").Err()
	ErrGRPCKeyNotFound   = status.New(codes.InvalidArgument, "oliveserver: key not found").Err()
	ErrGRPCValueProvided = status.New(codes.InvalidArgument, "oliveserver: value is provided").Err()
	ErrGRPCLeaseProvided = status.New(codes.InvalidArgument, "oliveserver: lease is provided").Err()
	ErrGRPCTooManyOps    = status.New(codes.InvalidArgument, "oliveserver: too many operations in txn request").Err()
	ErrGRPCDuplicateKey  = status.New(codes.InvalidArgument, "oliveserver: duplicate key given in txn request").Err()
	ErrGRPCCompacted     = status.New(codes.OutOfRange, "oliveserver: mvcc: required revision has been compacted").Err()
	ErrGRPCFutureRev     = status.New(codes.OutOfRange, "oliveserver: mvcc: required revision is a future revision").Err()
	ErrGRPCNoSpace       = status.New(codes.ResourceExhausted, "oliveserver: mvcc: database space exceeded").Err()

	ErrGRPCLeaseNotFound    = status.New(codes.NotFound, "oliveserver: requested lease not found").Err()
	ErrGRPCLeaseExist       = status.New(codes.FailedPrecondition, "oliveserver: lease already exists").Err()
	ErrGRPCLeaseTTLTooLarge = status.New(codes.OutOfRange, "oliveserver: too large lease TTL").Err()

	ErrGRPCWatchCanceled = status.New(codes.Canceled, "oliveserver: watch canceled").Err()

	ErrGRPCMemberExist            = status.New(codes.FailedPrecondition, "oliveserver: member ID already exist").Err()
	ErrGRPCPeerURLExist           = status.New(codes.FailedPrecondition, "oliveserver: Peer URLs already exists").Err()
	ErrGRPCMemberNotEnoughStarted = status.New(codes.FailedPrecondition, "oliveserver: re-configuration failed due to not enough started members").Err()
	ErrGRPCMemberBadURLs          = status.New(codes.InvalidArgument, "oliveserver: given member URLs are invalid").Err()
	ErrGRPCMemberNotFound         = status.New(codes.NotFound, "oliveserver: member not found").Err()
	ErrGRPCMemberNotLearner       = status.New(codes.FailedPrecondition, "oliveserver: can only promote a learner member").Err()
	ErrGRPCLearnerNotReady        = status.New(codes.FailedPrecondition, "oliveserver: can only promote a learner member which is in sync with leader").Err()
	ErrGRPCTooManyLearners        = status.New(codes.FailedPrecondition, "oliveserver: too many learner members in cluster").Err()

	ErrGRPCRequestTooLarge        = status.New(codes.InvalidArgument, "oliveserver: request is too large").Err()
	ErrGRPCRequestTooManyRequests = status.New(codes.ResourceExhausted, "oliveserver: too many requests").Err()

	ErrGRPCRootUserNotExist     = status.New(codes.FailedPrecondition, "oliveserver: root user does not exist").Err()
	ErrGRPCRootRoleNotExist     = status.New(codes.FailedPrecondition, "oliveserver: root user does not have root role").Err()
	ErrGRPCUserAlreadyExist     = status.New(codes.FailedPrecondition, "oliveserver: user name already exists").Err()
	ErrGRPCUserEmpty            = status.New(codes.InvalidArgument, "oliveserver: user name is empty").Err()
	ErrGRPCUserNotFound         = status.New(codes.FailedPrecondition, "oliveserver: user name not found").Err()
	ErrGRPCRoleAlreadyExist     = status.New(codes.FailedPrecondition, "oliveserver: role name already exists").Err()
	ErrGRPCRoleNotFound         = status.New(codes.FailedPrecondition, "oliveserver: role name not found").Err()
	ErrGRPCRoleEmpty            = status.New(codes.InvalidArgument, "oliveserver: role name is empty").Err()
	ErrGRPCAuthFailed           = status.New(codes.InvalidArgument, "oliveserver: authentication failed, invalid user ID or password").Err()
	ErrGRPCPermissionNotGiven   = status.New(codes.InvalidArgument, "oliveserver: permission not given").Err()
	ErrGRPCPermissionDenied     = status.New(codes.PermissionDenied, "oliveserver: permission denied").Err()
	ErrGRPCRoleNotGranted       = status.New(codes.FailedPrecondition, "oliveserver: role is not granted to the user").Err()
	ErrGRPCPermissionNotGranted = status.New(codes.FailedPrecondition, "oliveserver: permission is not granted to the role").Err()
	ErrGRPCAuthNotEnabled       = status.New(codes.FailedPrecondition, "oliveserver: authentication is not enabled").Err()
	ErrGRPCInvalidAuthToken     = status.New(codes.Unauthenticated, "oliveserver: invalid auth token").Err()
	ErrGRPCInvalidAuthMgmt      = status.New(codes.InvalidArgument, "oliveserver: invalid auth management").Err()
	ErrGRPCAuthOldRevision      = status.New(codes.InvalidArgument, "oliveserver: revision of auth store is old").Err()

	ErrGRPCNoLeader                   = status.New(codes.Unavailable, "oliveserver: no leader").Err()
	ErrGRPCNotLeader                  = status.New(codes.FailedPrecondition, "oliveserver: not leader").Err()
	ErrGRPCLeaderChanged              = status.New(codes.Unavailable, "oliveserver: leader changed").Err()
	ErrGRPCNotCapable                 = status.New(codes.Unavailable, "oliveserver: not capable").Err()
	ErrGRPCStopped                    = status.New(codes.Unavailable, "oliveserver: server stopped").Err()
	ErrGRPCTimeout                    = status.New(codes.Unavailable, "oliveserver: request timed out").Err()
	ErrGRPCTimeoutDueToLeaderFail     = status.New(codes.Unavailable, "oliveserver: request timed out, possibly due to previous leader failure").Err()
	ErrGRPCTimeoutDueToConnectionLost = status.New(codes.Unavailable, "oliveserver: request timed out, possibly due to connection lost").Err()
	ErrGRPCTimeoutWaitAppliedIndex    = status.New(codes.Unavailable, "oliveserver: request timed out, waiting for the applied index took too long").Err()
	ErrGRPCUnhealthy                  = status.New(codes.Unavailable, "oliveserver: unhealthy cluster").Err()
	ErrGRPCCorrupt                    = status.New(codes.DataLoss, "oliveserver: corrupt cluster").Err()
	ErrGPRCNotSupportedForLearner     = status.New(codes.Unavailable, "oliveserver: rpc not supported for learner").Err()
	ErrGRPCBadLeaderTransferee        = status.New(codes.FailedPrecondition, "oliveserver: bad leader transferee").Err()

	ErrGRPCClusterVersionUnavailable     = status.New(codes.Unavailable, "oliveserver: cluster version not found during downgrade").Err()
	ErrGRPCWrongDowngradeVersionFormat   = status.New(codes.InvalidArgument, "oliveserver: wrong downgrade target version format").Err()
	ErrGRPCInvalidDowngradeTargetVersion = status.New(codes.InvalidArgument, "oliveserver: invalid downgrade target version").Err()
	ErrGRPCDowngradeInProcess            = status.New(codes.FailedPrecondition, "oliveserver: cluster has a downgrade job in progress").Err()
	ErrGRPCNoInflightDowngrade           = status.New(codes.FailedPrecondition, "oliveserver: no inflight downgrade job").Err()

	ErrGRPCCanceled         = status.New(codes.Canceled, "oliveserver: request canceled").Err()
	ErrGRPCDeadlineExceeded = status.New(codes.DeadlineExceeded, "oliveserver: context deadline exceeded").Err()

	errStringToError = map[string]error{
		ErrorDesc(ErrGRPCBadDefinition): ErrGRPCBadDefinition,
		ErrorDesc(ErrGRPCEmptyKey):      ErrGRPCEmptyKey,
		ErrorDesc(ErrGRPCKeyNotFound):   ErrGRPCKeyNotFound,
		ErrorDesc(ErrGRPCValueProvided): ErrGRPCValueProvided,
		ErrorDesc(ErrGRPCLeaseProvided): ErrGRPCLeaseProvided,

		ErrorDesc(ErrGRPCTooManyOps):   ErrGRPCTooManyOps,
		ErrorDesc(ErrGRPCDuplicateKey): ErrGRPCDuplicateKey,
		ErrorDesc(ErrGRPCCompacted):    ErrGRPCCompacted,
		ErrorDesc(ErrGRPCFutureRev):    ErrGRPCFutureRev,
		ErrorDesc(ErrGRPCNoSpace):      ErrGRPCNoSpace,

		ErrorDesc(ErrGRPCLeaseNotFound):    ErrGRPCLeaseNotFound,
		ErrorDesc(ErrGRPCLeaseExist):       ErrGRPCLeaseExist,
		ErrorDesc(ErrGRPCLeaseTTLTooLarge): ErrGRPCLeaseTTLTooLarge,

		ErrorDesc(ErrGRPCMemberExist):            ErrGRPCMemberExist,
		ErrorDesc(ErrGRPCPeerURLExist):           ErrGRPCPeerURLExist,
		ErrorDesc(ErrGRPCMemberNotEnoughStarted): ErrGRPCMemberNotEnoughStarted,
		ErrorDesc(ErrGRPCMemberBadURLs):          ErrGRPCMemberBadURLs,
		ErrorDesc(ErrGRPCMemberNotFound):         ErrGRPCMemberNotFound,
		ErrorDesc(ErrGRPCMemberNotLearner):       ErrGRPCMemberNotLearner,
		ErrorDesc(ErrGRPCLearnerNotReady):        ErrGRPCLearnerNotReady,
		ErrorDesc(ErrGRPCTooManyLearners):        ErrGRPCTooManyLearners,

		ErrorDesc(ErrGRPCRequestTooLarge):        ErrGRPCRequestTooLarge,
		ErrorDesc(ErrGRPCRequestTooManyRequests): ErrGRPCRequestTooManyRequests,

		ErrorDesc(ErrGRPCRootUserNotExist):     ErrGRPCRootUserNotExist,
		ErrorDesc(ErrGRPCRootRoleNotExist):     ErrGRPCRootRoleNotExist,
		ErrorDesc(ErrGRPCUserAlreadyExist):     ErrGRPCUserAlreadyExist,
		ErrorDesc(ErrGRPCUserEmpty):            ErrGRPCUserEmpty,
		ErrorDesc(ErrGRPCUserNotFound):         ErrGRPCUserNotFound,
		ErrorDesc(ErrGRPCRoleAlreadyExist):     ErrGRPCRoleAlreadyExist,
		ErrorDesc(ErrGRPCRoleNotFound):         ErrGRPCRoleNotFound,
		ErrorDesc(ErrGRPCRoleEmpty):            ErrGRPCRoleEmpty,
		ErrorDesc(ErrGRPCAuthFailed):           ErrGRPCAuthFailed,
		ErrorDesc(ErrGRPCPermissionDenied):     ErrGRPCPermissionDenied,
		ErrorDesc(ErrGRPCRoleNotGranted):       ErrGRPCRoleNotGranted,
		ErrorDesc(ErrGRPCPermissionNotGranted): ErrGRPCPermissionNotGranted,
		ErrorDesc(ErrGRPCAuthNotEnabled):       ErrGRPCAuthNotEnabled,
		ErrorDesc(ErrGRPCInvalidAuthToken):     ErrGRPCInvalidAuthToken,
		ErrorDesc(ErrGRPCInvalidAuthMgmt):      ErrGRPCInvalidAuthMgmt,
		ErrorDesc(ErrGRPCAuthOldRevision):      ErrGRPCAuthOldRevision,

		ErrorDesc(ErrGRPCNoLeader):                   ErrGRPCNoLeader,
		ErrorDesc(ErrGRPCNotLeader):                  ErrGRPCNotLeader,
		ErrorDesc(ErrGRPCLeaderChanged):              ErrGRPCLeaderChanged,
		ErrorDesc(ErrGRPCNotCapable):                 ErrGRPCNotCapable,
		ErrorDesc(ErrGRPCStopped):                    ErrGRPCStopped,
		ErrorDesc(ErrGRPCTimeout):                    ErrGRPCTimeout,
		ErrorDesc(ErrGRPCTimeoutDueToLeaderFail):     ErrGRPCTimeoutDueToLeaderFail,
		ErrorDesc(ErrGRPCTimeoutDueToConnectionLost): ErrGRPCTimeoutDueToConnectionLost,
		ErrorDesc(ErrGRPCUnhealthy):                  ErrGRPCUnhealthy,
		ErrorDesc(ErrGRPCCorrupt):                    ErrGRPCCorrupt,
		ErrorDesc(ErrGPRCNotSupportedForLearner):     ErrGPRCNotSupportedForLearner,
		ErrorDesc(ErrGRPCBadLeaderTransferee):        ErrGRPCBadLeaderTransferee,

		ErrorDesc(ErrGRPCClusterVersionUnavailable):     ErrGRPCClusterVersionUnavailable,
		ErrorDesc(ErrGRPCWrongDowngradeVersionFormat):   ErrGRPCWrongDowngradeVersionFormat,
		ErrorDesc(ErrGRPCInvalidDowngradeTargetVersion): ErrGRPCInvalidDowngradeTargetVersion,
		ErrorDesc(ErrGRPCDowngradeInProcess):            ErrGRPCDowngradeInProcess,
		ErrorDesc(ErrGRPCNoInflightDowngrade):           ErrGRPCNoInflightDowngrade,
	}
)

// client-side error
var (
	ErrEmptyKey      = Error(ErrGRPCEmptyKey)
	ErrKeyNotFound   = Error(ErrGRPCKeyNotFound)
	ErrValueProvided = Error(ErrGRPCValueProvided)
	ErrLeaseProvided = Error(ErrGRPCLeaseProvided)
	ErrTooManyOps    = Error(ErrGRPCTooManyOps)
	ErrDuplicateKey  = Error(ErrGRPCDuplicateKey)
	ErrCompacted     = Error(ErrGRPCCompacted)
	ErrFutureRev     = Error(ErrGRPCFutureRev)
	ErrNoSpace       = Error(ErrGRPCNoSpace)

	ErrLeaseNotFound    = Error(ErrGRPCLeaseNotFound)
	ErrLeaseExist       = Error(ErrGRPCLeaseExist)
	ErrLeaseTTLTooLarge = Error(ErrGRPCLeaseTTLTooLarge)

	ErrMemberExist            = Error(ErrGRPCMemberExist)
	ErrPeerURLExist           = Error(ErrGRPCPeerURLExist)
	ErrMemberNotEnoughStarted = Error(ErrGRPCMemberNotEnoughStarted)
	ErrMemberBadURLs          = Error(ErrGRPCMemberBadURLs)
	ErrMemberNotFound         = Error(ErrGRPCMemberNotFound)
	ErrMemberNotLearner       = Error(ErrGRPCMemberNotLearner)
	ErrMemberLearnerNotReady  = Error(ErrGRPCLearnerNotReady)
	ErrTooManyLearners        = Error(ErrGRPCTooManyLearners)

	ErrRequestTooLarge = Error(ErrGRPCRequestTooLarge)
	ErrTooManyRequests = Error(ErrGRPCRequestTooManyRequests)

	ErrRootUserNotExist     = Error(ErrGRPCRootUserNotExist)
	ErrRootRoleNotExist     = Error(ErrGRPCRootRoleNotExist)
	ErrUserAlreadyExist     = Error(ErrGRPCUserAlreadyExist)
	ErrUserEmpty            = Error(ErrGRPCUserEmpty)
	ErrUserNotFound         = Error(ErrGRPCUserNotFound)
	ErrRoleAlreadyExist     = Error(ErrGRPCRoleAlreadyExist)
	ErrRoleNotFound         = Error(ErrGRPCRoleNotFound)
	ErrRoleEmpty            = Error(ErrGRPCRoleEmpty)
	ErrAuthFailed           = Error(ErrGRPCAuthFailed)
	ErrPermissionDenied     = Error(ErrGRPCPermissionDenied)
	ErrRoleNotGranted       = Error(ErrGRPCRoleNotGranted)
	ErrPermissionNotGranted = Error(ErrGRPCPermissionNotGranted)
	ErrAuthNotEnabled       = Error(ErrGRPCAuthNotEnabled)
	ErrInvalidAuthToken     = Error(ErrGRPCInvalidAuthToken)
	ErrAuthOldRevision      = Error(ErrGRPCAuthOldRevision)
	ErrInvalidAuthMgmt      = Error(ErrGRPCInvalidAuthMgmt)

	ErrNoLeader                   = Error(ErrGRPCNoLeader)
	ErrNotLeader                  = Error(ErrGRPCNotLeader)
	ErrLeaderChanged              = Error(ErrGRPCLeaderChanged)
	ErrNotCapable                 = Error(ErrGRPCNotCapable)
	ErrStopped                    = Error(ErrGRPCStopped)
	ErrTimeout                    = Error(ErrGRPCTimeout)
	ErrTimeoutDueToLeaderFail     = Error(ErrGRPCTimeoutDueToLeaderFail)
	ErrTimeoutDueToConnectionLost = Error(ErrGRPCTimeoutDueToConnectionLost)
	ErrTimeoutWaitAppliedIndex    = Error(ErrGRPCTimeoutWaitAppliedIndex)
	ErrUnhealthy                  = Error(ErrGRPCUnhealthy)
	ErrCorrupt                    = Error(ErrGRPCCorrupt)
	ErrBadLeaderTransferee        = Error(ErrGRPCBadLeaderTransferee)

	ErrClusterVersionUnavailable     = Error(ErrGRPCClusterVersionUnavailable)
	ErrWrongDowngradeVersionFormat   = Error(ErrGRPCWrongDowngradeVersionFormat)
	ErrInvalidDowngradeTargetVersion = Error(ErrGRPCInvalidDowngradeTargetVersion)
	ErrDowngradeInProcess            = Error(ErrGRPCDowngradeInProcess)
	ErrNoInflightDowngrade           = Error(ErrGRPCNoInflightDowngrade)
)

// OliveError defines gRPC server errors.
// (https://github.com/grpc/grpc-go/blob/master/rpc_util.go#L319-L323)
type OliveError struct {
	code codes.Code
	desc string
}

// Code returns grpc/codes.Code.
// TODO: define client/codes.Code.
func (e OliveError) Code() codes.Code {
	return e.code
}

func (e OliveError) Error() string {
	return e.desc
}

func Error(err error) error {
	if err == nil {
		return nil
	}
	verr, ok := errStringToError[ErrorDesc(err)]
	if !ok { // not gRPC error
		return err
	}
	ev, ok := status.FromError(verr)
	var desc string
	if ok {
		desc = ev.Message()
	} else {
		desc = verr.Error()
	}
	return OliveError{code: ev.Code(), desc: desc}
}

func ErrorDesc(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}
