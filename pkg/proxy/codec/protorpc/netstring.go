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

package protorpc

import (
	"encoding/binary"
	"io"
)

// WriteNetString writes data to a big-endian netstring on a Write.
// Size is always a 32-bit unsigned int.
func WriteNetString(w io.Writer, data []byte) (written int, err error) {
	size := make([]byte, 4)
	binary.BigEndian.PutUint32(size, uint32(len(data)))
	if written, err = w.Write(size); err != nil {
		return
	}
	return w.Write(data)
}

// ReadNetString reads data from a big-endian netstring.
func ReadNetString(r io.Reader) (data []byte, err error) {
	sizeBuf := make([]byte, 4)
	_, err = r.Read(sizeBuf)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(sizeBuf)
	if size == 0 {
		return nil, nil
	}
	data = make([]byte, size)
	_, err = r.Read(data)
	if err != nil {
		return nil, err
	}
	return
}
