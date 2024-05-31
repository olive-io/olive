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

// RunnerInfoLister interface represents anything that can list/get RunnerInfo objects from node name.
type RunnerInfoLister interface {
	// List returns the list of RunnerInfos.
	List() ([]*RunnerInfo, error)
	// HaveRegionsWithAffinityList returns the list of RunnerInfos of nodes with regions with affinity terms.
	HaveRegionsWithAffinityList() ([]*RunnerInfo, error)
	// HaveRegionsWithRequiredAntiAffinityList returns the list of RunnerInfos of nodes with regions with required anti-affinity terms.
	HaveRegionsWithRequiredAntiAffinityList() ([]*RunnerInfo, error)
	// Get returns the RunnerInfo of the given node name.
	Get(nodeName string) (*RunnerInfo, error)
}

// SharedLister groups scheduler-specific listers.
type SharedLister interface {
	RunnerInfos() RunnerInfoLister
}
