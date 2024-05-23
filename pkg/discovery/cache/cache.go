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

package cache

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	dsypb "github.com/olive-io/olive/apis/pb/discovery"

	"github.com/olive-io/olive/pkg/discovery"
)

// Cache is the registry cache interface
type Cache interface {
	// IDiscovery embed the registry interface
	discovery.IDiscovery
	// Stop stop the cache watcher
	Stop()
}

type Options struct {
	// TTL is the cache TTL
	TTL time.Duration
}

type Option func(o *Options)

type cache struct {
	discovery.IDiscovery
	opts Options

	// registry cache
	sync.RWMutex
	cache   map[string][]*dsypb.Service
	ttls    map[string]time.Time
	watched map[string]bool

	// used to stop the cache
	exit chan bool

	// indicate whether its running
	running bool
	// status of the registry
	// used to hold onto the cache
	// in failure state
	status error
}

var (
	DefaultTTL = time.Minute
)

func backoff(attempts int) time.Duration {
	if attempts == 0 {
		return time.Duration(0)
	}
	return time.Duration(math.Pow(10, float64(attempts))) * time.Millisecond
}

func (c *cache) getStatus() error {
	c.RLock()
	defer c.RUnlock()
	return c.status
}

func (c *cache) setStatus(err error) {
	c.Lock()
	c.status = err
	c.Unlock()
}

// isValid checks if the service is valid
func (c *cache) isValid(services []*dsypb.Service, ttl time.Time) bool {
	// no services exist
	if len(services) == 0 {
		return false
	}

	// ttl is invalid
	if ttl.IsZero() {
		return false
	}

	// time since ttl is longer than timeout
	if time.Since(ttl) > 0 {
		return false
	}

	// ok
	return true
}

func (c *cache) quit() bool {
	select {
	case <-c.exit:
		return true
	default:
		return false
	}
}

func (c *cache) del(service string) {
	// don't blow away cache in error state
	if err := c.status; err != nil {
		return
	}
	// otherwise delete entries
	delete(c.cache, service)
	delete(c.ttls, service)
}

func (c *cache) get(ctx context.Context, service string, opts ...discovery.GetOption) ([]*dsypb.Service, error) {
	// read lock
	c.RLock()

	// check the cache first
	services := c.cache[service]
	// get cache ttl
	ttl := c.ttls[service]
	// make a copy
	cp := services

	// got services && within ttl so return cache
	if c.isValid(cp, ttl) {
		c.RUnlock()
		// return services
		return cp, nil
	}

	// get does the actual request for a service and cache it
	get := func(service string, cached []*dsypb.Service, opts ...discovery.GetOption) ([]*dsypb.Service, error) {
		// ask the registry
		svcs, err := c.IDiscovery.GetService(ctx, service, opts...)
		if err != nil {
			// check the cache
			if len(cached) > 0 {
				// set the error status
				c.setStatus(err)

				// return the stale cache
				return cached, nil
			}
			// otherwise return error
			return nil, err
		}

		// reset the status
		if err = c.getStatus(); err != nil {
			c.setStatus(nil)
		}

		// cache results
		c.Lock()
		c.set(service, svcs)
		c.Unlock()

		return svcs, nil
	}

	// watch service if not watched
	_, ok := c.watched[service]

	// unlock the read lock
	c.RUnlock()

	// check if its being watched
	if !ok {
		c.Lock()

		// set to watched
		c.watched[service] = true

		// only kick it off if not running
		if !c.running {
			go c.run()
		}

		c.Unlock()
	}

	// get and return services
	return get(service, cp, opts...)
}

func (c *cache) set(service string, services []*dsypb.Service) {
	c.cache[service] = services
	c.ttls[service] = time.Now().Add(c.opts.TTL)
}

