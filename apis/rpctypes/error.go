/*
Copyright 2023 The olive Authors

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

package rpctypes

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server-side error
var (
	ErrGRPCEmptyKey           = status.New(codes.InvalidArgument, "olive: key is not provided").Err()
	ErrGRPCKeyNotFound        = status.New(codes.InvalidArgument, "olive: key not found").Err()
	ErrGRPCEmptyValue         = status.New(codes.InvalidArgument, "olive: value is not provided").Err()
	ErrGRPCInvalidRunner      = status.New(codes.InvalidArgument, "olive: runner is invalid").Err()
	ErrGRPCDefinitionNotReady = status.New(codes.FailedPrecondition, "olive: definition not ready").Err()

	ErrGRPCRequestTooLarge        = status.Error(codes.InvalidArgument, "olive: request is too large")
	ErrGRPCRequestTooManyRequests = status.Error(codes.ResourceExhausted, "olive: too many requests")

	ErrGRPCRootUserNotExist = status.Error(codes.FailedPrecondition, "olive: root user does not exist")
	ErrGRPCRootRoleNotExist = status.Error(codes.FailedPrecondition, "olive: root user does not have root role")
	ErrGRPCUserAlreadyExist = status.Error(codes.FailedPrecondition, "olive: user name already exists")
	ErrGRPCUserEmpty        = status.Error(codes.InvalidArgument, "olive: user name is empty")
	ErrGRPCUserNotFound     = status.Error(codes.FailedPrecondition, "olive: user name not found")
	ErrGRPCRoleAlreadyExist = status.Error(codes.FailedPrecondition, "olive: role name already exists")
	ErrGRPCRoleNotFound     = status.Error(codes.FailedPrecondition, "olive: role name not found")
	ErrGRPCRoleEmpty        = status.Error(codes.InvalidArgument, "olive: role name is empty")
	ErrGRPCAuthFailed       = status.Error(codes.InvalidArgument, "olive: authentication failed, invalid user ID or password")

	ErrGRPCInvalidAuthToken = status.Error(codes.Unauthenticated, "olive: invalid auth token")
	ErrGRPCInvalidAuthMgmt  = status.Error(codes.InvalidArgument, "olive: invalid auth management")

	ErrGRPCNoLeader  = status.New(codes.Unavailable, "olive: no leader").Err()
	ErrGRPCNotLeader = status.New(codes.FailedPrecondition, "olive: not leader").Err()

	errStringToError = map[string]error{
		ErrorDesc(ErrGRPCEmptyKey):           ErrGRPCEmptyKey,
		ErrorDesc(ErrGRPCKeyNotFound):        ErrGRPCKeyNotFound,
		ErrorDesc(ErrGRPCEmptyValue):         ErrGRPCEmptyValue,
		ErrorDesc(ErrGRPCInvalidRunner):      ErrGRPCInvalidRunner,
		ErrorDesc(ErrGRPCDefinitionNotReady): ErrGRPCDefinitionNotReady,

		ErrorDesc(ErrGRPCRequestTooLarge):        ErrGRPCRequestTooLarge,
		ErrorDesc(ErrGRPCRequestTooManyRequests): ErrGRPCRequestTooManyRequests,

		ErrorDesc(ErrGRPCRootUserNotExist): ErrGRPCRootUserNotExist,
		ErrorDesc(ErrGRPCRootRoleNotExist): ErrGRPCRootRoleNotExist,
		ErrorDesc(ErrGRPCUserAlreadyExist): ErrGRPCUserAlreadyExist,
		ErrorDesc(ErrGRPCUserEmpty):        ErrGRPCUserEmpty,
		ErrorDesc(ErrGRPCUserNotFound):     ErrGRPCUserNotFound,
		ErrorDesc(ErrGRPCRoleAlreadyExist): ErrGRPCRoleAlreadyExist,
		ErrorDesc(ErrGRPCRoleNotFound):     ErrGRPCRoleNotFound,
		ErrorDesc(ErrGRPCRoleEmpty):        ErrGRPCRoleEmpty,
		ErrorDesc(ErrGRPCAuthFailed):       ErrGRPCAuthFailed,

		ErrorDesc(ErrGRPCInvalidAuthToken): ErrGRPCInvalidAuthToken,
		ErrorDesc(ErrGRPCInvalidAuthMgmt):  ErrGRPCInvalidAuthMgmt,

		ErrorDesc(ErrGRPCNoLeader):  ErrGRPCNoLeader,
		ErrorDesc(ErrGRPCNotLeader): ErrGRPCNotLeader,
	}
)

// client-side error
var (
	ErrEmptyKey           = Error(ErrGRPCEmptyKey)
	ErrKeyNotFound        = Error(ErrGRPCKeyNotFound)
	ErrEmptyValue         = Error(ErrGRPCEmptyValue)
	ErrInvalidRunner      = Error(ErrGRPCInvalidRunner)
	ErrDefinitionNotReady = Error(ErrGRPCDefinitionNotReady)

	ErrRequestTooLarge = Error(ErrGRPCRequestTooLarge)
	ErrTooManyRequests = Error(ErrGRPCRequestTooManyRequests)

	ErrRootUserNotExist = Error(ErrGRPCRootUserNotExist)
	ErrRootRoleNotExist = Error(ErrGRPCRootRoleNotExist)
	ErrUserAlreadyExist = Error(ErrGRPCUserAlreadyExist)
	ErrUserEmpty        = Error(ErrGRPCUserEmpty)
	ErrUserNotFound     = Error(ErrGRPCUserNotFound)
	ErrRoleAlreadyExist = Error(ErrGRPCRoleAlreadyExist)
	ErrRoleNotFound     = Error(ErrGRPCRoleNotFound)
	ErrRoleEmpty        = Error(ErrGRPCRoleEmpty)
	ErrAuthFailed       = Error(ErrGRPCAuthFailed)

	ErrInvalidAuthToken = Error(ErrGRPCInvalidAuthToken)
	ErrInvalidAuthMgmt  = Error(ErrGRPCInvalidAuthMgmt)

	ErrNoLeader  = Error(ErrGRPCNoLeader)
	ErrNotLeader = Error(ErrGRPCNotLeader)
)

type OliveError struct {
	code codes.Code
	desc string
}

// Code returns grpc/codes.Code.
func (e *OliveError) Code() codes.Code {
	return e.code
}

func (e *OliveError) Error() string {
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
	return &OliveError{code: ev.Code(), desc: desc}
}

func ErrorDesc(err error) string {
	if s, ok := status.FromError(err); ok {
		return s.Message()
	}
	return err.Error()
}
