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
	"encoding/json"
	"fmt"
	"io"

	"github.com/olive-io/olive/x/proxy/codec"
)

type serverCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  serverRequest
	resp serverResponse
}

type serverRequest struct {
	Method string           `json:"method"`
	Params *json.RawMessage `json:"params"`
	ID     interface{}      `json:"id"`
}

type serverResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

func newServerCodec(conn io.ReadWriteCloser) *serverCodec {
	return &serverCodec{
		dec: json.NewDecoder(conn),
		enc: json.NewEncoder(conn),
		c:   conn,
	}
}

func (r *serverRequest) reset() {
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[0:0]
	}
	if r.ID != nil {
		r.ID = nil
	}
}

func (c *serverCodec) ReadHeader(m *codec.Message) error {
	c.req.reset()
	if err := c.dec.Decode(&c.req); err != nil {
		return err
	}
	m.Method = c.req.Method
	m.Id = fmt.Sprintf("%v", c.req.ID)
	c.req.ID = nil
	return nil
}

func (c *serverCodec) ReadBody(x interface{}) error {
	if x == nil {
		return nil
	}
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.req.Params, &params)
}

var null = json.RawMessage([]byte("null"))

func (c *serverCodec) Write(m *codec.Message, x interface{}) error {
	var resp serverResponse
	resp.ID = m.Id
	resp.Result = x
	if m.Error == "" {
		resp.Error = nil
	} else {
		resp.Error = m.Error
	}
	return c.enc.Encode(resp)
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}
