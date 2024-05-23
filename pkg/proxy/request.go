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

package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	json "github.com/json-iterator/go"
	"github.com/oxtoacart/bpool"

	dsypb "github.com/olive-io/olive/apis/pb/discovery"
	pb "github.com/olive-io/olive/apis/pb/gateway"

	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/proxy/codec"
	"github.com/olive-io/olive/pkg/proxy/codec/jsonrpc"
	"github.com/olive-io/olive/pkg/proxy/codec/protorpc"
	"github.com/olive-io/olive/pkg/qson"

	cxmd "github.com/olive-io/olive/pkg/context/metadata"
)

var (
	// supported json codecs
	jsonCodecs = []string{
		"application/grpc+json",
		"application/json",
		"application/json-rpc",
	}

	// supported proto codecs
	protoCodecs = []string{
		"application/grpc",
		"application/grpc+proto",
		"application/proto",
		"application/protobuf",
		"application/proto-rpc",
		"application/octet-stream",
	}

	// supported multipart/form-data codecs
	dataCodecs = []string{
		"multipart/form-data",
	}

	bufferPool = bpool.NewSizedBufferPool(1024, 8)
)

func hasCodec(ct string, codecs []string) bool {
	for _, c := range codecs {
		if ct == c {
			return true
		}
	}
	return false
}

type RawMessage json.RawMessage

