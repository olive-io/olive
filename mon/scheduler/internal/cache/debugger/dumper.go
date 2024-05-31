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

package debugger

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
	"github.com/olive-io/olive/mon/scheduler/internal/queue"
)

// CacheDumper writes some information from the scheduler cache and the scheduling queue to the
// scheduler logs for debugging purposes.
type CacheDumper struct {
	cache       internalcache.Cache
	regionQueue queue.SchedulingQueue
}

// DumpAll writes cached nodes and scheduling queue information to the scheduler logs.
func (d *CacheDumper) DumpAll(logger klog.Logger) {
	d.dumpRunners(logger)
	d.dumpSchedulingQueue(logger)
}

// dumpRunners writes RunnerInfo to the scheduler logs.
func (d *CacheDumper) dumpRunners(logger klog.Logger) {
	dump := d.cache.Dump()
	nodeInfos := make([]string, 0, len(dump.Runners))
	for name, nodeInfo := range dump.Runners {
		nodeInfos = append(nodeInfos, d.printRunnerInfo(name, nodeInfo))
	}
	// Extra blank line added between node entries for readability.
	logger.Info("Dump of cached RunnerInfo", "nodes", strings.Join(nodeInfos, "\n\n"))
}

// dumpSchedulingQueue writes regions in the scheduling queue to the scheduler logs.
func (d *CacheDumper) dumpSchedulingQueue(logger klog.Logger) {
	pendingRegions, s := d.regionQueue.PendingRegions()
	var regionData strings.Builder
	for _, p := range pendingRegions {
		regionData.WriteString(printRegion(p))
	}
	logger.Info("Dump of scheduling queue", "summary", s, "regions", regionData.String())
}

// printRunnerInfo writes parts of RunnerInfo to a string.
func (d *CacheDumper) printRunnerInfo(name string, n *framework.RunnerInfo) string {
	var nodeData strings.Builder
	nodeData.WriteString(fmt.Sprintf("Runner name: %s\nDeleted: %t\nRequested Resources: %+v\nAllocatable Resources:%+v\nScheduled Regions(number: %v):\n",
		name, n.Runner() == nil, n.Requested, n.Allocatable, len(n.Regions)))
	// Dumping Region Info
	for _, p := range n.Regions {
		nodeData.WriteString(printRegion(p.Region))
	}
	// Dumping nominated regions info on the node
	nominatedRegionInfos := d.regionQueue.NominatedRegionsForRunner(name)
	if len(nominatedRegionInfos) != 0 {
		nodeData.WriteString(fmt.Sprintf("Nominated Regions(number: %v):\n", len(nominatedRegionInfos)))
		for _, pi := range nominatedRegionInfos {
			nodeData.WriteString(printRegion(pi.Region))
		}
	}
	return nodeData.String()
}

// printRegion writes parts of a Region object to a string.
func printRegion(p *corev1.Region) string {
	return fmt.Sprintf("name: %v, namespace: %v, uid: %v, phase: %v, nominated region: %v\n", p.Name, p.Namespace, p.UID, p.Status.Phase, "")
}
