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

package rpctypes

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server-side error
var (
	ErrGRPCEmptyKey      = status.New(codes.InvalidArgument, "olive: key is not provided").Err()
	ErrGRPCKeyNotFound   = status.New(codes.InvalidArgument, "olive: key not found").Err()
	ErrGRPCInvalidRunner = status.New(codes.InvalidArgument, "olive: runner is invalid").Err()

	ErrGRPCNoLeader  = status.New(codes.Unavailable, "olive: no leader").Err()
	ErrGRPCNotLeader = status.New(codes.FailedPrecondition, "olive: not leader").Err()

	errStringToError = map[string]error{
		ErrorDesc(ErrGRPCEmptyKey):    ErrGRPCEmptyKey,
		ErrorDesc(ErrGRPCKeyNotFound): ErrGRPCKeyNotFound,

		ErrorDesc(ErrGRPCNoLeader):      ErrGRPCNoLeader,
		ErrorDesc(ErrGRPCNotLeader):     ErrGRPCNotLeader,
		ErrorDesc(ErrGRPCInvalidRunner): ErrGRPCInvalidRunner,
	}
)

// client-side error
var (
	ErrEmptyKey      = Error(ErrGRPCEmptyKey)
	ErrKeyNotFound   = Error(ErrGRPCKeyNotFound)
	ErrInvalidRunner = Error(ErrGRPCInvalidRunner)

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
