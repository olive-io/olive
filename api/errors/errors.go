/*
Copyright 2025 The olive Authors

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

package errors

import (
	"encoding/json"
)

func (e *Error) Error() string {
	data, _ := json.Marshal(e)
	return string(data)
}

func NewErr(code Code, detail string) *Error {
	return &Error{
		Code:    code,
		Message: code.String(),
		Detail:  detail,
	}
}

func ErrUnknown(detail string) *Error {
	return NewErr(Code_Unknown, detail)
}

func ErrInternal(detail string) *Error {
	return NewErr(Code_Internal, detail)
}

func ErrNotReady(detail string) *Error {
	return NewErr(Code_NotReady, detail)
}

func ParseErr(err error) *Error {
	switch e := err.(type) {
	case *Error:
		if e.Code == Code_Ok {
			return nil
		}
		return e
	default:
		var ee *Error
		if e1 := json.Unmarshal([]byte(err.Error()), &ee); e1 == nil {
			return ee
		}

		return ErrUnknown(err.Error())
	}
}
