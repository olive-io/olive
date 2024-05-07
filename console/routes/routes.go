/*
   Copyright 2024 The olive Authors

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

package routes

import (
	"context"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/console/config"
	"github.com/olive-io/olive/pkg/tonic/fizz"
)

var (
	ErrAlreadyExists = errors.New("RouteGroup already exists")
)

type RouteGroupSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type IRouteGroup interface {
	Summary() RouteGroupSummary
	HandlerChains() []gin.HandlerFunc
}

type RouteTree struct {
	ctx context.Context
	lg  *zap.Logger
	cfg *config.Config

	root *fizz.RouterGroup
	oct  *client.Client

	mu     sync.RWMutex
	groups map[string]IRouteGroup
}

func RegisterRoutes(ctx context.Context, cfg *config.Config, root *fizz.RouterGroup, oct *client.Client) (*RouteTree, error) {
	routes := &RouteTree{
		ctx:    ctx,
		lg:     cfg.GetLogger(),
		root:   root,
		oct:    oct,
		groups: map[string]IRouteGroup{},
	}

	var err error
	if err = routes.registerCluster(); err != nil {
		return nil, err
	}
	if err = routes.registerRunnerGroup(); err != nil {
		return nil, err
	}
	if err = routes.registerRegionGroup(); err != nil {
		return nil, err
	}
	if err = routes.registerDefinitionsGroup(); err != nil {
		return nil, err
	}
	if err = routes.registerServiceGroup(); err != nil {
		return nil, err
	}

	return routes, nil
}

func (tree *RouteTree) Group(group IRouteGroup) error {
	tree.mu.Lock()
	defer tree.mu.Unlock()

	summary := group.Summary()
	name := summary.Name
	if _, ok := tree.groups[name]; ok {
		return errors.Wrap(ErrAlreadyExists, name)
	}
	tree.groups[name] = group
	return nil
}

func (tree *RouteTree) Destroy() {}
