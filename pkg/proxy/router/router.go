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

package router

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/context/metadata"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/discovery/cache"
	"github.com/olive-io/olive/pkg/proxy/api"
	"github.com/olive-io/olive/pkg/proxy/httprule"
	"github.com/olive-io/olive/pkg/proxy/resolver"
)

var (
	ErrClosed     = errors.New("router closed")
	ErrNotFound   = errors.New("not found")
	ErrNoResolver = errors.New("no resolver")
)

// Router is used to determine an endpoint for a request
type Router interface {
	// Endpoint returns an api.Service endpoint or an error if it does not exist
	Endpoint(r *http.Request) (*api.Service, error)
	// Register endpoint in router
	Register(ep *api.Endpoint) error
	// Deregister endpoint from router
	Deregister(ep *api.Endpoint) error
	// Route returns an api.Service route
	Route(r *http.Request) (*api.Service, error)
	// Close stop the router
	Close() error
}

// endpoint struct, that holds compiled pcre
type endpoint struct {
	hostregs []*regexp.Regexp
	pathregs []httprule.Pattern
	pcreregs []*regexp.Regexp
}

// router is the default router
type registryRouter struct {
	ctx    context.Context
	cancel context.CancelFunc

	lg *zap.Logger

	discovery dsy.IDiscovery
	resolvers map[string]resolver.IResolver

	// registry cache
	rc cache.Cache

	sync.RWMutex
	eps map[string]*api.Service
	// compiled regexp for host and path
	ceps map[string]*endpoint

	stopc chan struct{}
}

