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
	cache           internalcache.Cache
	definitionQueue queue.SchedulingQueue
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

// dumpSchedulingQueue writes definitions in the scheduling queue to the scheduler logs.
func (d *CacheDumper) dumpSchedulingQueue(logger klog.Logger) {
	pendingDefinitions, s := d.definitionQueue.PendingDefinitions()
	var definitionData strings.Builder
	for _, p := range pendingDefinitions {
		definitionData.WriteString(printDefinition(p))
	}
	logger.Info("Dump of scheduling queue", "summary", s, "definitions", definitionData.String())
}

// printRunnerInfo writes parts of RunnerInfo to a string.
func (d *CacheDumper) printRunnerInfo(name string, n *framework.RunnerInfo) string {
	var nodeData strings.Builder
	nodeData.WriteString(fmt.Sprintf("Runner name: %s\nDeleted: %t\nRequested Resources: %+v\nAllocatable Resources:%+v\nScheduled Definitions(number: %v):\n",
		name, n.Runner() == nil, n.Requested, n.Allocatable, len(n.Definitions)))
	// Dumping Definition Info
	for _, p := range n.Definitions {
		nodeData.WriteString(printDefinition(p.Definition))
	}
	// Dumping nominated definitions info on the node
	nominatedDefinitionInfos := d.definitionQueue.NominatedDefinitionsForRunner(name)
	if len(nominatedDefinitionInfos) != 0 {
		nodeData.WriteString(fmt.Sprintf("Nominated Definitions(number: %v):\n", len(nominatedDefinitionInfos)))
		for _, pi := range nominatedDefinitionInfos {
			nodeData.WriteString(printDefinition(pi.Definition))
		}
	}
	return nodeData.String()
}

// printDefinition writes parts of a Definition object to a string.
func printDefinition(p *corev1.Definition) string {
	return fmt.Sprintf("name: %v, namespace: %v, uid: %v, phase: %v, nominated region: %v\n", p.Name, p.Namespace, p.UID, p.Status.Phase, "")
}
