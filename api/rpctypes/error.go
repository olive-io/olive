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
	ErrGRPCInvalidRunner      = status.New(codes.InvalidArgument, "olive: runner is invalid").Err()
	ErrGRPCDefinitionNotReady = status.New(codes.FailedPrecondition, "olive: definition not ready").Err()

	ErrGRPCNoLeader  = status.New(codes.Unavailable, "olive: no leader").Err()
	ErrGRPCNotLeader = status.New(codes.FailedPrecondition, "olive: not leader").Err()

	errStringToError = map[string]error{
		ErrorDesc(ErrGRPCEmptyKey):           ErrGRPCEmptyKey,
		ErrorDesc(ErrGRPCKeyNotFound):        ErrGRPCKeyNotFound,
		ErrorDesc(ErrGRPCInvalidRunner):      ErrGRPCInvalidRunner,
		ErrorDesc(ErrGRPCDefinitionNotReady): ErrGRPCDefinitionNotReady,

		ErrorDesc(ErrGRPCNoLeader):  ErrGRPCNoLeader,
		ErrorDesc(ErrGRPCNotLeader): ErrGRPCNotLeader,
	}
)

// client-side error
var (
	ErrEmptyKey           = Error(ErrGRPCEmptyKey)
	ErrKeyNotFound        = Error(ErrGRPCKeyNotFound)
	ErrInvalidRunner      = Error(ErrGRPCInvalidRunner)
	ErrDefinitionNotReady = Error(ErrGRPCDefinitionNotReady)

	ErrNoLeader  = Error(ErrGRPCNoLeader)
	ErrNotLeader = Error(ErrGRPCNotLeader)
)

type OliveError struct {
	code codes.Code
	desc string
}

// Code returns grpc/codes.Code.
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
