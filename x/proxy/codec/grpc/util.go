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

package grpc

import (
	"encoding/binary"
	"fmt"
	"io"
)

var (
	MaxMessageSize = 1024 * 1024 * 32 // 32Mb
	maxInt         = int(^uint(0) >> 1)
)

func decode(r io.Reader) (uint8, []byte, error) {
	header := make([]byte, 5)

	// read the header
	if _, err := r.Read(header[:]); err != nil {
		return uint8(0), nil, err
	}

	// get encoding format e.g compressed
	cf := uint8(header[0])

	// get message length
	length := binary.BigEndian.Uint32(header[1:])

	// no encoding format
	if length == 0 {
		return cf, nil, nil
	}

	if int64(length) > int64(maxInt) {
		return cf, nil, fmt.Errorf("grpc: received message larger than max length allowed on current machine (%d vs. %d)", length, maxInt)
	}
	if int(length) > MaxMessageSize {
		return cf, nil, fmt.Errorf("grpc: received message larger than max (%d vs. %d)", length, MaxMessageSize)
	}

	msg := make([]byte, int64(length))

	if _, err := r.Read(msg); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return cf, nil, err
	}

	return cf, msg, nil
}

func encode(cf uint8, buf []byte, w io.Writer) error {
	header := make([]byte, 5)

	// set compress
	header[0] = byte(cf)

	// write length as header
	binary.BigEndian.PutUint32(header[1:], uint32(len(buf)))

	// read the header
	if _, err := w.Write(header[:]); err != nil {
		return err
	}

	// write the buffer
	_, err := w.Write(buf)
	return err
}
