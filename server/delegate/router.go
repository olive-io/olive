/*
Copyright 2025 The olive Authors

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

package delegate

import (
	"errors"
	"sync"

	"github.com/olive-io/olive/api/types"
	"github.com/olive-io/olive/pkg/tree"
)

var (
	ErrNotFound = errors.New("endpoint not found")
)

type Runner struct {
	Id uint64
}

type Endpoint struct {
	types.Endpoint

	Runners []*Runner
}

// InsertRunner inserts or sets Runner
func (e *Endpoint) InsertRunner(runner *Runner) {
	for i, r := range e.Runners {
		if r.Id == runner.Id {
			e.Runners[i] = runner
			return
		}
	}
	e.Runners = append(e.Runners, runner)
}

// Router means Endpoint Radix Tree and Synchronous safe.
type Router struct {
	mu   sync.RWMutex
	root *tree.Tree[*Endpoint]
}

func NewRouter() (*Router, error) {
	routerTree := tree.New[*Endpoint]()
	controller := &Router{
		root: routerTree,
	}

	return controller, nil
}

// Insert inserts a endpoint into Router.
func (r *Router) Insert(url string, endpoint *Endpoint) error {
	exists, _ := r.Find(url)
	if exists != nil {
		for _, runner := range endpoint.Runners {
			exists.InsertRunner(runner)
		}
		endpoint = exists
	}

	r.mu.Lock()
	r.root.Insert(url, endpoint)
	r.mu.Unlock()
	return nil
}

// Find returns an endpoint by url.
func (r *Router) Find(url string) (*Endpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.root.Get(url)
	if !ok {
		return nil, ErrNotFound
	}
	return value, nil
}
