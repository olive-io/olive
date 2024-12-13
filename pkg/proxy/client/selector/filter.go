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

package selector

import (
	dsypb "github.com/olive-io/olive/api/discoverypb"
)

// FilterEndpoint is an endpoint based Select Filter which will
// only return services with the endpoint specified.
func FilterEndpoint(name string) Filter {
	return func(old []*dsypb.Service) []*dsypb.Service {
		var services []*dsypb.Service

		for _, service := range old {
			for _, ep := range service.Endpoints {
				if ep.Name == name {
					services = append(services, service)
					break
				}
			}
		}

		return services
	}
}

// FilterLabel is a label based Select Filter which will
// only return services with the label specified.
func FilterLabel(key, val string) Filter {
	return func(old []*dsypb.Service) []*dsypb.Service {
		var services []*dsypb.Service

		for _, service := range old {
			serv := new(dsypb.Service)
			var nodes []*dsypb.Node

			for _, node := range service.Nodes {
				if node.Metadata == nil {
					continue
				}

				if node.Metadata[key] == val {
					nodes = append(nodes, node)
				}
			}

			// only add service if there's some nodes
			if len(nodes) > 0 {
				// copy
				*serv = *service
				serv.Nodes = nodes
				services = append(services, serv)
			}
		}

		return services
	}
}

// FilterVersion is a version based Select Filter which will
// only return services with the version specified.
func FilterVersion(version string) Filter {
	return func(old []*dsypb.Service) []*dsypb.Service {
		var services []*dsypb.Service

		for _, service := range old {
			if service.Version == version {
				services = append(services, service)
			}
		}

		return services
	}
}
