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

package cache

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	monv1 "github.com/olive-io/olive/apis/mon/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// Snapshot is a snapshot of cache RunnerInfo and RunnerTree order. The scheduler takes a
// snapshot at the beginning of each scheduling cycle and uses it for its operations in that cycle.
type Snapshot struct {
	// runnerInfoMap a map of runner name to a snapshot of its RunnerInfo.
	runnerInfoMap map[string]*framework.RunnerInfo
	// runnerInfoList is the list of runners as ordered in the cache's runnerTree.
	runnerInfoList []*framework.RunnerInfo
	// haveDefinitionsWithAffinityRunnerInfoList is the list of runners with at least one definition declaring affinity terms.
	haveDefinitionsWithAffinityRunnerInfoList []*framework.RunnerInfo
	// haveDefinitionsWithRequiredAntiAffinityRunnerInfoList is the list of runners with at least one definition declaring
	// required anti-affinity terms.
	haveDefinitionsWithRequiredAntiAffinityRunnerInfoList []*framework.RunnerInfo
	// usedPVCSet contains a set of PVC names that have one or more scheduled definitions using them,
	// keyed in the format "namespace/name".
	usedPVCSet sets.Set[string]
	generation int64
}

var _ framework.SharedLister = &Snapshot{}

// NewEmptySnapshot initializes a Snapshot struct and returns it.
func NewEmptySnapshot() *Snapshot {
	return &Snapshot{
		runnerInfoMap: make(map[string]*framework.RunnerInfo),
		usedPVCSet:    sets.New[string](),
	}
}

// NewSnapshot initializes a Snapshot struct and returns it.
func NewSnapshot(definitions []*corev1.Definition, runners []*monv1.Runner) *Snapshot {
	runnerInfoMap := createRunnerInfoMap(definitions, runners)
	runnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	haveDefinitionsWithAffinityRunnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	haveDefinitionsWithRequiredAntiAffinityRunnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	for _, v := range runnerInfoMap {
		runnerInfoList = append(runnerInfoList, v)
		if len(v.DefinitionsWithAffinity) > 0 {
			haveDefinitionsWithAffinityRunnerInfoList = append(haveDefinitionsWithAffinityRunnerInfoList, v)
		}
		if len(v.DefinitionsWithRequiredAntiAffinity) > 0 {
			haveDefinitionsWithRequiredAntiAffinityRunnerInfoList = append(haveDefinitionsWithRequiredAntiAffinityRunnerInfoList, v)
		}
	}

	s := NewEmptySnapshot()
	s.runnerInfoMap = runnerInfoMap
	s.runnerInfoList = runnerInfoList
	s.haveDefinitionsWithAffinityRunnerInfoList = haveDefinitionsWithAffinityRunnerInfoList
	s.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList = haveDefinitionsWithRequiredAntiAffinityRunnerInfoList
	s.usedPVCSet = createUsedPVCSet(definitions)

	return s
}

// createRunnerInfoMap obtains a list of definitions and pivots that list into a map
// where the keys are runner names and the values are the aggregated information
// for that runner.
func createRunnerInfoMap(definitions []*corev1.Definition, runners []*monv1.Runner) map[string]*framework.RunnerInfo {
	runnerNameToInfo := make(map[string]*framework.RunnerInfo)
	//for _, definition := range definitions {
	//	runnerName := definition.Spec.RunnerName
	//	if _, ok := runnerNameToInfo[runnerName]; !ok {
	//		runnerNameToInfo[runnerName] = framework.NewRunnerInfo()
	//	}
	//	runnerNameToInfo[runnerName].AddDefinition(definition)
	//}
	////imageExistenceMap := createImageExistenceMap(runners)
	//
	//for _, runner := range runners {
	//	if _, ok := runnerNameToInfo[runner.Name]; !ok {
	//		runnerNameToInfo[runner.Name] = framework.NewRunnerInfo()
	//	}
	//	runnerInfo := runnerNameToInfo[runner.Name]
	//	runnerInfo.SetRunner(runner)
	//	//runnerInfo.ImageStates = getRunnerImageStates(runner, imageExistenceMap)
	//}
	return runnerNameToInfo
}

func createUsedPVCSet(definitions []*corev1.Definition) sets.Set[string] {
	usedPVCSet := sets.New[string]()
	for _, definition := range definitions {
		if definition.Spec.RegionName == "" {
			continue
		}

	}
	return usedPVCSet
}

// RunnerInfos returns a RunnerInfoLister.
func (s *Snapshot) RunnerInfos() framework.RunnerInfoLister {
	return s
}

// StorageInfos returns a StorageInfoLister.
func (s *Snapshot) StorageInfos() framework.StorageInfoLister {
	return s
}

// NumRunners returns the number of runners in the snapshot.
func (s *Snapshot) NumRunners() int {
	return len(s.runnerInfoList)
}

// List returns the list of runners in the snapshot.
func (s *Snapshot) List() ([]*framework.RunnerInfo, error) {
	return s.runnerInfoList, nil
}

// HaveDefinitionsWithAffinityList returns the list of runners with at least one definition with inter-definition affinity
func (s *Snapshot) HaveDefinitionsWithAffinityList() ([]*framework.RunnerInfo, error) {
	return s.haveDefinitionsWithAffinityRunnerInfoList, nil
}

// HaveDefinitionsWithRequiredAntiAffinityList returns the list of runners with at least one definition with
// required inter-definition anti-affinity
func (s *Snapshot) HaveDefinitionsWithRequiredAntiAffinityList() ([]*framework.RunnerInfo, error) {
	return s.haveDefinitionsWithRequiredAntiAffinityRunnerInfoList, nil
}

// Get returns the RunnerInfo of the given runner name.
func (s *Snapshot) Get(runnerName string) (*framework.RunnerInfo, error) {
	if v, ok := s.runnerInfoMap[runnerName]; ok && v.Runner() != nil {
		return v, nil
	}
	return nil, fmt.Errorf("runnerinfo not found for runner name %q", runnerName)
}

func (s *Snapshot) IsPVCUsedByDefinitions(key string) bool {
	return s.usedPVCSet.Has(key)
}
