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

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"

	json "github.com/bytedance/sonic"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/runner/delegate"
)

var _ delegate.Delegate = (*DelegateForHttp)(nil)

type DelegateForHttp struct {
}

func New() *DelegateForHttp {
	dh := &DelegateForHttp{}
	return dh
}

func (dh *DelegateForHttp) GetTheme() delegate.Theme {
	return delegate.Theme{
		Major: types.FlowNodeType_ServiceTask,
		Minor: "http",
	}
}

func (dh *DelegateForHttp) Call(ctx context.Context, req *delegate.Request) (*delegate.Response, error) {
	timeout := req.Timeout
	transport := &http.Transport{}
	conn := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	var url *urlpkg.URL
	var contentType string
	method := http.MethodGet
	header := http.Header{}
	for name, value := range req.Headers {
		match := false
		if name, match = delegate.FetchHeader(name); match {
			switch strings.ToLower(name) {
			case "method":
				method = strings.ToUpper(value)
			case "url":
				var err error
				url, err = urlpkg.Parse(value)
				if err != nil {
					return nil, fmt.Errorf("parse url: %w", err)
				}
			}
		} else {
			header.Set(name, value)
		}

		if strings.ToLower(name) == "content-type" || strings.ToLower(name) == "content_type" {
			contentType = value
		}
	}

	if url == nil {
		return nil, fmt.Errorf("no url found")
	}

	if url.Scheme == "https" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	if contentType == "" {
		contentType = "application/json"
		header.Set("Content-Type", contentType)
	}

	var body io.Reader
	switch contentType {
	case "application/json":
		data, err := json.Marshal(req.Properties)
		if err != nil {
			return nil, fmt.Errorf("encode http body: %w", err)
		}
		body = bytes.NewBuffer(data)
	case "application/multipart-form-data":
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for key, value := range req.Properties {
			var vs string
			switch vv := value.(type) {
			case string:
				vs = vv
			case []byte:
				vs = string(vv)
			case int64:
				vs = strconv.FormatInt(vv, 10)
			case float64:
				vs = strconv.FormatFloat(vv, 'f', -1, 64)
			}
			_ = writer.WriteField(key, vs)
		}
		writer.Close()

		body = &buffer
	case "application/form-data":
		form := urlpkg.Values{}
		for key, value := range req.Properties {
			var vs string
			switch vv := value.(type) {
			case string:
				vs = vv
			case []byte:
				vs = string(vv)
			case int64:
				vs = strconv.FormatInt(vv, 10)
			case float64:
				vs = strconv.FormatFloat(vv, 'f', -1, 64)
			}
			form.Set(key, vs)
		}

		body = bytes.NewBufferString(form.Encode())
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	hr, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	hr.Header = header

	resp, err := conn.Do(hr)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	result := map[string]any{}
	dataObjects := make(map[string]any)

	result["code"] = resp.StatusCode
	result["result"] = string(data)

	dresp := &delegate.Response{
		Result:      result,
		DataObjects: dataObjects,
	}

	return dresp, nil
}
