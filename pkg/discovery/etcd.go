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

package discovery

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	hash "github.com/mitchellh/hashstructure"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

type etcdRegistrar struct {
	client  *clientv3.Client
	options Options

	sync.RWMutex
	register map[string]uint64
	endpoint map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func NewDiscovery(client *clientv3.Client, opts ...Option) (IDiscovery, error) {
	options := NewOptions(opts...)
	e := &etcdRegistrar{
		client:   client,
		options:  options,
		register: make(map[string]uint64),
		endpoint: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}

	for _, o := range opts {
		o(&e.options)
	}

	if e.options.Timeout == 0 {
		e.options.Timeout = 10 * time.Second
	}

	return e, nil
}

func (e *etcdRegistrar) Options() Options {
	return e.options
}

func (e *etcdRegistrar) registerNode(ctx context.Context, s *dsypb.Service, node *dsypb.Node, opts ...RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("require at lease one node")
	}

	// check existing lease cache
	e.RLock()
	leaseID, ok := e.leases[s.Name+node.Id]
	e.RUnlock()

	var options RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	prefix := e.options.Prefix
	lg := e.options.Logger

	if !ok {
		// missing lease, check if the key exists

		// look for the existing key
		rsp, err := e.client.Get(ctx, nodePath(prefix, namespace, s.Name, node.Id), clientv3.WithSerializable())
		if err != nil {
			return err
		}

		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)

				// decode the existing node
				svc := decode(kv.Value)
				if svc == nil || len(svc.Nodes) == 0 {
					continue
				}

				// create hash of service; uint64
				h, err := hash.Hash(svc.Nodes[0], nil)
				if err != nil {
					continue
				}

				// save the info
				e.Lock()
				e.leases[s.Name+node.Id] = leaseID
				e.register[s.Name+node.Id] = h
				e.Unlock()

				break
			}
		}
	}

	var leaseNotFound bool

	// renew the lease if it exists
	if leaseID > 0 {
		lg.Debug("Renewing existing lease",
			zap.String("name", s.Name),
			zap.Uint64("lease_id", uint64(leaseID)))

		if _, err := e.client.KeepAliveOnce(ctx, leaseID); err != nil {
			if !errors.Is(err, rpctypes.ErrLeaseNotFound) {
				return err
			}

			lg.Error("Lease not found",
				zap.String("name", s.Name),
				zap.Uint64("lease_id", uint64(leaseID)))
			// lease not found do register
			leaseNotFound = true
		}
	}

	// create hash of service; uint64
	h, err := hash.Hash(node, nil)
	if err != nil {
		return err
	}

	// get existing hash for the service node
	e.Lock()
	v, ok := e.register[s.Name+node.Id]
	e.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h && !leaseNotFound {
		lg.Sugar().Debugf("Service %s node %s unchanged skipping registration", s.Name, node.Id)
		return nil
	}

	if s.Namespace == "" {
		s.Namespace = namespace
	}

	service := &dsypb.Service{
		Name:      s.Name,
		Version:   s.Version,
		Namespace: s.Namespace,
		Metadata:  s.Metadata,
		Endpoints: s.Endpoints,
		Nodes:     []*dsypb.Node{node},
	}

	var lgr *clientv3.LeaseGrantResponse
	if options.TTL.Seconds() <= 0 {
		options.TTL = time.Second * 30
	}

	// get a lease used to expire keys since we have a ttl
	lgr, err = e.client.Grant(ctx, int64(options.TTL.Seconds()))
	if err != nil {
		return err
	}

	lg.Sugar().Infof("Registering '%s' namespace '%s' id '%s' with lease %v and ttl %v",
		service.Name, service.Namespace, node.Id, lgr.ID, options.TTL)
	// create an entry for the node
	if lgr.ID != 0 {
		_, err = e.client.Put(ctx, nodePath(prefix, service.Namespace, service.Name, node.Id), encode(service), clientv3.WithLease(lgr.ID))
	} else {
		_, err = e.client.Put(ctx, nodePath(prefix, service.Namespace, service.Name, node.Id), encode(service))
	}
	if err != nil {
		return err
	}

	e.Lock()
	// save our hash of the service
	e.register[s.Name+node.Id] = h
	// save our leaseID of the service
	if lgr.ID != 0 {
		e.leases[s.Name+node.Id] = lgr.ID
	}
	e.Unlock()

	return nil
}

