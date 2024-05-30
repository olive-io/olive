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

import "context"

// ErrorChannel supports non-blocking send and receive operation to capture error.
// A maximum of one error is kept in the channel and the rest of the errors sent
// are ignored, unless the existing error is received and the channel becomes empty
// again.
type ErrorChannel struct {
	errCh chan error
}

// SendError sends an error without blocking the sender.
func (e *ErrorChannel) SendError(err error) {
	select {
	case e.errCh <- err:
	default:
	}
}

// SendErrorWithCancel sends an error without blocking the sender and calls
// cancel function.
func (e *ErrorChannel) SendErrorWithCancel(err error, cancel context.CancelFunc) {
	e.SendError(err)
	cancel()
}

// ReceiveError receives an error from channel without blocking on the receiver.
func (e *ErrorChannel) ReceiveError() error {
	select {
	case err := <-e.errCh:
		return err
	default:
		return nil
	}
}

// NewErrorChannel returns a new ErrorChannel.
func NewErrorChannel() *ErrorChannel {
	return &ErrorChannel{
		errCh: make(chan error, 1),
	}
}
