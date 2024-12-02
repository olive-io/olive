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

// Package memory provides an in-memory registry
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
	dsy "github.com/olive-io/olive/x/discovery"
)

var (
	sendEventTime = 10 * time.Millisecond
	ttlPruneTime  = time.Second
)

type node struct {
	*dsypb.Node
	TTL      time.Duration
	LastSeen time.Time
}

type record struct {
	Name      string
	Version   string
	Metadata  map[string]string
	Nodes     map[string]*node
	Endpoints []*dsypb.Endpoint
}

type Registrar struct {
	options dsy.Options

	sync.RWMutex
	records  map[string]map[string]*record
	watchers map[string]*Watcher
}

func NewRegistrar(opts ...dsy.Option) dsy.IDiscovery {
	options := dsy.Options{}

	for _, o := range opts {
		o(&options)
	}

	if options.Logger == nil {
		options.Logger = zap.NewNop()
	}

	ctx := context.Background()
	records := getServiceRecords(ctx)
	if records == nil {
		records = make(map[string]map[string]*record)
	}

	reg := &Registrar{
		options:  options,
		records:  records,
		watchers: make(map[string]*Watcher),
	}

	go reg.ttlPrune()

	return reg
}

func (m *Registrar) ttlPrune() {
	prune := time.NewTicker(ttlPruneTime)
	defer prune.Stop()

	lg := m.options.Logger.Sugar()

	for {
		select {
		case <-prune.C:
			m.Lock()
			for name, records := range m.records {
				for version, record := range records {
					for id, n := range record.Nodes {
						if n.TTL != 0 && time.Since(n.LastSeen) > n.TTL {
							lg.Debugf("Registrar TTL expired for node %s of service %s", n.Id, name)
						}
						delete(m.records[name][version].Nodes, id)
					}
				}
			}
			m.Unlock()
		}
	}
}

func (m *Registrar) sendEvent(r *dsypb.Result) {
	m.RLock()
	watchers := make([]*Watcher, 0, len(m.watchers))
	for i := range m.watchers {
		w := m.watchers[i]
		watchers = append(watchers, w)
	}
	m.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			m.Lock()
			delete(m.watchers, w.id)
			m.Unlock()
		default:
			select {
			case w.res <- r:
			case <-time.After(sendEventTime):
			}
		}
	}
}

func (m *Registrar) Init(opts ...dsy.Option) error {
	for _, o := range opts {
		o(&m.options)
	}

	// add services
	m.Lock()
	defer m.Unlock()

	ctx := context.Background()
	records := getServiceRecords(ctx)
	for name, record := range records {
		// add a whole new service including all of its versions
		if _, ok := m.records[name]; !ok {
			m.records[name] = record
			continue
		}
		// add the versions of the service we don't track yet
		for version, r := range record {
			if _, ok := m.records[name][version]; !ok {
				m.records[name][version] = r
				continue
			}
		}
	}

	return nil
}

func (m *Registrar) Options() dsy.Options {
	return m.options
}

func (m *Registrar) Register(ctx context.Context, s *dsypb.Service, opts ...dsy.RegisterOption) error {
	m.Lock()
	defer m.Unlock()

	lg := m.options.Logger.Sugar()

	var options dsy.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	r := serviceToRecord(s, options.TTL)

	if _, ok := m.records[s.Name]; !ok {
		m.records[s.Name] = make(map[string]*record)
	}

	if _, ok := m.records[s.Name][s.Version]; !ok {
		m.records[s.Name][s.Version] = r
		lg.Debugf("Registrar added new service: %s, version: %s", s.Name, s.Version)
		go m.sendEvent(&dsypb.Result{Action: "update", Service: s})
		return nil
	}

	addedNodes := false
	for _, n := range s.Nodes {
		if _, ok := m.records[s.Name][s.Version].Nodes[n.Id]; !ok {
			addedNodes = true
			metadata := make(map[string]string)
			for k, v := range n.Metadata {
				metadata[k] = v
				m.records[s.Name][s.Version].Nodes[n.Id] = &node{
					Node: &dsypb.Node{
						Id:       n.Id,
						Address:  n.Address,
						Metadata: metadata,
					},
					TTL:      options.TTL,
					LastSeen: time.Now(),
				}
			}
		}
	}

	if addedNodes {
		lg.Debugf("Registrar added new node to service: %s, version: %s", s.Name, s.Version)
		go m.sendEvent(&dsypb.Result{Action: "update", Service: s})
		return nil
	}

	// refresh TTL and timestamp
	for _, n := range s.Nodes {
		lg.Debugf("Updated registration for service: %s, version: %s", s.Name, s.Version)
		m.records[s.Name][s.Version].Nodes[n.Id].TTL = options.TTL
		m.records[s.Name][s.Version].Nodes[n.Id].LastSeen = time.Now()
	}

	return nil
}

func (m *Registrar) Deregister(ctx context.Context, s *dsypb.Service, opts ...dsy.DeregisterOption) error {
	m.Lock()
	defer m.Unlock()

	lg := m.options.Logger.Sugar()

	if _, ok := m.records[s.Name]; ok {
		if _, ok := m.records[s.Name][s.Version]; ok {
			for _, n := range s.Nodes {
				if _, ok := m.records[s.Name][s.Version].Nodes[n.Id]; ok {
					lg.Debugf("Registrar removed node from service: %s, version: %s", s.Name, s.Version)
					delete(m.records[s.Name][s.Version].Nodes, n.Id)
				}
			}
			if len(m.records[s.Name][s.Version].Nodes) == 0 {
				delete(m.records[s.Name], s.Version)
				lg.Debugf("Registrar removed service: %s, version: %s", s.Name, s.Version)
			}
		}
		if len(m.records[s.Name]) == 0 {
			delete(m.records, s.Name)
			lg.Debugf("Registrar removed service: %s", s.Name)
		}
		go m.sendEvent(&dsypb.Result{Action: "delete", Service: s})
	}

	return nil
}

func (m *Registrar) Inject(ctx context.Context, endpoint *dsypb.Endpoint, option ...dsy.InjectOption) error {
	return nil
}

func (m *Registrar) ListEndpoints(ctx context.Context, option ...dsy.ListEndpointsOption) ([]*dsypb.Endpoint, error) {
	return make([]*dsypb.Endpoint, 0), nil
}

func (m *Registrar) GetService(ctx context.Context, name string, opts ...dsy.GetOption) ([]*dsypb.Service, error) {
	m.RLock()
	defer m.RUnlock()

	records, ok := m.records[name]
	if !ok {
		return nil, dsy.ErrNotFound
	}

	services := make([]*dsypb.Service, len(m.records[name]))
	i := 0
	for _, record := range records {
		services[i] = recordToService(record)
		i++
	}

	return services, nil
}

func (m *Registrar) ListServices(ctx context.Context, opts ...dsy.ListOption) ([]*dsypb.Service, error) {
	m.RLock()
	defer m.RUnlock()

	var services []*dsypb.Service
	for _, records := range m.records {
		for _, record := range records {
			services = append(services, recordToService(record))
		}
	}

	return services, nil
}

func (m *Registrar) Watch(ctx context.Context, opts ...dsy.WatchOption) (dsy.Watcher, error) {
	var wo dsy.WatchOptions
	for _, o := range opts {
		o(&wo)
	}

	w := &Watcher{
		exit: make(chan bool),
		res:  make(chan *dsypb.Result),
		id:   uuid.New().String(),
		wo:   wo,
	}

	m.Lock()
	m.watchers[w.id] = w
	m.Unlock()

	return w, nil
}

func (m *Registrar) String() string {
	return "memory"
}
