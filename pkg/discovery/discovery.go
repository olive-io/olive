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
	"time"

	"github.com/cockroachdb/errors"

	dsypb "github.com/olive-io/olive/api/discoverypb"
)

var (
	DefaultRegistrarTimeout = time.Second * 10
	// DefaultNamespace the default value of namespace
	DefaultNamespace = "default"

	ErrNoDiscovery = errors.New("no discovery")
	// ErrNotFound not found error when GetService is called
	ErrNotFound = errors.New("service not found")
	// ErrWatcherStopped watcher stopped error when watcher is stopped
	ErrWatcherStopped = errors.New("watcher stopped")
)

// IRegistrar registry service
type IRegistrar interface {
	Register(context.Context, *dsypb.Service, ...RegisterOption) error
	Deregister(context.Context, *dsypb.Service, ...DeregisterOption) error
}

// IRouter registry endpoint
type IRouter interface {
	Inject(context.Context, *dsypb.Endpoint, ...InjectOption) error
	ListEndpoints(context.Context, ...ListEndpointsOption) ([]*dsypb.Endpoint, error)
}

// IDiscovery the registry provides an interface for service discovery
// and an abstraction over varying implementations
type IDiscovery interface {
	IRegistrar
	IRouter

	Options() Options
	GetService(context.Context, string, ...GetOption) ([]*dsypb.Service, error)
	ListServices(context.Context, ...ListOption) ([]*dsypb.Service, error)
	Watch(context.Context, ...WatchOption) (Watcher, error)
}
