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

package discovery

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	hash "github.com/mitchellh/hashstructure"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

type etcdRegistry struct {
	client  *clientv3.Client
	options Options

	sync.RWMutex
	register map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func NewDiscovery(client *clientv3.Client, opts ...Option) (IDiscovery, error) {
	options := NewOptions(opts...)
	e := &etcdRegistry{
		client:   client,
		options:  options,
		register: make(map[string]uint64),
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

func (e *etcdRegistry) Options() Options {
	return e.options
}

func (e *etcdRegistry) registerNode(ctx context.Context, s *dsypb.Service, node *dsypb.Node, opts ...RegisterOption) error {
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

	lg.Sugar().Infof("Registering %s namespace %s id %s with lease %v and ttl %v",
		service.Name, service.Namespace, node.Id, lgr.ID, options.TTL)
	// create an entry for the node
	if lgr != nil {
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
	if lgr != nil {
		e.leases[s.Name+node.Id] = lgr.ID
	}
	e.Unlock()

	return nil
}

func (e *etcdRegistry) Register(ctx context.Context, s *dsypb.Service, opts ...RegisterOption) error {
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

func (e *etcdRegistry) Deregister(ctx context.Context, s *dsypb.Service, opts ...DeregisterOption) error {
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

func (e *etcdRegistry) GetService(ctx context.Context, name string, opts ...GetOption) ([]*dsypb.Service, error) {
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

func (e *etcdRegistry) ListServices(ctx context.Context, opts ...ListOption) ([]*dsypb.Service, error) {
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
	key := path.Join(prefix, namespace) + "/"
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

func (e *etcdRegistry) Watch(ctx context.Context, opts ...WatchOption) (Watcher, error) {
	return newEtcdWatcher(ctx, e, opts...)
}

func encode(s *dsypb.Service) string {
	b, _ := proto.Marshal(s)
	return string(b)
}

func decode(ds []byte) *dsypb.Service {
	s := &dsypb.Service{}
	_ = proto.Unmarshal(ds, s)
	return s
}

func nodePath(prefix, ns, s, id string) string {
	service := strings.ReplaceAll(s, "/", "-")
	node := strings.ReplaceAll(id, "/", "-")
	return path.Join(prefix, ns, service, node)
}

func servicePath(prefix, ns, s string) string {
	return path.Join(prefix, ns, strings.Replace(s, "/", "-", -1))
}
