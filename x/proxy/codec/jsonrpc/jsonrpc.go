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

package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/olive-io/olive/x/proxy/codec"
)

type jsonCodec struct {
	buf *bytes.Buffer
	mt  codec.MessageType
	rwc io.ReadWriteCloser
	c   *clientCodec
	s   *serverCodec
}

func (j *jsonCodec) Close() error {
	j.buf.Reset()
	return j.rwc.Close()
}

func (j *jsonCodec) String() string {
	return "json-rpc"
}

func (j *jsonCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		return j.c.Write(m, b)
	case codec.Response:
		return j.s.Write(m, b)
	case codec.Event:
		data, err := json.Marshal(b)
		if err != nil {
			return err
		}
		_, err = j.rwc.Write(data)
		return err
	default:
		return fmt.Errorf("unrecognised message type: %v", m.Type)
	}
}

func (j *jsonCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	j.buf.Reset()
	j.mt = mt

	switch mt {
	case codec.Request:
		return j.s.ReadHeader(m)
	case codec.Response:
		return j.s.ReadBody(m)
	case codec.Event:
		_, err := io.Copy(j.buf, j.rwc)
		return err
	default:
		return fmt.Errorf("unrecognised message type: %v", mt)
	}
}

func (j *jsonCodec) ReadBody(b interface{}) error {
	switch j.mt {
	case codec.Request:
		return j.s.ReadBody(b)
	case codec.Response:
		return j.s.ReadBody(b)
	case codec.Event:
		if b != nil {
			return json.Unmarshal(j.buf.Bytes(), b)
		}
	default:
		return fmt.Errorf("unrecognised message type: %v", j.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &jsonCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
		c:   newClientCodec(rwc),
		s:   newServerCodec(rwc),
	}
}
