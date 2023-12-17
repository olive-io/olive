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
	"path"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	pb "github.com/olive-io/olive/api/discoverypb"
	"github.com/olive-io/olive/pkg/runtime"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

var (
	ErrNotFoundNode = errors.New("node not found")
	ErrNoNode       = errors.New("no node")
)

type oliveNode struct {
	node *pb.Node

	endpoints map[pb.Activity][]*pb.Endpoint
}

func newOliveNode(node *pb.Node) *oliveNode {
	on := &oliveNode{
		endpoints: map[pb.Activity][]*pb.Endpoint{},
	}
	on.set(node)
	return on
}

func (on *oliveNode) Get() *pb.Node {
	return on.node
}

func (on *oliveNode) set(node *pb.Node) {
	on.node = node
	on.endpoints = map[pb.Activity][]*pb.Endpoint{}
	for i := range node.Endpoints {
		endpoint := node.Endpoints[i]
		ea := endpoint.Activity
		list, ok := on.endpoints[ea]
		if !ok {
			list = make([]*pb.Endpoint, 0)
			on.endpoints[ea] = list
		}
		list = append(list, endpoint)
	}
}

func (on *oliveNode) isReady() bool {
	if on.node.Online() {
		return true
	}
	return false
}

func (on *oliveNode) GetExecutor(ctx context.Context, options ...DiscoverOption) (IExecutor, error) {
	var option DiscoverOptions
	for _, opt := range options {
		opt(&option)
	}

	act := option.activity
	switch act {
	case pb.Activity_Task:
	case pb.Activity_Service:
	case pb.Activity_Script:
	case pb.Activity_User:
	case pb.Activity_Call:
	case pb.Activity_Send:
	case pb.Activity_Receive:
	}

	return nil, nil
}

type etcdDiscovery struct {
	ctx    context.Context
	cancel context.CancelFunc

	lg     *zap.Logger
	client *clientv3.Client
	rev    int64

	nmu   sync.RWMutex
	nodes map[string]*oliveNode

	stopc <-chan struct{}
}

func NewDiscovery(lg *zap.Logger, client *clientv3.Client, stopc <-chan struct{}) (IDiscovery, error) {
	if lg == nil {
		lg = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(context.Background())

	dsy := &etcdDiscovery{
		ctx:    ctx,
		cancel: cancel,
		lg:     lg,
		client: client,
		nodes:  map[string]*oliveNode{},
		stopc:  stopc,
	}

	if err := dsy.syncNodes(ctx); err != nil {
		return nil, err
	}

	go dsy.run()
	return dsy, nil
}

func (dsy *etcdDiscovery) ListNode(ctx context.Context) ([]INode, error) {
	nodes := make([]INode, 0)
	dsy.nmu.RLock()
	defer dsy.nmu.Unlock()
	for id := range dsy.nodes {
		nodes = append(nodes, dsy.nodes[id])
	}
	return nodes, nil
}

func (dsy *etcdDiscovery) Discover(ctx context.Context, options ...DiscoverOption) (INode, error) {
	var option DiscoverOptions
	for _, opt := range options {
		opt(&option)
	}

	if id := option.nodeId; id != "" {
		node, ok := dsy.getNode(id)
		if !ok {
			return nil, errors.Wrapf(ErrNoNode, "id=%s", id)
		}
		return node, nil
	}

	dsy.nmu.RLock()
	defer dsy.nmu.RUnlock()
	for id := range dsy.nodes {
		node := dsy.nodes[id]
		if node.isReady() {
			return node, nil
		}
	}

	return nil, ErrNotFoundNode
}

func (dsy *etcdDiscovery) run() {
	defer dsy.cancel()

	rev := dsy.rev
	prefix := runtime.DefaultRunnerDiscoveryPrefix

	for {
		ctx, cancel := context.WithCancel(dsy.ctx)

		options := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithRev(rev + 1),
		}
		wch := dsy.client.Watch(ctx, prefix, options...)

	LOOP:
		for {
			select {
			case <-dsy.stopc:
				cancel()
				return
			case result := <-wch:
				if err := result.Err(); err != nil {
					break LOOP
				}

				for _, event := range result.Events {
					kv := event.Kv
					rev = kv.ModRevision

					key := string(kv.Key)
					// handler olive node event
					if strings.HasPrefix(key, runtime.DefaultRunnerDiscoveryNode) {

						if event.Type == clientv3.EventTypeDelete {
							id := path.Base(key)
							dsy.delNode(id)
							continue
						}

						node := new(pb.Node)
						if err := node.Unmarshal(kv.Value); err != nil {
							continue
						}

						if event.IsCreate() {
							dsy.pushNode(newOliveNode(node))
						} else if event.IsModify() {
							if v, ok := dsy.getNode(node.Id); ok {
								v.set(node)
							} else {
								dsy.pushNode(newOliveNode(node))
							}
						}
					}
				}
			}
		}

		cancel()
	}
}

func (dsy *etcdDiscovery) syncNodes(ctx context.Context) error {
	key := runtime.DefaultRunnerDiscoveryNode
	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSerializable(),
	}

	rsp, err := dsy.client.Get(ctx, key, options...)
	if err != nil {
		return err
	}
	dsy.rev = rsp.Header.Revision

	for _, kv := range rsp.Kvs {
		node := new(pb.Node)
		err = node.Unmarshal(kv.Value)
		if err != nil {
			continue
		}
		on := newOliveNode(node)
		dsy.pushNode(on)
	}

	return nil
}

func (dsy *etcdDiscovery) getNode(id string) (*oliveNode, bool) {
	dsy.nmu.RLock()
	node, ok := dsy.nodes[id]
	dsy.nmu.RUnlock()
	return node, ok
}

func (dsy *etcdDiscovery) pushNode(node *oliveNode) {
	dsy.nmu.Lock()
	dsy.nodes[node.Get().Id] = node
	dsy.nmu.Unlock()
}

func (dsy *etcdDiscovery) delNode(id string) {
	dsy.nmu.Lock()
	_, ok := dsy.nodes[id]
	if ok {
		dsy.nodes[id] = nil
		delete(dsy.nodes, id)
	}
	dsy.nmu.Unlock()
}
