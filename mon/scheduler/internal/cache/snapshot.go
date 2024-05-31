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

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
)

// Snapshot is a snapshot of cache RunnerInfo and RunnerTree order. The scheduler takes a
// snapshot at the beginning of each scheduling cycle and uses it for its operations in that cycle.
type Snapshot struct {
	// runnerInfoMap a map of runner name to a snapshot of its RunnerInfo.
	runnerInfoMap map[string]*framework.RunnerInfo
	// runnerInfoList is the list of runners as ordered in the cache's runnerTree.
	runnerInfoList []*framework.RunnerInfo
	// haveRegionsWithAffinityRunnerInfoList is the list of runners with at least one region declaring affinity terms.
	haveRegionsWithAffinityRunnerInfoList []*framework.RunnerInfo
	// haveRegionsWithRequiredAntiAffinityRunnerInfoList is the list of runners with at least one region declaring
	// required anti-affinity terms.
	haveRegionsWithRequiredAntiAffinityRunnerInfoList []*framework.RunnerInfo
	// usedPVCSet contains a set of PVC names that have one or more scheduled regions using them,
	// keyed in the format "namespace/name".
	generation int64
}

var _ framework.SharedLister = &Snapshot{}

// NewEmptySnapshot initializes a Snapshot struct and returns it.
func NewEmptySnapshot() *Snapshot {
	return &Snapshot{
		runnerInfoMap: make(map[string]*framework.RunnerInfo),
	}
}

// NewSnapshot initializes a Snapshot struct and returns it.
func NewSnapshot(regions []*corev1.Region, runners []*corev1.Runner) *Snapshot {
	runnerInfoMap := createRunnerInfoMap(regions, runners)
	runnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	haveRegionsWithAffinityRunnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	haveRegionsWithRequiredAntiAffinityRunnerInfoList := make([]*framework.RunnerInfo, 0, len(runnerInfoMap))
	for _, v := range runnerInfoMap {
		runnerInfoList = append(runnerInfoList, v)
		if len(v.RegionsWithAffinity) > 0 {
			haveRegionsWithAffinityRunnerInfoList = append(haveRegionsWithAffinityRunnerInfoList, v)
		}
		if len(v.RegionsWithRequiredAntiAffinity) > 0 {
			haveRegionsWithRequiredAntiAffinityRunnerInfoList = append(haveRegionsWithRequiredAntiAffinityRunnerInfoList, v)
		}
	}

	s := NewEmptySnapshot()
	s.runnerInfoMap = runnerInfoMap
	s.runnerInfoList = runnerInfoList
	s.haveRegionsWithAffinityRunnerInfoList = haveRegionsWithAffinityRunnerInfoList
	s.haveRegionsWithRequiredAntiAffinityRunnerInfoList = haveRegionsWithRequiredAntiAffinityRunnerInfoList

	return s
}

// createRunnerInfoMap obtains a list of regions and pivots that list into a map
// where the keys are runner names and the values are the aggregated information
// for that runner.
func createRunnerInfoMap(regions []*corev1.Region, runners []*corev1.Runner) map[string]*framework.RunnerInfo {
	runnerNameToInfo := make(map[string]*framework.RunnerInfo)
	//for _, region := range regions {
	//	runnerName := region.Spec.RunnerName
	//	if _, ok := runnerNameToInfo[runnerName]; !ok {
	//		runnerNameToInfo[runnerName] = framework.NewRunnerInfo()
	//	}
	//	runnerNameToInfo[runnerName].AddRegion(region)
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

// RunnerInfos returns a RunnerInfoLister.
func (s *Snapshot) RunnerInfos() framework.RunnerInfoLister {
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

// HaveRegionsWithAffinityList returns the list of runners with at least one region with inter-region affinity
func (s *Snapshot) HaveRegionsWithAffinityList() ([]*framework.RunnerInfo, error) {
	return s.haveRegionsWithAffinityRunnerInfoList, nil
}

// HaveRegionsWithRequiredAntiAffinityList returns the list of runners with at least one region with
// required inter-region anti-affinity
func (s *Snapshot) HaveRegionsWithRequiredAntiAffinityList() ([]*framework.RunnerInfo, error) {
	return s.haveRegionsWithRequiredAntiAffinityRunnerInfoList, nil
}

// Get returns the RunnerInfo of the given runner name.
func (s *Snapshot) Get(runnerName string) (*framework.RunnerInfo, error) {
	if v, ok := s.runnerInfoMap[runnerName]; ok && v.Runner() != nil {
		return v, nil
	}
	return nil, fmt.Errorf("runnerinfo not found for runner name %q", runnerName)
}
