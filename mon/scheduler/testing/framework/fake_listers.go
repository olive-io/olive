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

package framework

import (
	"fmt"

	"github.com/olive-io/olive/mon/scheduler/framework"
)

// RunnerInfoLister declares a framework.RunnerInfo type for testing.
type RunnerInfoLister []*framework.RunnerInfo

// Get returns a fake node object in the fake nodes.
func (nodes RunnerInfoLister) Get(nodeName string) (*framework.RunnerInfo, error) {
	for _, node := range nodes {
		if node != nil && node.Runner().Name == nodeName {
			return node, nil
		}
	}
	return nil, fmt.Errorf("unable to find node: %s", nodeName)
}

// List lists all nodes.
func (nodes RunnerInfoLister) List() ([]*framework.RunnerInfo, error) {
	return nodes, nil
}

// HavePodsWithAffinityList is supposed to list nodes with at least one pod with affinity. For the fake lister
// we just return everything.
func (nodes RunnerInfoLister) HavePodsWithAffinityList() ([]*framework.RunnerInfo, error) {
	return nodes, nil
}

// HavePodsWithRequiredAntiAffinityList is supposed to list nodes with at least one pod with
// required anti-affinity. For the fake lister we just return everything.
func (nodes RunnerInfoLister) HavePodsWithRequiredAntiAffinityList() ([]*framework.RunnerInfo, error) {
	return nodes, nil
}