func (e *etcdRegistrar) Register(ctx context.Context, s *dsypb.Service, opts ...RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("require at lease one node")
	}

	var grr error

	// registry each node individually
	for _, node := range s.Nodes {
		err := e.registerNode(ctx, s, node, opts...)
		if err != nil {
			grr = err
		}
	}

	return grr
}

func (e *etcdRegistrar) Deregister(ctx context.Context, s *dsypb.Service, opts ...DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("required at lease one node")
	}

	var options DeregisterOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	if s.Namespace == "" {
		s.Namespace = namespace
	}

	prefix := e.options.Prefix
	log := e.options.Logger.Sugar()
	for _, node := range s.Nodes {
		e.Lock()
		// delete our hash of the service
		delete(e.register, s.Name+node.Id)
		// delete our lease of the service
		delete(e.leases, s.Name+node.Id)
		e.Unlock()

		ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)

		log.Infof("Deregistering %s id %s", s.Name, node.Id)
		_, err := e.client.Delete(ctx, nodePath(prefix, namespace, s.Name, node.Id))
		if err != nil {
			cancel()
			return err
		}
		cancel()
	}

	return nil
}

func (e *etcdRegistrar) Inject(ctx context.Context, endpoint *dsypb.Endpoint, opts ...InjectOption) error {
	var options InjectOptions
	for _, opt := range opts {
		opt(&options)
	}

	// check existing lease cache
	sname := endpoint.Metadata["service"]
	id := options.Id
	e.RLock()
	leaseID, ok := e.leases[sname+id]
	e.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	prefix := e.options.Prefix
	lg := e.options.Logger

	if !ok {
		// missing lease, check if the key exists

		// look for the existing key
		rsp, err := e.client.Get(ctx, endpointPath(prefix, namespace, id, sname), clientv3.WithSerializable())
		if err != nil {
			return err
		}

		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)

				// decode the existing endpoints
				ep := decodeEp(kv.Value)
				if ep == nil || len(ep.Name) == 0 {
					continue
				}

				// create hash of service; uint64
				h, err := hash.Hash(ep, nil)
				if err != nil {
					continue
				}

				// save the info
				e.Lock()
				e.leases[sname] = leaseID
				e.endpoint[sname+ep.Name] = h
				e.Unlock()

				break
			}
		}
	}

	var leaseNotFound bool

	// renew the lease if it exists
	if leaseID > 0 {
		lg.Debug("Renewing existing lease",
			zap.String("name", sname),
			zap.Uint64("lease_id", uint64(leaseID)))

		if _, err := e.client.KeepAliveOnce(ctx, leaseID); err != nil {
			if !errors.Is(err, rpctypes.ErrLeaseNotFound) {
				return err
			}

			lg.Error("Lease not found",
				zap.String("name", sname),
				zap.Uint64("lease_id", uint64(leaseID)))
			// lease not found do register
			leaseNotFound = true
		}
	}

	// create hash of service; uint64
	h, err := hash.Hash(endpoint, nil)
	if err != nil {
		return err
	}

	// get existing hash for the service node
	e.Lock()
	v, ok := e.endpoint[sname+id+endpoint.Name]
	e.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h && !leaseNotFound {
		lg.Sugar().Debugf("Service %s endpoint %s unchanged skipping registration", sname, endpoint.Name)
		return nil
	}

	var lgr *clientv3.LeaseGrantResponse
	if options.TTL.Seconds() <= 0 {
		options.TTL = time.Second * 30
	}

	// get a lease used to expire keys since we have a ttl
	lgr, err = e.client.Grant(ctx, int64(options.TTL.Seconds()))
	if err != nil {
		return err
	}

	lg.Sugar().Infof("Registering '%s' with lease %v and ttl %v",
		endpoint.Name, lgr.ID, options.TTL)
	// create an entry for the node
	if lgr.ID != 0 {
		_, err = e.client.Put(ctx, endpointPath(prefix, namespace, id, endpoint.Name), encodeEp(endpoint), clientv3.WithLease(lgr.ID))
	} else {
		_, err = e.client.Put(ctx, endpointPath(prefix, namespace, id, endpoint.Name), encodeEp(endpoint))
	}
	if err != nil {
		return err
	}

	e.Lock()
	// save our hash of the endpoint
	e.endpoint[sname+id+endpoint.Name] = h
	// save our leaseID of the endpoint
	if lgr.ID != 0 {
		e.leases[sname+id] = lgr.ID
	}
	e.Unlock()

	return nil
}

