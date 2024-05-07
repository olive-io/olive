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
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConvert(t *testing.T) {
	e1 := status.New(codes.InvalidArgument, "olive: key is not provided").Err()
	e2 := ErrGRPCEmptyKey
	e3 := ErrEmptyKey

	if e1.Error() != e2.Error() {
		t.Fatalf("expected %q == %q", e1.Error(), e2.Error())
	}
	if ev1, ok := status.FromError(e1); ok && ev1.Code() != e3.(OliveError).Code() {
		t.Fatalf("expected them to be equal, got %v / %v", ev1.Code(), e3.(OliveError).Code())
	}

	if e1.Error() == e3.Error() {
		t.Fatalf("expected %q != %q", e1.Error(), e3.Error())
	}
	if ev2, ok := status.FromError(e2); ok && ev2.Code() != e3.(OliveError).Code() {
		t.Fatalf("expected them to be equal, got %v / %v", ev2.Code(), e3.(OliveError).Code())
	}
}
