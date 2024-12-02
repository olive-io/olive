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

// Package bytes provides a bytes codec which does not encode or decode anything
package bytes

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/olive-io/olive/x/proxy/codec"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

// Frame gives us the ability to define raw data to send over the pipes
type Frame struct {
	Data []byte
}

func (c *Codec) ReadHeader(m *codec.Message, t codec.MessageType) error {
	return nil
}

func (c *Codec) ReadBody(b interface{}) error {
	// read bytes
	buf, err := ioutil.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	switch v := b.(type) {
	case *[]byte:
		*v = buf
	case *Frame:
		v.Data = buf
	default:
		return fmt.Errorf("failed to read body: %v is not type of *[]byte", b)
	}

	return nil
}

func (c *Codec) Write(m *codec.Message, b interface{}) error {
	var v []byte
	switch vb := b.(type) {
	case *Frame:
		v = vb.Data
	case *[]byte:
		v = *vb
	case []byte:
		v = vb
	default:
		return fmt.Errorf("failed to write: %v is not type of *[]byte", b)
	}
	_, err := c.Conn.Write(v)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}

func (c *Codec) String() string {
	return "bytes"
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}
