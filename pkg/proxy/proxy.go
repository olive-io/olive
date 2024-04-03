// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	jsonpatch "github.com/evanphx/json-patch/v5"
	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/api/gatewaypb"
	cx "github.com/olive-io/olive/pkg/context"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/proxy/client"
	"github.com/olive-io/olive/pkg/proxy/client/grpc"
	"github.com/olive-io/olive/pkg/proxy/client/selector"
	"github.com/olive-io/olive/pkg/proxy/router"
)

var (
	ErrLargeRequest = errors.New("request too large")
)

type IProxy interface {
	Handle(ctx context.Context, req *gatewaypb.TransmitRequest) (*gatewaypb.TransmitResponse, error)
	Stop() error
}

// strategy is a hack for selection
func strategy(services []*dsypb.Service) selector.Strategy {
	return func(_ []*dsypb.Service) selector.Next {
		// ignore input to this function, use services above
		return selector.RoundRobin(services)
	}
}

type proxy struct {
	Config

	gr  router.Router
	gsr selector.ISelector
	cc  client.IClient
}

func NewProxy(cfg Config) (IProxy, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	discovery := cfg.Discovery
	sr, err := selector.NewSelector(selector.Discovery(discovery))
	if err != nil {
		return nil, err
	}

	cc, err := grpc.NewClient(client.Discovery(discovery), client.Selector(sr))
	if err != nil {
		return nil, err
	}

	r := router.NewRouter(cfg.Logger, cfg.Discovery)
	gw := &proxy{
		Config: cfg,
		gr:     r,
		gsr:    sr,
		cc:     cc,
	}

	return gw, nil
}

func (p *proxy) Handle(ctx context.Context, req *gatewaypb.TransmitRequest) (*gatewaypb.TransmitResponse, error) {

	hr, err := newRequest(req)
	if err != nil {
		return nil, err
	}

	// set merged context to request
	service, err := p.gr.Route(hr.Clone(ctx))
	if err != nil {
		return nil, err
	}

	if len(service.Services) == 0 {
		return nil, router.ErrNotFound
	}
	ep := service.Endpoint

	so := selector.WithStrategy(strategy(service.Services))
	callOpts := []client.CallOption{
		client.WithSelectOption(so),
	}

	if ep.Name == api.DefaultTaskURL {
		rsp := &gatewaypb.TransmitResponse{}
		cr := p.cc.NewRequest(
			service.Name,
			service.Endpoint.Name,
			req,
		)

		// make the call
		if err = p.cc.Call(ctx, cr, rsp, callOpts...); err != nil {
			return nil, err
		}
		return rsp, nil
	}

	hr = hr.WithContext(ctx)
	// create context
	ctx = cx.FromRequest(hr)
	hr.WithContext(ctx)
	//for k, v := range h.opts.Metadata {
	//	cx = metadata.Set(cx, k, v)
	//}

	ct := hr.Header.Get("Content-Type")
	// Strip charset from Content-Type (like `application/json; charset=UTF-8`)
	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

	cc := p.cc
	requestBox, responseBox := ep.Request, ep.Response
	br, err := EncodeBox(requestBox)
	if err != nil {
		return nil, err
	}
	// walk the standard call path
	// get payload
	payload, err := requestPayload(hr)
	if err != nil {
		return nil, err
	}
	br, err = jsonpatch.MergePatch(br, payload)
	if err != nil {
		return nil, err
	}

	bsize := DefaultMaxRecvSize
	if p.MaxRecvSize > 0 {
		bsize = p.MaxRecvSize
	}

	if lsize := int64(len(br)); lsize > bsize {
		return nil, errors.Wrapf(ErrLargeRequest, "request(%d) > limit(%d)", lsize, bsize)
	}

	var out []byte

	if handler := req.Headers[api.HandlerKey]; handler == api.HTTPHandler {
		//TODO: handles http request
	}

	switch {
	// proto codecs
	case hasCodec(ct, protoCodecs):
		request := &Message{}
		// if the extracted payload isn't empty lets use it
		if len(br) > 0 {
			request = NewMessage(br)
		}

		rsp := &Message{}
		rr := cc.NewRequest(
			service.Name,
			ep.Name,
			request,
			client.WithContentType(ct),
		)

		// make the call
		if err = cc.Call(ctx, rr, rsp, callOpts...); err != nil {
			return nil, err
		}

		// marshall response
		out, err = rsp.Marshal()
		if err != nil {
			return nil, err
		}

	default:
		// if json codec is not present set to json
		if !hasCodec(ct, jsonCodecs) {
			ct = "application/json"
		}

		// default to trying json
		var request RawMessage
		// if the extracted payload isn't empty lets use it
		if len(br) > 0 {
			request = br
		}

		// create request/response
		var rsp RawMessage
		rr := cc.NewRequest(
			service.Name,
			ep.Name,
			&request,
			client.WithContentType(ct),
		)
		// make the call
		if err = cc.Call(ctx, rr, &rsp, callOpts...); err != nil {
			return nil, err
		}

		// marshall response
		out, err = rsp.MarshalJSON()
		if err != nil {
			return nil, err
		}
	}

	if err = DecodeBox(out, responseBox); err != nil {
		return nil, err
	}

	//TODO: *discoverypb.Box is Property or DataObject?
	response := &gatewaypb.TransmitResponse{
		Properties:  make(map[string]*dsypb.Box),
		DataObjects: make(map[string]*dsypb.Box),
	}
	for name := range responseBox.Parameters {
		response.Properties[name] = responseBox.Parameters[name]
	}

	return response, nil
}

func (p *proxy) Stop() error {
	if err := p.gr.Close(); err != nil {
		return err
	}
	return nil
}
