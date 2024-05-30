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
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"

	monv1 "github.com/olive-io/olive/apis/mon/v1"
)

// runnerTree is a tree-like data structure that holds runner names in each zone. Zone names are
// keys to "RunnerTree.tree" and values of "RunnerTree.tree" are arrays of runner names.
// RunnerTree is NOT thread-safe, any concurrent updates/reads from it must be synchronized by the caller.
// It is used only by schedulerCache, and should stay as such.
type runnerTree struct {
	tree       map[string][]string // a map from zone (region-zone) to an array of runners in the zone.
	zones      []string            // a list of all the zones in the tree (keys)
	numRunners int
}

// newRunnerTree creates a RunnerTree from runners.
func newRunnerTree(logger klog.Logger, runners []*monv1.Runner) *runnerTree {
	nt := &runnerTree{
		tree: make(map[string][]string, len(runners)),
	}
	for _, n := range runners {
		nt.addRunner(logger, n)
	}
	return nt
}

// addRunner adds a runner and its corresponding zone to the tree. If the zone already exists, the runner
// is added to the array of runners in that zone.
func (nt *runnerTree) addRunner(logger klog.Logger, n *monv1.Runner) {
	zone := GetZoneKey(n)
	if na, ok := nt.tree[zone]; ok {
		for _, runnerName := range na {
			if runnerName == n.Name {
				logger.Info("Did not add to the RunnerTree because it already exists", "runner", klog.KObj(n))
				return
			}
		}
		nt.tree[zone] = append(na, n.Name)
	} else {
		nt.zones = append(nt.zones, zone)
		nt.tree[zone] = []string{n.Name}
	}
	logger.V(2).Info("Added runner in listed group to RunnerTree", "runner", klog.KObj(n), "zone", zone)
	nt.numRunners++
}

// removeRunner removes a runner from the RunnerTree.
func (nt *runnerTree) removeRunner(logger klog.Logger, n *monv1.Runner) error {
	zone := GetZoneKey(n)
	if na, ok := nt.tree[zone]; ok {
		for i, runnerName := range na {
			if runnerName == n.Name {
				nt.tree[zone] = append(na[:i], na[i+1:]...)
				if len(nt.tree[zone]) == 0 {
					nt.removeZone(zone)
				}
				logger.V(2).Info("Removed runner in listed group from RunnerTree", "runner", klog.KObj(n), "zone", zone)
				nt.numRunners--
				return nil
			}
		}
	}
	logger.Error(nil, "Did not remove Runner in RunnerTree because it was not found", "runner", klog.KObj(n), "zone", zone)
	return fmt.Errorf("runner %q in group %q was not found", n.Name, zone)
}

// removeZone removes a zone from tree.
// This function must be called while writer locks are hold.
func (nt *runnerTree) removeZone(zone string) {
	delete(nt.tree, zone)
	for i, z := range nt.zones {
		if z == zone {
			nt.zones = append(nt.zones[:i], nt.zones[i+1:]...)
			return
		}
	}
}

// updateRunner updates a runner in the RunnerTree.
func (nt *runnerTree) updateRunner(logger klog.Logger, old, new *monv1.Runner) {
	var oldZone string
	if old != nil {
		oldZone = GetZoneKey(old)
	}
	newZone := GetZoneKey(new)
	// If the zone ID of the runner has not changed, we don't need to do anything. Name of the runner
	// cannot be changed in an update.
	if oldZone == newZone {
		return
	}
	nt.removeRunner(logger, old) // No error checking. We ignore whether the old runner exists or not.
	nt.addRunner(logger, new)
}

// list returns the list of names of the runner. RunnerTree iterates over zones and in each zone iterates
// over runners in a round robin fashion.
func (nt *runnerTree) list() ([]string, error) {
	if len(nt.zones) == 0 {
		return nil, nil
	}
	runnersList := make([]string, 0, nt.numRunners)
	numExhaustedZones := 0
	runnerIndex := 0
	for len(runnersList) < nt.numRunners {
		if numExhaustedZones >= len(nt.zones) { // all zones are exhausted.
			return runnersList, errors.New("all zones exhausted before reaching count of runners expected")
		}
		for zoneIndex := 0; zoneIndex < len(nt.zones); zoneIndex++ {
			na := nt.tree[nt.zones[zoneIndex]]
			if runnerIndex >= len(na) { // If the zone is exhausted, continue
				if runnerIndex == len(na) { // If it is the first time the zone is exhausted
					numExhaustedZones++
				}
				continue
			}
			runnersList = append(runnersList, na[runnerIndex])
		}
		runnerIndex++
	}
	return runnersList, nil
}

// GetZoneKey is a helper function that builds a string identifier that is unique per failure-zone;
// it returns empty-string for no zone.
// Since there are currently two separate zone keys:
//   - "failure-domain.beta.olive.io/zone"
//   - "topology.olive.io/zone"
//
// GetZoneKey will first check failure-domain.beta.olive.io/zone and if not exists, will then check
// topology.olive.io/zone
func GetZoneKey(runner *monv1.Runner) string {
	labels := runner.Labels
	if labels == nil {
		return ""
	}

	// TODO: "failure-domain.beta..." names are deprecated, but will
	// stick around a long time due to existing on old extant objects like PVs.
	// Maybe one day we can stop considering them (see #88493).
	zone, _ := labels[v1.LabelTopologyZone]
	region, _ := labels[v1.LabelTopologyRegion]

	if region == "" && zone == "" {
		return ""
	}

	// We include the null character just in case region or failureDomain has a colon
	// (We do assume there's no null characters in a region or failureDomain)
	// As a nice side-benefit, the null character is not printed by fmt.Print or glog
	return region + ":\x00:" + zone
}
