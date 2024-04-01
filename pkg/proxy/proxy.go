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
	json "github.com/json-iterator/go"
	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/api/gatewaypb"
	cx "github.com/olive-io/olive/pkg/context"
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

	hreq, err := newRequest(req)
	if err != nil {
		return nil, err
	}
	hreq = hreq.WithContext(ctx)

	// set merged context to request
	r := hreq.Clone(ctx)
	service, err := p.gr.Route(r)
	if err != nil {
		return nil, err
	}

	if len(service.Services) == 0 {
		return nil, router.ErrNotFound
	}

	so := selector.WithStrategy(strategy(service.Services))
	callOpts := []client.CallOption{
		client.WithSelectOption(so),
	}

	if req.Activity.Type != dsypb.ActivityType_ServiceTask {
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

	// create context
	ctx = cx.FromRequest(hreq)
	//for k, v := range h.opts.Metadata {
	//	cx = metadata.Set(cx, k, v)
	//}

	ct := hreq.Header.Get("Content-Type")
	if ct == "" {
		hreq.Header.Set("Content-Type", "application/json")
		ct = "application/json"
	}
	// Strip charset from Content-Type (like `application/json; charset=UTF-8`)
	if idx := strings.IndexRune(ct, ';'); idx >= 0 {
		ct = ct[:idx]
	}

	// walk the standard call path
	// get payload
	br, err := requestPayload(r)
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

	request := &gatewaypb.TransmitRequest{}
	if err = json.Unmarshal(br, request); err != nil {
		return nil, err
	}

	// create request/response
	rsp := &gatewaypb.TransmitResponse{}
	cr := p.cc.NewRequest(
		service.Name,
		service.Endpoint.Name,
		request,
	)

	// make the call
	if err = p.cc.Call(ctx, cr, rsp, callOpts...); err != nil {
		return nil, err
	}

	return rsp, nil
}

func (p *proxy) Stop() error {
	if err := p.gr.Close(); err != nil {
		return err
	}
	return nil
}
