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

package parallelize

import (
	"context"
	"errors"
	"testing"
)

func TestErrorChannel(t *testing.T) {
	errCh := NewErrorChannel()

	if actualErr := errCh.ReceiveError(); actualErr != nil {
		t.Errorf("expect nil from err channel, but got %v", actualErr)
	}

	err := errors.New("unknown error")
	errCh.SendError(err)
	if actualErr := errCh.ReceiveError(); actualErr != err {
		t.Errorf("expect %v from err channel, but got %v", err, actualErr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh.SendErrorWithCancel(err, cancel)
	if actualErr := errCh.ReceiveError(); actualErr != err {
		t.Errorf("expect %v from err channel, but got %v", err, actualErr)
	}

	if ctxErr := ctx.Err(); ctxErr != context.Canceled {
		t.Errorf("expect context canceled, but got %v", ctxErr)
	}
}