// NewRouter returns the default router
func NewRouter(lg *zap.Logger, discovery dsy.IDiscovery) Router {
	if lg == nil {
		lg = zap.NewNop()
	}

	rc := cache.New(discovery)

	resolvers := map[string]resolver.IResolver{
		api.RPCHandler:  resolver.NewRPCResolver(),
		api.HTTPHandler: resolver.NewHTTPResolver(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &registryRouter{
		ctx:       ctx,
		cancel:    cancel,
		lg:        lg,
		rc:        rc,
		discovery: discovery,
		resolvers: resolvers,
		eps:       make(map[string]*api.Service),
		ceps:      make(map[string]*endpoint),
		stopc:     make(chan struct{}),
	}
	go r.watch()
	go r.refresh()
	return r
}

func (r *registryRouter) isClosed() bool {
	select {
	case <-r.stopc:
		return true
	default:
		return false
	}
}

// refresh list of api services
func (r *registryRouter) refresh() {
	var attempts int

	lg := r.lg
	ctx := r.ctx
	for {
		services, err := r.discovery.ListServices(ctx)
		if err != nil {
			attempts++
			lg.Error("unable to list services", zap.Error(err))
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		}

		attempts = 0

		// for each service, get service and store endpoints
		for _, s := range services {
			service, err := r.rc.GetService(ctx, s.Name)
			if err != nil {
				lg.Error("unable to get service", zap.Error(err))
				continue
			}
			r.store(service)
		}

		// refresh list in 3 minutes... cruft
		// use registry watching
		select {
		case <-time.After(time.Minute * 3):
		case <-r.stopc:
			return
		}
	}
}

// process watch event
func (r *registryRouter) process(res *dsypb.Result) {
	// skip these things
	if res == nil || res.Service == nil {
		return
	}

	// get entry from cache
	service, err := r.rc.GetService(r.ctx, res.Service.Name)
	if err != nil {
		r.lg.Error("unable to get service",
			zap.String("service", res.Service.Name),
			zap.Error(err))
		return
	}

	// update our local endpoints
	r.store(service)
}

// store local endpoint cache
func (r *registryRouter) store(services []*dsypb.Service) {
	// endpoints
	eps := map[string]*api.Service{}

	// services
	names := map[string]bool{}

	// create a new endpoint mapping
	for _, service := range services {
		// set names we need later
		names[service.Name] = true

		// map per endpoint
		for _, sep := range service.Endpoints {
			// create a key service:endpoint_name
			key := fmt.Sprintf("%s.%s", service.Name, sep.Name)
			// decode endpoint
			end := api.Decode(sep.Metadata)
			end.Request = sep.Request
			end.Response = sep.Response

			// if we got nothing skip
			if err := api.Validate(end); err != nil {
				r.lg.Debug("endpoint validation failed", zap.Error(err))
				continue
			}

			// try to get endpoint
			ep, ok := eps[key]
			if !ok {
				ep = &api.Service{Name: service.Name}
			}

			// overwrite the endpoint
			ep.Endpoint = end
			// append services
			ep.Services = append(ep.Services, service)
			// store it
			eps[key] = ep
		}
	}

	r.Lock()
	defer r.Unlock()

	// delete any existing eps for services we know
	for key, service := range r.eps {
		// skip what we don't care about
		if !names[service.Name] {
			continue
		}

		// ok we know this thing
		// delete delete delete
		delete(r.eps, key)
	}

	// now set the eps we have
	for name, ep := range eps {
		r.eps[name] = ep
		cep := &endpoint{}

		for _, h := range ep.Endpoint.Host {
			if h == "" || h == "*" {
				continue
			}
			hostreg, err := regexp.CompilePOSIX(h)
			if err != nil {
				r.lg.Debug("endpoint have invalid host regexp", zap.Error(err))
				continue
			}
			cep.hostregs = append(cep.hostregs, hostreg)
		}

		for _, p := range ep.Endpoint.Path {
			var pcreok bool

			if p[0] == '^' && p[len(p)-1] == '$' {
				pcrereg, err := regexp.CompilePOSIX(p)
				if err == nil {
					cep.pcreregs = append(cep.pcreregs, pcrereg)
					pcreok = true
				}
			}

			rule, err := httprule.Parse(p)
			if err != nil && !pcreok {
				r.lg.Debug("endpoint have invalid path pattern", zap.Error(err))
				continue
			} else if err != nil && pcreok {
				continue
			}

			tpl := rule.Compile()
			pathreg, err := httprule.NewPattern(tpl.Version, tpl.OpCodes, tpl.Pool, "")
			if err != nil {
				r.lg.Debug("endpoint have invalid path pattern", zap.Error(err))
				continue
			}
			cep.pathregs = append(cep.pathregs, pathreg)
		}

		r.ceps[name] = cep
	}
}

// watch for endpoint changes
func (r *registryRouter) watch() {
	var attempts int

	for {
		if r.isClosed() {
			return
		}

		// watch for changes
		w, err := r.discovery.Watch(r.ctx)
		if err != nil {
			attempts++
			r.lg.Error("error watching endpoints", zap.Error(err))
			time.Sleep(time.Duration(attempts) * time.Second)
			continue
		}

		ch := make(chan bool)

		go func() {
			select {
			case <-ch:
				w.Stop()
			case <-r.stopc:
				w.Stop()
			}
		}()

		// reset if we get here
		attempts = 0

		for {
			// process next event
			res, err := w.Next()
			if err != nil {
				r.lg.Error("error getting next endpoint", zap.Error(err))
				close(ch)
				break
			}
			r.process(res)
		}
	}
}

func (r *registryRouter) Register(ep *api.Endpoint) error {
	return nil
}

func (r *registryRouter) Deregister(ep *api.Endpoint) error {
	return nil
}

func (r *registryRouter) Endpoint(req *http.Request) (*api.Service, error) {
	if r.isClosed() {
		return nil, ErrClosed
	}

	r.RLock()
	defer r.RUnlock()

	logger := r.lg.Sugar()

	var idx int
	path := req.URL.Path
	if len(path) > 0 && path != "/" {
		idx = 1
	}
	paths := strings.Split(path[idx:], "/")

	// use the first match
	// TODO: weighted matching
	for n, e := range r.eps {
		cep, ok := r.ceps[n]
		if !ok {
			continue
		}
		ep := e.Endpoint
		var mMatch, hMatch, pMatch bool
		// 1. try method
		for _, m := range ep.Method {
			if m == req.Method {
				mMatch = true
				break
			}
		}
		if !mMatch {
			continue
		}
		logger.Debugf("api method match %s", req.Method)

		// 2. try host
		if len(ep.Host) == 0 {
			hMatch = true
		} else {
			for idx, h := range ep.Host {
				if h == "" || h == "*" {
					hMatch = true
					break
				} else {
					if cep.hostregs[idx].MatchString(req.Host) {
						hMatch = true
						break
					}
				}
			}
		}
		if !hMatch {
			continue
		}
		logger.Debugf("api host match %s", req.Host)

		// 3. try path via google.api path matching
		for _, pathreg := range cep.pathregs {
			matches, err := pathreg.Match(paths, "")
			if err != nil {
				logger.Debugf("api gpath not match %s != %v", path, pathreg)
				continue
			}
			logger.Debugf("api gpath match %s = %v", path, pathreg)
			pMatch = true
			ctx := req.Context()
			md, ok := metadata.FromContext(ctx)
			if !ok {
				md = make(metadata.Metadata)
			}
			for k, v := range matches {
				md.Set("x-api-field-"+k, v)
			}
			//TODO: parse request
			// md.Set("x-api-body", ep.Body)
			md.Set("x-api-body", "*")
			*req = *req.Clone(metadata.NewContext(ctx, md))
			break
		}

		if !pMatch {
			// 4. try path via pcre path matching
			for _, pathreg := range cep.pcreregs {
				if !pathreg.MatchString(req.URL.Path) {
					logger.Debugf("api pcre path not match %s != %v", path, pathreg)
					continue
				}
				logger.Debugf("api pcre path match %s != %v", path, pathreg)
				pMatch = true
				break
			}
		}

		if !pMatch {
			continue
		}

		// TODO: Percentage traffic
		// we got here, so its a match
		return e, nil
	}

	// no match
	return nil, ErrNotFound
}

func (r *registryRouter) Route(req *http.Request) (*api.Service, error) {
	if r.isClosed() {
		return nil, ErrClosed
	}

	handler := req.Header.Get(api.OliveHttpKey(api.HandlerKey))
	rsv, ok := r.resolvers[handler]
	if !ok {
		return nil, ErrNoResolver
	}

	switch handler {
	case api.RPCHandler:
		return r.routeRPC(req, rsv)
	case api.HTTPHandler:
		return r.routeHTTP(req, rsv)
	default:
		return nil, ErrNotFound
	}
}

func (r *registryRouter) routeRPC(req *http.Request, rsv resolver.IResolver) (*api.Service, error) {
	// try to get an endpoint
	ep, err := r.Endpoint(req)
	if err == nil {
		req.Header.Set("Content-Type", "application/grpc+json")
		return ep, nil
	}

	// get the service name
	rp, err := rsv.Resolve(req)
	if err != nil {
		return nil, err
	}

	// service name
	name := rp.Name
	var opts []dsy.GetOption

	ctx := req.Context()
	// get service
	services, err := r.rc.GetService(ctx, name, opts...)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/proto")
	return &api.Service{
		Name: name,
		Endpoint: &api.Endpoint{
			Name:     rp.Method,
			Handler:  rp.Handler,
			Request:  &dsypb.Box{Type: dsypb.BoxType_object},
			Response: &dsypb.Box{Type: dsypb.BoxType_object},
		},
		Services: services,
	}, nil
}

func (r *registryRouter) routeHTTP(req *http.Request, rsv resolver.IResolver) (*api.Service, error) {
	// try to get an endpoint
	ep, err := r.Endpoint(req)
	if err == nil {
		return ep, nil
	}

	// get the service name
	rp, err := rsv.Resolve(req)
	if err != nil {
		return nil, err
	}

	// service name
	name := rp.Name
	var opts []dsy.GetOption

	ctx := req.Context()
	// get service
	services, err := r.rc.GetService(ctx, name, opts...)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	service := &api.Service{
		Name: name,
		Endpoint: &api.Endpoint{
			Name:    rp.Path,
			Host:    []string{req.Host},
			Method:  []string{req.Method},
			Path:    []string{req.URL.Path},
			Handler: rp.Handler,
		},
		Services: services,
	}

	return service, nil
}

func (r *registryRouter) Close() error {
	select {
	case <-r.stopc:
		return nil
	default:
		close(r.stopc)
		r.rc.Stop()
		r.cancel()
	}
	return nil
}
