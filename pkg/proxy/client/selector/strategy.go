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
	"math/rand"
	"sync"
	"time"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Random is a random strategy algorithm for node selection
func Random(services []*dsypb.Service) Next {
	nodes := make([]*dsypb.Node, 0, len(services))

	for _, service := range services {
		nodes = append(nodes, service.Nodes...)
	}

	return func() (*dsypb.Node, error) {
		if len(nodes) == 0 {
			return nil, ErrNoneAvailable
		}

		i := rand.Int() % len(nodes)
		return nodes[i], nil
	}
}

// RoundRobin is a roundrobin strategy algorithm for node selection
func RoundRobin(services []*dsypb.Service) Next {
	nodes := make([]*dsypb.Node, 0, len(services))

	for _, service := range services {
		nodes = append(nodes, service.Nodes...)
	}

	var i = rand.Int()
	var mtx sync.Mutex

	return func() (*dsypb.Node, error) {
		if len(nodes) == 0 {
			return nil, ErrNoneAvailable
		}

		mtx.Lock()
		node := nodes[i%len(nodes)]
		i++
		mtx.Unlock()

		return node, nil
	}
}
