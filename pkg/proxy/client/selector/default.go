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

package selector

import (
	"context"
	"errors"
	"time"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/discovery/cache"
)

type registrySelector struct {
	so Options
	rc cache.Cache
}

func (c *registrySelector) newCache() cache.Cache {
	ropts := []cache.Option{}
	if c.so.Context != nil {
		if t, ok := c.so.Context.Value("selector_ttl").(time.Duration); ok {
			ropts = append(ropts, cache.WithTTL(t))
		}
	}
	return cache.New(c.so.Discovery, ropts...)
}

func (c *registrySelector) Init(opts ...Option) error {
	for _, o := range opts {
		o(&c.so)
	}

	c.rc.Stop()
	c.rc = c.newCache()

	return nil
}

func (c *registrySelector) Options() Options {
	return c.so
}

func (c *registrySelector) Select(service string, opts ...SelectOption) (Next, error) {
	sopts := SelectOptions{
		Strategy: c.so.Strategy,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	// get the service
	// try the cache first
	// if that fails go directly to the registry
	services, err := c.rc.GetService(context.TODO(), service)
	if err != nil {
		if errors.Is(err, discovery.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// apply the filters
	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	// if there's nothing left, return
	if len(services) == 0 {
		return nil, ErrNoneAvailable
	}

	return sopts.Strategy(services), nil
}

func (c *registrySelector) Mark(service string, node *dsypb.Node, err error) {}

func (c *registrySelector) Reset(service string) {}

// Close stops the watcher and destroys the cache
func (c *registrySelector) Close() error {
	c.rc.Stop()

	return nil
}

func NewSelector(opts ...Option) (ISelector, error) {
	sopts := Options{
		Strategy: Random,
	}

	for _, opt := range opts {
		opt(&sopts)
	}

	if sopts.Discovery == nil {
		return nil, discovery.ErrNoDiscovery
	}

	s := &registrySelector{
		so: sopts,
	}
	s.rc = s.newCache()

	return s, nil
}
