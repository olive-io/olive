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

package delegate

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	urlpkg "net/url"
	"strconv"
	"strings"
	"time"

	pb "github.com/olive-io/olive/api/rpc/serverpb"
)

type httpDelegate struct{}

func New() Delegate {
	hd := &httpDelegate{}
	return hd
}

func (dh *httpDelegate) Call(ctx context.Context, req *pb.CallTaskRequest) (*pb.CallTaskResponse, error) {
	timeout := time.Duration(req.Timeout) * time.Second
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
		name = strings.ToLower(name)
		switch name {
		case "content-type":
			contentType = value
		case "method":
			method = value
		case "url":
			var err error
			url, err = urlpkg.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("parse url: %w", err)
			}
		default:
			header.Set(name, value)
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
			_ = writer.WriteField(key, value)
		}
		writer.Close()

		body = &buffer
	case "application/form-data":
		form := urlpkg.Values{}
		for key, value := range req.Properties {
			form.Set(key, value)
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

	results := map[string]string{}
	dataObjects := make(map[string]string)

	results["code"] = strconv.Itoa(resp.StatusCode)
	results["result"] = string(data)

	dresp := &pb.CallTaskResponse{
		Results:     results,
		DataObjects: dataObjects,
	}

	return dresp, nil
}