func (e *etcdRegistrar) ListEndpoints(ctx context.Context, opts ...ListEndpointsOption) ([]*dsypb.Endpoint, error) {
	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	var options ListEndpointsOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}
	id := options.Id

	endpoints := make([]*dsypb.Endpoint, 0)
	if len(options.Service) != 0 {
		svcs, _ := e.GetService(ctx, options.Service, GetNamespace(namespace))
		for _, svc := range svcs {
			endpoints = append(endpoints, svc.Endpoints...)
		}
	}

	prefix := e.options.Prefix
	key := path.Join(prefix, namespace, "_ep_", id) + "/"
	listOpts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := e.client.Get(ctx, key, listOpts...)
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return endpoints, nil
	}

	for _, n := range rsp.Kvs {
		ep := decodeEp(n.Value)
		if ep == nil {
			continue
		}
		endpoints = append(endpoints, ep)
	}

	// sort the services
	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].Name < endpoints[j].Name })

	return endpoints, nil
}

func (e *etcdRegistrar) GetService(ctx context.Context, name string, opts ...GetOption) ([]*dsypb.Service, error) {
	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	var options GetOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	prefix := e.options.Prefix
	getOpts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := e.client.Get(ctx, servicePath(prefix, namespace, name), getOpts...)
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, ErrNotFound
	}

	serviceMap := map[string]*dsypb.Service{}

	for _, n := range rsp.Kvs {
		if sn := decode(n.Value); sn != nil {
			s, ok := serviceMap[sn.Version]
			if !ok {
				s = &dsypb.Service{
					Name:      sn.Name,
					Version:   sn.Version,
					Namespace: sn.Namespace,
					Metadata:  sn.Metadata,
					Endpoints: sn.Endpoints,
				}
				serviceMap[s.Version] = s
			}

			s.Nodes = append(s.Nodes, sn.Nodes...)
		}
	}

	services := make([]*dsypb.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (e *etcdRegistrar) ListServices(ctx context.Context, opts ...ListOption) ([]*dsypb.Service, error) {
	versions := make(map[string]*dsypb.Service)

	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	var options ListOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	prefix := e.options.Prefix
	key := path.Join(prefix, namespace, "_svc_") + "/"
	listOpts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}
	rsp, err := e.client.Get(ctx, key, listOpts...)
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return []*dsypb.Service{}, nil
	}

	for _, n := range rsp.Kvs {
		sn := decode(n.Value)
		if sn == nil {
			continue
		}
		v, ok := versions[sn.Name+sn.Version]
		if !ok {
			versions[sn.Name+sn.Version] = sn
			continue
		}
		// append to service:version nodes
		v.Nodes = append(v.Nodes, sn.Nodes...)
	}

	services := make([]*dsypb.Service, 0, len(versions))
	for _, service := range versions {
		services = append(services, &dsypb.Service{
			Name:      service.Name,
			Version:   service.Version,
			Namespace: service.Namespace,
			Metadata:  service.Metadata,
			Nodes:     service.Nodes,
			Endpoints: service.Endpoints,
		})
	}

	// sort the services
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })

	return services, nil
}

func (e *etcdRegistrar) Watch(ctx context.Context, opts ...WatchOption) (Watcher, error) {
	return newEtcdWatcher(ctx, e, opts...)
}

func encode(s *dsypb.Service) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds []byte) *dsypb.Service {
	s := &dsypb.Service{}
	_ = json.Unmarshal(ds, s)
	return s
}

func nodePath(prefix, ns, s, id string) string {
	service := strings.ReplaceAll(s, "/", "-")
	node := strings.ReplaceAll(id, "/", "-")
	return path.Join(prefix, ns, "_svc_", service, node)
}

func servicePath(prefix, ns, s string) string {
	return path.Join(prefix, ns, "_svc_", strings.ReplaceAll(s, "/", "-"))
}

func encodeEp(ep *dsypb.Endpoint) string {
	b, _ := json.Marshal(ep)
	return string(b)
}

func decodeEp(data []byte) *dsypb.Endpoint {
	ep := &dsypb.Endpoint{}
	_ = json.Unmarshal(data, ep)
	return ep
}

func endpointPath(prefix, ns, id, ep string) string {
	return path.Join(prefix, ns, "_ep_", id, strings.ReplaceAll(ep, "/", "-"))
}
