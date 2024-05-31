/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testing

import (
	"fmt"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type keyVal struct {
	k string
	v string
}

// MakeRunnersAndRegionsForEvenRegionsSpread serves as a testing helper for EvenRegionsSpread feature.
// It builds a fake cluster containing running Regions and Runners.
// The size of Regions and Runners are determined by input arguments.
// The specs of Regions and Runners are generated with the following rules:
//   - Each generated node is applied with a unique label: "node: node<i>".
//   - Each generated node is applied with a rotating label: "zone: zone[0-9]".
//   - Depending on the input labels, each generated pod will be applied with
//     label "key1", "key1,key2", ..., "key1,key2,...,keyN" in a rotating manner.
func MakeRunnersAndRegionsForEvenRegionsSpread(labels map[string]string, existingRegionsNum, allRunnersNum, filteredRunnersNum int) (existingRegions []*corev1.Region, allRunners []*corev1.Runner, filteredRunners []*corev1.Runner) {
	var labelPairs []keyVal
	for k, v := range labels {
		labelPairs = append(labelPairs, keyVal{k: k, v: v})
	}
	zones := 10
	// build nodes
	for i := 0; i < allRunnersNum; i++ {
		node := MakeRunner().Name(fmt.Sprintf("node%d", i)).
			Label(corev1.LabelTopologyZone, fmt.Sprintf("zone%d", i%zones)).
			Label(corev1.LabelHostname, fmt.Sprintf("node%d", i)).Obj()
		allRunners = append(allRunners, node)
	}
	filteredRunners = allRunners[:filteredRunnersNum]
	// build pods
	for i := 0; i < existingRegionsNum; i++ {
		podWrapper := MakeRegion().Name(fmt.Sprintf("pod%d", i)).Runners(fmt.Sprintf("node%d", i%allRunnersNum))
		// apply labels[0], labels[0,1], ..., labels[all] to each pod in turn
		for _, p := range labelPairs[:i%len(labelPairs)+1] {
			podWrapper = podWrapper.Label(p.k, p.v)
		}
		existingRegions = append(existingRegions, podWrapper.Obj())
	}
	return
}
