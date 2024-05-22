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

package consumer

import (
	"bytes"
	"io"
	"net/http"
	urlpkg "net/url"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	jsonpatch "github.com/evanphx/json-patch/v5"
	json "github.com/json-iterator/go"

	dsypb "github.com/olive-io/olive/apis/pb/discovery"

	"github.com/olive-io/olive/pkg/proxy/api"
)

var (
	DefaultMaxSize = int64(20 * 1024 * 1024)
	DefaultTimeout = time.Second * 30
)

type HttpConsumer struct {
	ymu sync.RWMutex
	// the mapping between address and id for apidiscovery.Yard
	addrs map[string]string
	yards map[string]*dsypb.Yard
}

func NewHttpConsumer() *HttpConsumer {
	h := &HttpConsumer{
		addrs: make(map[string]string),
		yards: make(map[string]*dsypb.Yard),
	}
	return h
}

func (h *HttpConsumer) Inject(yard *dsypb.Yard) {
	h.ymu.Lock()
	defer h.ymu.Unlock()
	h.yards[yard.Id] = yard
	h.addrs[yard.Address] = yard.Id
}

// DigOut deletes the yard by id
func (h *HttpConsumer) DigOut(id string) {
	h.ymu.Lock()
	defer h.ymu.Unlock()
	yard, exists := h.yards[id]
	if !exists {
		return
	}
	delete(h.addrs, yard.Address)
	delete(h.yards, id)
}

func (h *HttpConsumer) Handle(ctx *Context) (any, error) {
	id, ok := ctx.Headers[api.OliveHttpKey(api.IdKey)]
	if !ok {
		addr := ctx.Headers[api.OliveHttpKey(api.HostKey)]
		url, err := urlpkg.Parse(addr)
		if err != nil {
			return nil, errors.Newf("invalid format of address: %s", addr)
		}
		id = h.addrs[url.Host]
	}
	h.ymu.RLock()
	yard, ok := h.yards[id]
	h.ymu.RUnlock()
	if !ok {
		return nil, errors.Newf("unknown consumer: id '%s'", id)
	}

	urlText, ok := ctx.Headers[api.OliveHttpKey(api.URLKey)]
	if !ok {
		return nil, errors.Newf("unknown consumer: url not found")
	}
	cs, ok := yard.Consumers[urlText]
	if !ok {
		return nil, errors.Newf("unknown consumer: id '%s'", urlText)
	}

	if strings.HasPrefix(urlText, "/") {
		urlText = yard.Address + urlText
	}

	url, err := urlpkg.Parse(urlText)
	if err != nil {
		return nil, errors.Newf("invalid url: %s", urlText)
	}

	responseBox := cs.Response
	req, err := h.newRequest(ctx, cs)
	if err != nil {
		return nil, errors.Wrapf(err, "build request")
	}
	req.URL = url

	hc := &http.Client{
		Transport: &http.Transport{
			MaxResponseHeaderBytes: DefaultMaxSize,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil
		},
		Timeout: DefaultTimeout,
	}
	rsp, err := hc.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "send request")
	}
	data, err := io.ReadAll(rsp.Body)
	_ = rsp.Body.Close()
	if err != nil {
		return nil, errors.Newf("read response body: %v", err)
	}
	if err = responseBox.DecodeJSON(data); err != nil {
		return nil, errors.Newf("decode response: %v", err)
	}
	return responseBox, nil
}

func (h *HttpConsumer) newRequest(ctx *Context, cs *dsypb.Consumer) (*http.Request, error) {
	method, ok := ctx.Headers[api.OliveHttpKey(api.MethodKey)]
	if !ok {
		return nil, errors.Newf("http method not found")
	}

	requestBox := cs.Request
	if requestBox.Parameters == nil {
		requestBox.Parameters = map[string]*dsypb.Box{}
	}

	br, err := requestBox.EncodeJSON()
	if err != nil {
		return nil, errors.Newf("encode request: %v", err)
	}

	payload := map[string]any{}
	for key, value := range ctx.Properties {
		if _, found := requestBox.Parameters[key]; found {
			payload[key] = value.Value()
		}
	}
	for key, value := range ctx.DataObjects {
		if _, found := requestBox.Parameters[key]; found {
			payload[key] = value.Value()
		}
	}
	data, _ := json.Marshal(payload)
	br, _ = jsonpatch.CreateMergePatch(br, data)

	header := http.Header{}
	for name, value := range ctx.Headers {
		if !strings.HasPrefix(name, api.HttpNativePrefix) {
			continue
		}
		header.Set(strings.TrimPrefix(name, api.HttpNativePrefix), value)
	}
	header.Set("Content-Type", "application/json")

	req := &http.Request{
		Method: strings.ToUpper(method),
		Header: header,
		Body:   io.NopCloser(bytes.NewBuffer(br)),
	}
	return req, nil
}