func (c *cache) update(res *dsypb.Result) {
	if res == nil || res.Service == nil {
		return
	}

	c.Lock()
	defer c.Unlock()

	// only save watched services
	if _, ok := c.watched[res.Service.Name]; !ok {
		return
	}

	services, ok := c.cache[res.Service.Name]
	if !ok {
		// we're not going to cache anything
		// unless there was already a lookup
		return
	}

	if len(res.Service.Nodes) == 0 {
		switch res.Action {
		case "delete":
			c.del(res.Service.Name)
		}
		return
	}

	// existing service found
	var service *dsypb.Service
	var index int
	for i, s := range services {
		if s.Version == res.Service.Version {
			service = s
			index = i
		}
	}

	switch res.Action {
	case "create", "update":
		if service == nil {
			c.set(res.Service.Name, append(services, res.Service))
			return
		}

		// append old nodes to new service
		for _, cur := range service.Nodes {
			var seen bool
			for _, node := range res.Service.Nodes {
				if cur.Id == node.Id {
					seen = true
					break
				}
			}
			if !seen {
				res.Service.Nodes = append(res.Service.Nodes, cur)
			}
		}

		services[index] = res.Service
		c.set(res.Service.Name, services)

	case "delete":
		if service == nil {
			return
		}

		var nodes []*dsypb.Node

		// filter cur nodes to remove the dead one
		for _, cur := range service.Nodes {
			var seen bool
			for _, del := range res.Service.Nodes {
				if del.Id == cur.Id {
					seen = true
					break
				}
			}
			if !seen {
				nodes = append(nodes, cur)
			}
		}

		// still got nodes, save and return
		if len(nodes) > 0 {
			service.Nodes = nodes
			services[index] = service
			c.set(service.Name, services)
			return
		}

		// zero nodes left

		// only have one thing to delete
		// nuke the thing
		if len(services) == 1 {
			c.del(service.Name)
			return
		}

		// still have more than 1 service
		// check the version and keep what we know
		var svcs []*dsypb.Service
		for _, s := range services {
			if s.Version != service.Version {
				svcs = append(svcs, s)
			}
		}

		// save
		c.set(service.Name, svcs)
	}
}

// run starts the cache watcher loop
// it creates a new watcher if there's a problem
func (c *cache) run() {
	c.Lock()
	c.running = true
	c.Unlock()

	// reset watcher on exit
	defer func() {
		c.Lock()
		c.watched = make(map[string]bool)
		c.running = false
		c.Unlock()
	}()

	var a, b int
	lg := c.Options().Logger

	ctx := context.TODO()
	for {
		// exit early if already dead
		if c.quit() {
			return
		}

		// jitter before starting
		j := rand.Int63n(100)
		time.Sleep(time.Duration(j) * time.Millisecond)

		// create new watcher
		w, err := c.IDiscovery.Watch(ctx)
		if err != nil {
			if c.quit() {
				return
			}

			d := backoff(a)
			c.setStatus(err)

			if a > 3 {
				lg.Sugar().Debug("rcache: ", err, " backing off ", d)
				a = 0
			}

			time.Sleep(d)
			a++

			continue
		}

		// reset a
		a = 0

		// watch for events
		if err := c.watch(w); err != nil {
			if c.quit() {
				return
			}

			d := backoff(b)
			c.setStatus(err)

			if b > 3 {
				lg.Sugar().Debug("rcache: ", err, " backing off ", d)
				b = 0
			}

			time.Sleep(d)
			b++

			continue
		}

		// reset b
		b = 0
	}
}

// watch loops the next event and calls update
// it returns if there's an error
func (c *cache) watch(w discovery.Watcher) error {
	// used to stop the watch
	stop := make(chan bool)

	// manage this loop
	go func() {
		defer w.Stop()

		select {
		// wait for exit
		case <-c.exit:
			return
		// we've been stopped
		case <-stop:
			return
		}
	}()

	for {
		res, err := w.Next()
		if err != nil {
			close(stop)
			return err
		}

		// reset the error status since we succeeded
		if err := c.getStatus(); err != nil {
			// reset status
			c.setStatus(nil)
		}

		c.update(res)
	}
}

func (c *cache) GetService(ctx context.Context, service string, opts ...discovery.GetOption) ([]*dsypb.Service, error) {
	// get the service
	services, err := c.get(ctx, service, opts...)
	if err != nil {
		return nil, err
	}

	// if there's nothing return err
	if len(services) == 0 {
		return nil, discovery.ErrNotFound
	}

	// return services
	return services, nil
}

func (c *cache) Stop() {
	c.Lock()
	defer c.Unlock()

	select {
	case <-c.exit:
		return
	default:
		close(c.exit)
	}
}

// New returns a new cache
func New(dsy discovery.IDiscovery, opts ...Option) Cache {
	rand.Seed(time.Now().UnixNano())
	options := Options{
		TTL: DefaultTTL,
	}

	for _, o := range opts {
		o(&options)
	}

	return &cache{
		IDiscovery: dsy,
		opts:       options,
		watched:    make(map[string]bool),
		cache:      make(map[string][]*dsypb.Service),
		ttls:       make(map[string]time.Time),
		exit:       make(chan bool),
	}
}
