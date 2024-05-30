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
	// HaveDefinitionsWithAffinityList returns the list of RunnerInfos of nodes with definitions with affinity terms.
	HaveDefinitionsWithAffinityList() ([]*RunnerInfo, error)
	// HaveDefinitionsWithRequiredAntiAffinityList returns the list of RunnerInfos of nodes with definitions with required anti-affinity terms.
	HaveDefinitionsWithRequiredAntiAffinityList() ([]*RunnerInfo, error)
	// Get returns the RunnerInfo of the given node name.
	Get(nodeName string) (*RunnerInfo, error)
}

// StorageInfoLister interface represents anything that handles storage-related operations and resources.
type StorageInfoLister interface {
	// IsPVCUsedByDefinitions returns true/false on whether the PVC is used by one or more scheduled definitions,
	// keyed in the format "namespace/name".
	IsPVCUsedByDefinitions(key string) bool
}

// SharedLister groups scheduler-specific listers.
type SharedLister interface {
	RunnerInfos() RunnerInfoLister
	StorageInfos() StorageInfoLister
}
