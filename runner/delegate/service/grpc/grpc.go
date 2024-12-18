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
	"bytes"
	"context"
	"fmt"
	"strings"

	json "github.com/bytedance/sonic"
	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	corev1 "github.com/olive-io/olive/api/types/core/v1"
	"github.com/olive-io/olive/runner/delegate"
)

var _ delegate.Delegate = (*DelegateForGRPC)(nil)

type DelegateForGRPC struct{}

func New() *DelegateForGRPC {
	dg := &DelegateForGRPC{}
	return dg
}

func (dg *DelegateForGRPC) GetTheme() delegate.Theme {
	return delegate.Theme{
		Major: corev1.ServiceTask,
		Minor: "grpc",
	}
}

func (dg *DelegateForGRPC) Call(ctx context.Context, req *delegate.Request) (*delegate.Response, error) {
	headers := map[string]string{}

	var host, name string
	for key, value := range req.Headers {
		match := false
		if name, match = delegate.FetchHeader(key); match {
			switch strings.ToLower(name) {
			case "host":
				host = value
			case "name":
				name = value
			}
		} else {
			headers[key] = value
		}
	}

	if host == "" {
		return nil, fmt.Errorf("missing host field")
	}
	if name == "" {
		return nil, fmt.Errorf("missing name field")
	}

	timeout := req.Timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := NewClient(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", host, err)
	}

	in := req.Properties
	result := map[string]any{}
	grpcStatus, err := conn.Call(ctx, name, headers, in, &result)
	if err != nil {
		return nil, fmt.Errorf("call %s: %w", name, err)
	}

	if grpcStatus.Code() != codes.OK {
		return nil, fmt.Errorf("call %s: %w", name, grpcStatus.Err())
	}

	dresp := delegate.Response{
		Result:      result,
		DataObjects: map[string]any{},
	}

	return &dresp, nil
}

type Client struct {
	source grpcurl.DescriptorSource
	cc     *grpc.ClientConn
}

func NewClient(ctx context.Context, host string) (*Client, error) {
	network := "tcp"
	var creds credentials.TransportCredentials
	options := []grpc.DialOption{}

	cc, err := grpcurl.BlockingDial(ctx, network, host, creds, options...)
	if err != nil {
		return nil, err
	}

	refCtx := metadata.NewIncomingContext(ctx, metadata.Pairs())
	refClient := grpcreflect.NewClientAuto(refCtx, cc)
	refClient.AllowMissingFileDescriptors()
	rs := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	client := &Client{source: rs, cc: cc}
	return client, nil
}

//func (c *Client) ListServices(ctx context.Context) ([]string, error) {
//	svcs, err := c.source.ListServices()
//	return svcs, err
//}
//
//func (c *Client) FindSymbol(ctx context.Context, name string) (desc.Descriptor, error) {
//	if name[0] == '.' {
//		name = name[1:]
//	}
//
//	symbol, err := c.source.FindSymbol(name)
//	if err != nil {
//		return nil, err
//	}
//	return symbol, nil
//}

func (c *Client) Call(ctx context.Context, name string, headers map[string]string, req, resp any) (*status.Status, error) {
	options := grpcurl.FormatOptions{
		EmitJSONDefaultFields: true,
		IncludeTextSeparator:  true,
		AllowUnknownFields:    true,
	}
	format := "json"

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	in := bytes.NewBuffer(data)
	rf, formatter, err := grpcurl.RequestParserAndFormatter(grpcurl.Format(format), c.source, in, options)
	if err != nil {
		return nil, err
	}
	verbosityLevel := 0

	out := bytes.NewBufferString("")
	h := &grpcurl.DefaultEventHandler{
		Out:            out,
		Formatter:      formatter,
		VerbosityLevel: verbosityLevel,
	}

	headerSlice := []string{}
	for key, value := range headers {
		headerSlice = append(headerSlice, key+":"+value)
	}

	err = grpcurl.InvokeRPC(ctx, c.source, c.cc, name, headerSlice, h, rf.Next)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(out.Bytes(), resp); err != nil {
		return nil, err
	}

	return h.Status, nil
}