// MarshalJSON returns m as the JSON encoding of m.
func (m *RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return *m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return fmt.Errorf("RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

type Message struct {
	data []byte
}

func NewMessage(data []byte) *Message {
	return &Message{data}
}

func (m *Message) ProtoMessage() {}

func (m *Message) Reset() {
	*m = Message{}
}

func (m *Message) String() string {
	return string(m.data)
}

func (m *Message) Marshal() ([]byte, error) {
	return m.data, nil
}

func (m *Message) Unmarshal(data []byte) error {
	m.data = data
	return nil
}

type buffer struct {
	io.ReadCloser
}

func (b *buffer) Write(_ []byte) (int, error) {
	return 0, nil
}

// newRequest builds new *http.Request from *gateway.TransmitRequest
func newRequest(req *pb.TransmitRequest) (*http.Request, error) {
	headers := req.Headers
	if _, ok := headers[api.HandlerKey]; !ok {
		headers[api.HandlerKey] = api.RPCHandler
	}

	if req.Activity.Type == dsypb.ActivityType_ServiceTask {
		taskType := req.Activity.TaskType
		switch taskType {
		case "grpc":
			headers[api.HandlerKey] = api.RPCHandler
		case "http":
			headers[api.HandlerKey] = api.HTTPHandler
		}
	}

	urlText, ok := headers[api.URLKey]
	if !ok {
		urlText = api.DefaultTaskURL
		headers[api.URLKey] = urlText
	}
	method, ok := headers[api.MethodKey]
	if !ok {
		method = http.MethodPost
		headers[api.MethodKey] = method
	}
	protocol := "HTTP/1.2"

	hurl, err := urlpkg.Parse(urlText)
	if err != nil {
		return nil, err
	}

	// set the header of *http.Request
	header := http.Header{}
	header.Set(api.OliveHttpKey(api.ActivityKey), req.Activity.Type.String())
	for key, value := range headers {
		if key == api.ContentTypeKey {
			key = "Content-Type"
		}
		if strings.HasPrefix(key, api.HeaderKeyPrefix) {
			key = api.OliveHttpKey(key)
		}
		header.Set(key, value)
	}

	hr := &http.Request{
		Method: method,
		URL:    hurl,
		Proto:  protocol,
		Header: header,
		Body:   io.NopCloser(bytes.NewBufferString("{}")),
	}

	return hr, nil
}

// requestPayload takes a *http.Request.
// If the request is a GET the query string parameters are extracted and marshaled to JSON and the raw bytes are returned.
// If the request method is a POST the request body is read and returned
func requestPayload(r *http.Request) ([]byte, error) {
	var err error

	// we have to decode json-rpc and proto-rpc because we suck
	// well actually because there's no proxy codec right now
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "application/json-rpc"):
		msg := codec.Message{
			Type:   codec.Request,
			Header: make(map[string]string),
		}

		c := jsonrpc.NewCodec(&buffer{r.Body})
		if err = c.ReadHeader(&msg, codec.Request); err != nil {
			return nil, err
		}
		var raw RawMessage
		if err = c.ReadBody(&raw); err != nil {
			return nil, err
		}
		return raw, nil
	case strings.Contains(ct, "application/proto-rpc"), strings.Contains(ct, "application/octet-stream"):
		msg := codec.Message{
			Type:   codec.Request,
			Header: make(map[string]string),
		}
		c := protorpc.NewCodec(&buffer{r.Body})
		if err = c.ReadHeader(&msg, codec.Request); err != nil {
			return nil, err
		}
		var raw Message
		if err = c.ReadBody(&raw); err != nil {
			return nil, err
		}
		return raw.Marshal()
	case strings.Contains(ct, "application/x-www-form-urlencoded"):
		// generate a new set of values from the form
		vals := make(map[string]string)
		for key, values := range r.PostForm {
			vals[key] = strings.Join(values, ",")
		}
		for key, values := range r.URL.Query() {
			vv, ok := vals[key]
			if !ok {
				vals[key] = strings.Join(values, ",")
			} else {
				vals[key] = vv + "," + strings.Join(values, ",")
			}
		}

		// marshal
		return json.Marshal(vals)
		// TODO: application/grpc
	}

	// otherwise as per usual
	rctx := r.Context()
	// dont user metadata.FromContext as it mangles names
	md, ok := cxmd.FromContext(rctx)
	if !ok {
		md = make(map[string]string)
	}

	// allocate maximum
	matches := make(map[string]interface{}, len(md))
	bodydst := ""

	// get fields from url path
	for k, v := range md {
		k = strings.ToLower(k)
		// filter own keys
		if strings.HasPrefix(k, "x-api-field-") {
			matches[strings.TrimPrefix(k, "x-api-field-")] = v
			delete(md, k)
		} else if k == "x-api-body" {
			bodydst = v
			delete(md, k)
		}
	}

	// get fields from url values
	if len(r.URL.RawQuery) > 0 {
		umd := make(map[string]interface{})
		err = qson.Unmarshal(&umd, r.URL.RawQuery)
		if err != nil {
			return nil, err
		}
		for k, v := range umd {
			matches[k] = v
		}
	}

	// restore context without fields
	*r = *r.Clone(cxmd.NewContext(rctx, md))

	// map of all fields
	fields := nestField(matches)
	pathbuf := []byte("{}")
	if len(fields) > 0 {
		pathbuf, err = json.Marshal(fields)
		if err != nil {
			return nil, err
		}
	}

	urlbuf := []byte("{}")
	out, err := jsonpatch.MergeMergePatches(urlbuf, pathbuf)
	if err != nil {
		return nil, err
	}

	switch r.Method {
	case "GET":
		// empty response
		if strings.Contains(ct, "application/json") && string(out) == "{}" {
			return out, nil
		} else if string(out) == "{}" && !strings.Contains(ct, "application/json") {
			return []byte{}, nil
		}
		return out, nil
	case "PATCH", "POST", "PUT", "DELETE":
		bodybuf := []byte("{}")
		buf := bufferPool.Get()
		defer bufferPool.Put(buf)
		if _, err = buf.ReadFrom(r.Body); err != nil {
			return nil, err
		}
		if b := buf.Bytes(); len(b) > 0 {
			bodybuf = b
		}
		if bodydst == "" || bodydst == "*" {
			if out, err = jsonpatch.MergeMergePatches(out, bodybuf); err == nil {
				return out, nil
			}
		}
		var jsonbody map[string]interface{}
		if json.Valid(bodybuf) {
			if err = json.Unmarshal(bodybuf, &jsonbody); err != nil {
				return nil, err
			}
		}
		dstmap := make(map[string]interface{})
		ps := strings.Split(bodydst, ".")
		if len(ps) == 1 {
			if jsonbody != nil {
				dstmap[ps[0]] = jsonbody
			} else {
				// old unexpected behaviour
				dstmap[ps[0]] = bodybuf
			}
		} else {
			em := make(map[string]interface{})
			if jsonbody != nil {
				em[ps[len(ps)-1]] = jsonbody
			} else {
				// old unexpected behaviour
				em[ps[len(ps)-1]] = bodybuf
			}
			for i := len(ps) - 2; i > 0; i-- {
				nm := make(map[string]interface{})
				nm[ps[i]] = em
				em = nm
			}
			dstmap[ps[0]] = em
		}

		bodyout, err := json.Marshal(dstmap)
		if err != nil {
			return nil, err
		}

		if out, err = jsonpatch.MergeMergePatches(out, bodyout); err == nil {
			return out, nil
		}

		//fallback to previous unknown behaviour
		return bodybuf, nil
	}

	return []byte("{}"), nil
}

func nestField(matches map[string]interface{}) map[string]interface{} {
	req := make(map[string]interface{})

	for k, v := range matches {
		ps := strings.Split(k, ".")
		if len(ps) == 1 {
			req[k] = v
			continue
		}

		em := req
		for i := 0; i < len(ps)-1; i++ {
			if vm, ok := em[ps[i]]; !ok {
				vmm := make(map[string]interface{})
				em[ps[i]] = vmm
				em = vmm
			} else {
				vv, ok := vm.(map[string]interface{})
				if !ok {
					em = make(map[string]interface{})
				} else {
					em = vv
				}
			}
		}

		em[ps[len(ps)-1]] = v
	}
	return req
}
