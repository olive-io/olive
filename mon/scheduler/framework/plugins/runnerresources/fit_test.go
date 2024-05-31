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


package runnerresources

//import (
//	"context"
//	"fmt"
//	"reflect"
//	"testing"
//
//	"github.com/google/go-cmp/cmp"
//	plugintesting "github.com/olive-io/olive/mon/scheduler/framework/plugins/testing"
//	"github.com/stretchr/testify/require"
//	v1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/api/resource"
//	"k8s.io/klog/v2/ktesting"
//	_ "k8s.io/klog/v2/ktesting/init"
//
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	"github.com/olive-io/olive/mon/scheduler/apis/config"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	plfeature "github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
//	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	"github.com/olive-io/olive/mon/scheduler/internal/cache"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//)
//
//var (
//	extendedResourceA     = corev1.ResourceName("example.com/aaa")
//	extendedResourceB     = corev1.ResourceName("example.com/bbb")
//	kubernetesIOResourceA = corev1.ResourceName("kubernetes.io/something")
//	kubernetesIOResourceB = corev1.ResourceName("subdomain.kubernetes.io/something")
//	hugePageResourceA     = corev1.ResourceName(corev1.ResourceHugePagesPrefix + "2Mi")
//)
//
//func makeResources(milliCPU, memory, regions, extendedA, storage, hugePageA int64) corev1.ResourceList {
//	return corev1.ResourceList{
//		corev1.ResourceCPU:              *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
//		corev1.ResourceMemory:           *resource.NewQuantity(memory, resource.BinarySI),
//		corev1.ResourceRegions:          *resource.NewQuantity(regions, resource.DecimalSI),
//		extendedResourceA:           *resource.NewQuantity(extendedA, resource.DecimalSI),
//		corev1.ResourceEphemeralStorage: *resource.NewQuantity(storage, resource.BinarySI),
//		hugePageResourceA:           *resource.NewQuantity(hugePageA, resource.BinarySI),
//	}
//}
//
//func makeAllocatableResources(milliCPU, memory, regions, extendedA, storage, hugePageA int64) corev1.ResourceList {
//	return corev1.ResourceList{
//		corev1.ResourceCPU:              *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
//		corev1.ResourceMemory:           *resource.NewQuantity(memory, resource.BinarySI),
//		corev1.ResourceRegions:          *resource.NewQuantity(regions, resource.DecimalSI),
//		extendedResourceA:           *resource.NewQuantity(extendedA, resource.DecimalSI),
//		corev1.ResourceEphemeralStorage: *resource.NewQuantity(storage, resource.BinarySI),
//		hugePageResourceA:           *resource.NewQuantity(hugePageA, resource.BinarySI),
//	}
//}
//
//func newResourceRegion(usage ...framework.Resource) *corev1.Region {
//	var containers []v1.Container
//	for _, req := range usage {
//		rl := corev1.ResourceList{
//			corev1.ResourceCPU:              *resource.NewMilliQuantity(req.MilliCPU, resource.DecimalSI),
//			corev1.ResourceMemory:           *resource.NewQuantity(req.Memory, resource.BinarySI),
//			corev1.ResourceRegions:          *resource.NewQuantity(int64(req.AllowedRegionNumber), resource.BinarySI),
//			corev1.ResourceEphemeralStorage: *resource.NewQuantity(req.EphemeralStorage, resource.BinarySI),
//		}
//		for rName, rQuant := range req.ScalarResources {
//			if rName == hugePageResourceA {
//				rl[rName] = *resource.NewQuantity(rQuant, resource.BinarySI)
//			} else {
//				rl[rName] = *resource.NewQuantity(rQuant, resource.DecimalSI)
//			}
//		}
//		containers = append(containers, v1.Container{
//			Resources: corev1.ResourceRequirements{Requests: rl},
//		})
//	}
//	return &corev1.Region{
//		Spec: corev1.RegionSpec{
//			Containers: containers,
//		},
//	}
//}
//
//func newResourceInitRegion(region *corev1.Region, usage ...framework.Resource) *corev1.Region {
//	region.Spec.InitContainers = newResourceRegion(usage...).Spec.Containers
//	return region
//}
//
//func newResourceOverheadRegion(region *corev1.Region, overhead corev1.ResourceList) *corev1.Region {
//	region.Spec.Overhead = overhead
//	return region
//}
//
//func getErrReason(rn corev1.ResourceName) string {
//	return fmt.Sprintf("Insufficient %v", rn)
//}
//
//var defaultScoringStrategy = &config.ScoringStrategy{
//	Type: config.LeastAllocated,
//	Resources: []config.ResourceSpec{
//		{Name: "cpu", Weight: 1},
//		{Name: "memory", Weight: 1},
//	},
//}
//
//func TestEnoughRequests(t *testing.T) {
//	enoughRegionsTests := []struct {
//		region                    *corev1.Region
//		runnerInfo                *framework.RunnerInfo
//		name                      string
//		args                      config.RunnerResourcesFitArgs
//		wantInsufficientResources []InsufficientResource
//		wantStatus                *framework.Status
//	}{
//		{
//			region: &corev1.Region{},
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 10, Memory: 20})),
//			name:                      "no resources requested always fits",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 10, Memory: 20})),
//			name:       "too many resources fails",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceCPU), getErrReason(corev1.ResourceMemory)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceCPU, Reason: getErrReason(corev1.ResourceCPU), Requested: 1, Used: 10, Capacity: 10},
//				{ResourceName: corev1.ResourceMemory, Reason: getErrReason(corev1.ResourceMemory), Requested: 1, Used: 20, Capacity: 20},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 3, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 8, Memory: 19})),
//			name:       "too many resources fails due to init container cpu",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceCPU)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceCPU, Reason: getErrReason(corev1.ResourceCPU), Requested: 3, Used: 8, Capacity: 10},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 3, Memory: 1}, framework.Resource{MilliCPU: 2, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 8, Memory: 19})),
//			name:       "too many resources fails due to highest init container cpu",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceCPU)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceCPU, Reason: getErrReason(corev1.ResourceCPU), Requested: 3, Used: 8, Capacity: 10},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 1, Memory: 3}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 9, Memory: 19})),
//			name:       "too many resources fails due to init container memory",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceMemory)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceMemory, Reason: getErrReason(corev1.ResourceMemory), Requested: 3, Used: 19, Capacity: 20},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 1, Memory: 3}, framework.Resource{MilliCPU: 1, Memory: 2}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 9, Memory: 19})),
//			name:       "too many resources fails due to highest init container memory",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceMemory)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceMemory, Reason: getErrReason(corev1.ResourceMemory), Requested: 3, Used: 19, Capacity: 20},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 9, Memory: 19})),
//			name:                      "init container fits because it's the max, not sum, of containers and init containers",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}), framework.Resource{MilliCPU: 1, Memory: 1}, framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 9, Memory: 19})),
//			name:                      "multiple init containers fit because it's the max, not sum, of containers and init containers",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 5})),
//			name:                      "both resources fit",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 2, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 9, Memory: 5})),
//			name:       "one resource memory fits",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceCPU)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceCPU, Reason: getErrReason(corev1.ResourceCPU), Requested: 2, Used: 9, Capacity: 10},
//			},
//		},
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 2}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 19})),
//			name:       "one resource cpu fits",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceMemory)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceMemory, Reason: getErrReason(corev1.ResourceMemory), Requested: 2, Used: 19, Capacity: 20},
//			},
//		},
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 19})),
//			name:                      "equal edge case",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 4, Memory: 1}), framework.Resource{MilliCPU: 5, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 19})),
//			name:                      "equal edge case for init container",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region:                    newResourceRegion(framework.Resource{ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}}),
//			runnerInfo:                framework.NewRunnerInfo(newResourceRegion(framework.Resource{})),
//			name:                      "extended resource fits",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region:                    newResourceInitRegion(newResourceRegion(framework.Resource{}), framework.Resource{ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}}),
//			runnerInfo:                framework.NewRunnerInfo(newResourceRegion(framework.Resource{})),
//			name:                      "extended resource fits for init container",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 0}})),
//			name:       "extended resource capacity enforced",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 10, Used: 0, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 0}})),
//			name:       "extended resource capacity enforced for init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 10, Used: 0, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 5}})),
//			name:       "extended resource allocatable enforced",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 1, Used: 5, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 5}})),
//			name:       "extended resource allocatable enforced for init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 1, Used: 5, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 3}},
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 3}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 2}})),
//			name:       "extended resource allocatable enforced for multiple containers",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 6, Used: 2, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 3}},
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 3}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 2}})),
//			name:                      "extended resource allocatable admits multiple init containers",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 6}},
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 3}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 2}})),
//			name:       "extended resource allocatable enforced for multiple init containers",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceA, Reason: getErrReason(extendedResourceA), Requested: 6, Used: 2, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceB: 1}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			name:       "extended resource allocatable enforced for unknown resource",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceB)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceB, Reason: getErrReason(extendedResourceB), Requested: 1, Used: 0, Capacity: 0},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceB: 1}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			name:       "extended resource allocatable enforced for unknown resource for init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(extendedResourceB)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: extendedResourceB, Reason: getErrReason(extendedResourceB), Requested: 1, Used: 0, Capacity: 0},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{kubernetesIOResourceA: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			name:       "kubernetes.io resource capacity enforced",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(kubernetesIOResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: kubernetesIOResourceA, Reason: getErrReason(kubernetesIOResourceA), Requested: 10, Used: 0, Capacity: 0},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{kubernetesIOResourceB: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			name:       "kubernetes.io resource capacity enforced for init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(kubernetesIOResourceB)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: kubernetesIOResourceB, Reason: getErrReason(kubernetesIOResourceB), Requested: 10, Used: 0, Capacity: 0},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 0}})),
//			name:       "hugepages resource capacity enforced",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(hugePageResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: hugePageResourceA, Reason: getErrReason(hugePageResourceA), Requested: 10, Used: 0, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{}),
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 10}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 0}})),
//			name:       "hugepages resource capacity enforced for init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(hugePageResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: hugePageResourceA, Reason: getErrReason(hugePageResourceA), Requested: 10, Used: 0, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 3}},
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 3}}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{hugePageResourceA: 2}})),
//			name:       "hugepages resource allocatable enforced for multiple containers",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(hugePageResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: hugePageResourceA, Reason: getErrReason(hugePageResourceA), Requested: 6, Used: 2, Capacity: 5},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{MilliCPU: 1, Memory: 1, ScalarResources: map[corev1.ResourceName]int64{extendedResourceB: 1}}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			args: config.RunnerResourcesFitArgs{
//				IgnoredResources: []string{"example.com/bbb"},
//			},
//			name:                      "skip checking ignored extended resource",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceOverheadRegion(
//				newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//				corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("3m"), corev1.ResourceMemory: resource.MustParse("13")},
//			),
//			runnerInfo:                framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 5})),
//			name:                      "resources + region overhead fits",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceOverheadRegion(
//				newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//				corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1m"), corev1.ResourceMemory: resource.MustParse("15")},
//			),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 5})),
//			name:       "requests + overhead does not fit for memory",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceMemory)),
//			wantInsufficientResources: []InsufficientResource{
//				{ResourceName: corev1.ResourceMemory, Reason: getErrReason(corev1.ResourceMemory), Requested: 16, Used: 5, Capacity: 20},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{
//					MilliCPU: 1,
//					Memory:   1,
//					ScalarResources: map[corev1.ResourceName]int64{
//						extendedResourceB:     1,
//						kubernetesIOResourceA: 1,
//					}}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 0, Memory: 0})),
//			args: config.RunnerResourcesFitArgs{
//				IgnoredResourceGroups: []string{"example.com"},
//			},
//			name:       "skip checking ignored extended resource via resource groups",
//			wantStatus: framework.NewStatus(framework.Unschedulable, fmt.Sprintf("Insufficient %v", kubernetesIOResourceA)),
//			wantInsufficientResources: []InsufficientResource{
//				{
//					ResourceName: kubernetesIOResourceA,
//					Reason:       fmt.Sprintf("Insufficient %v", kubernetesIOResourceA),
//					Requested:    1,
//					Used:         0,
//					Capacity:     0,
//				},
//			},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{
//					MilliCPU: 1,
//					Memory:   1,
//					ScalarResources: map[corev1.ResourceName]int64{
//						extendedResourceA: 0,
//					}}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{
//				MilliCPU: 0, Memory: 0, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 6}})),
//			name:                      "skip checking extended resource request with quantity zero via resource groups",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//		{
//			region: newResourceRegion(
//				framework.Resource{
//					ScalarResources: map[corev1.ResourceName]int64{
//						extendedResourceA: 1,
//					}}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{
//				MilliCPU: 20, Memory: 30, ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}})),
//			name:                      "skip checking resource request with quantity zero",
//			wantInsufficientResources: []InsufficientResource{},
//		},
//	}
//
//	for _, test := range enoughRegionsTests {
//		t.Run(test.name, func(t *testing.T) {
//			runner := corev1.Runner{Status: corev1.RunnerStatus{Capacity: makeResources(10, 20, 32, 5, 20, 5), Allocatable: makeAllocatableResources(10, 20, 32, 5, 20, 5)}}
//			test.runnerInfo.SetRunner(&runner)
//
//			if test.args.ScoringStrategy == nil {
//				test.args.ScoringStrategy = defaultScoringStrategy
//			}
//
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			p, err := NewFit(ctx, &test.args, nil, plfeature.Features{})
//			if err != nil {
//				t.Fatal(err)
//			}
//			cycleState := framework.NewCycleState()
//			_, preFilterStatus := p.(framework.PreFilterPlugin).PreFilter(ctx, cycleState, test.region)
//			if !preFilterStatus.IsSuccess() {
//				t.Errorf("prefilter failed with status: %v", preFilterStatus)
//			}
//
//			gotStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, test.region, test.runnerInfo)
//			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
//				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
//			}
//
//			gotInsufficientResources := fitsRequest(computeRegionResourceRequest(test.region), test.runnerInfo, p.(*Fit).ignoredResources, p.(*Fit).ignoredResourceGroups)
//			if !reflect.DeepEqual(gotInsufficientResources, test.wantInsufficientResources) {
//				t.Errorf("insufficient resources do not match: %+v, want: %v", gotInsufficientResources, test.wantInsufficientResources)
//			}
//		})
//	}
//}
//
//func TestPreFilterDisabled(t *testing.T) {
//	_, ctx := ktesting.NewTestContext(t)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	region := &corev1.Region{}
//	runnerInfo := framework.NewRunnerInfo()
//	runner := corev1.Runner{}
//	runnerInfo.SetRunner(&runner)
//	p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{})
//	if err != nil {
//		t.Fatal(err)
//	}
//	cycleState := framework.NewCycleState()
//	gotStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, region, runnerInfo)
//	wantStatus := framework.AsStatus(fmt.Errorf(`error reading "PreFilterRunnerResourcesFit" from cycleState: %w`, framework.ErrNotFound))
//	if !reflect.DeepEqual(gotStatus, wantStatus) {
//		t.Errorf("status does not match: %v, want: %v", gotStatus, wantStatus)
//	}
//}
//
//func TestNotEnoughRequests(t *testing.T) {
//	notEnoughRegionsTests := []struct {
//		region     *corev1.Region
//		runnerInfo *framework.RunnerInfo
//		fits       bool
//		name       string
//		wantStatus *framework.Status
//	}{
//		{
//			region:     &corev1.Region{},
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 10, Memory: 20})),
//			name:       "even without specified resources, predicate fails when there's no space for additional region",
//			wantStatus: framework.NewStatus(framework.Unschedulable, "Too many regions"),
//		},
//		{
//			region:     newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 5})),
//			name:       "even if both resources fit, predicate fails when there's no space for additional region",
//			wantStatus: framework.NewStatus(framework.Unschedulable, "Too many regions"),
//		},
//		{
//			region:     newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 19})),
//			name:       "even for equal edge case, predicate fails when there's no space for additional region",
//			wantStatus: framework.NewStatus(framework.Unschedulable, "Too many regions"),
//		},
//		{
//			region:     newResourceInitRegion(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 1}), framework.Resource{MilliCPU: 5, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(newResourceRegion(framework.Resource{MilliCPU: 5, Memory: 19})),
//			name:       "even for equal edge case, predicate fails when there's no space for additional region due to init container",
//			wantStatus: framework.NewStatus(framework.Unschedulable, "Too many regions"),
//		},
//	}
//	for _, test := range notEnoughRegionsTests {
//		t.Run(test.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			runner := corev1.Runner{Status: corev1.RunnerStatus{Capacity: corev1.ResourceList{}, Allocatable: makeAllocatableResources(10, 20, 1, 0, 0, 0)}}
//			test.runnerInfo.SetRunner(&runner)
//
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{})
//			if err != nil {
//				t.Fatal(err)
//			}
//			cycleState := framework.NewCycleState()
//			_, preFilterStatus := p.(framework.PreFilterPlugin).PreFilter(ctx, cycleState, test.region)
//			if !preFilterStatus.IsSuccess() {
//				t.Errorf("prefilter failed with status: %v", preFilterStatus)
//			}
//
//			gotStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, test.region, test.runnerInfo)
//			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
//				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
//			}
//		})
//	}
//
//}
//
//func TestStorageRequests(t *testing.T) {
//	storageRegionsTests := []struct {
//		region     *corev1.Region
//		runnerInfo *framework.RunnerInfo
//		name       string
//		wantStatus *framework.Status
//	}{
//		{
//			region: newResourceRegion(framework.Resource{MilliCPU: 1, Memory: 1}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 2, Memory: 10})),
//			name: "empty storage requested, and region fits",
//		},
//		{
//			region: newResourceRegion(framework.Resource{EphemeralStorage: 25}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 2, Memory: 2})),
//			name:       "storage ephemeral local storage request exceeds allocatable",
//			wantStatus: framework.NewStatus(framework.Unschedulable, getErrReason(corev1.ResourceEphemeralStorage)),
//		},
//		{
//			region: newResourceInitRegion(newResourceRegion(framework.Resource{EphemeralStorage: 5})),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 2, Memory: 2, EphemeralStorage: 10})),
//			name: "ephemeral local storage is sufficient",
//		},
//		{
//			region: newResourceRegion(framework.Resource{EphemeralStorage: 10}),
//			runnerInfo: framework.NewRunnerInfo(
//				newResourceRegion(framework.Resource{MilliCPU: 2, Memory: 2})),
//			name: "region fits",
//		},
//	}
//
//	for _, test := range storageRegionsTests {
//		t.Run(test.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			runner := corev1.Runner{Status: corev1.RunnerStatus{Capacity: makeResources(10, 20, 32, 5, 20, 5), Allocatable: makeAllocatableResources(10, 20, 32, 5, 20, 5)}}
//			test.runnerInfo.SetRunner(&runner)
//
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{})
//			if err != nil {
//				t.Fatal(err)
//			}
//			cycleState := framework.NewCycleState()
//			_, preFilterStatus := p.(framework.PreFilterPlugin).PreFilter(ctx, cycleState, test.region)
//			if !preFilterStatus.IsSuccess() {
//				t.Errorf("prefilter failed with status: %v", preFilterStatus)
//			}
//
//			gotStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, test.region, test.runnerInfo)
//			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
//				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
//			}
//		})
//	}
//
//}
//
//func TestRestartableInitContainers(t *testing.T) {
//	newRegion := func() *corev1.Region {
//		return &corev1.Region{
//			Spec: corev1.RegionSpec{
//				Containers: []v1.Container{
//					{Name: "regular"},
//				},
//			},
//		}
//	}
//	newRegionWithRestartableInitContainers := func() *corev1.Region {
//		restartPolicyAlways := v1.ContainerRestartPolicyAlways
//		return &corev1.Region{
//			Spec: corev1.RegionSpec{
//				Containers: []v1.Container{
//					{Name: "regular"},
//				},
//				InitContainers: []v1.Container{
//					{
//						Name:          "restartable-init",
//						RestartPolicy: &restartPolicyAlways,
//					},
//				},
//			},
//		}
//	}
//
//	testCases := []struct {
//		name                    string
//		region                  *corev1.Region
//		enableSidecarContainers bool
//		wantPreFilterStatus     *framework.Status
//	}{
//		{
//			name:   "allow region without restartable init containers if sidecar containers is disabled",
//			region: newRegion(),
//		},
//		{
//			name:                "not allow region with restartable init containers if sidecar containers is disabled",
//			region:              newRegionWithRestartableInitContainers(),
//			wantPreFilterStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable, "Region has a restartable init container and the SidecarContainers feature is disabled"),
//		},
//		{
//			name:                    "allow region without restartable init containers if sidecar containers is enabled",
//			enableSidecarContainers: true,
//			region:                  newRegion(),
//		},
//		{
//			name:                    "allow region with restartable init containers if sidecar containers is enabled",
//			enableSidecarContainers: true,
//			region:                  newRegionWithRestartableInitContainers(),
//		},
//	}
//
//	for _, test := range testCases {
//		t.Run(test.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			runner := corev1.Runner{Status: corev1.RunnerStatus{Capacity: corev1.ResourceList{}, Allocatable: makeAllocatableResources(0, 0, 1, 0, 0, 0)}}
//			runnerInfo := framework.NewRunnerInfo()
//			runnerInfo.SetRunner(&runner)
//
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{EnableSidecarContainers: test.enableSidecarContainers})
//			if err != nil {
//				t.Fatal(err)
//			}
//			cycleState := framework.NewCycleState()
//			_, preFilterStatus := p.(framework.PreFilterPlugin).PreFilter(context.Background(), cycleState, test.region)
//			if diff := cmp.Diff(test.wantPreFilterStatus, preFilterStatus); diff != "" {
//				t.Error("status does not match (-expected +actual):\n", diff)
//			}
//			if !preFilterStatus.IsSuccess() {
//				return
//			}
//
//			filterStatus := p.(framework.FilterPlugin).Filter(ctx, cycleState, test.region, runnerInfo)
//			if !filterStatus.IsSuccess() {
//				t.Error("status does not match (-expected +actual):\n- Success\n +\n", filterStatus.Code())
//			}
//		})
//	}
//
//}
//
//func TestFitScore(t *testing.T) {
//	tests := []struct {
//		name                   string
//		requestedRegion        *corev1.Region
//		runners                []*corev1.Runner
//		existingRegions        []*corev1.Region
//		expectedPriorities     framework.RunnerScoreList
//		runnerResourcesFitArgs config.RunnerResourcesFitArgs
//		runPreScore            bool
//	}{
//		{
//			name: "test case for ScoringStrategy RequestedToCapacityRatio case1",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "3000", "memory": "5000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 10}, {Name: "runner2", Score: 32}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: defaultResources,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 10},
//							{Utilization: 100, Score: 0},
//						},
//					},
//				},
//			},
//			runPreScore: true,
//		},
//		{
//			name: "test case for ScoringStrategy RequestedToCapacityRatio case2",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "3000", "memory": "5000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 95}, {Name: "runner2", Score: 68}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: defaultResources,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 0},
//							{Utilization: 100, Score: 10},
//						},
//					},
//				},
//			},
//			runPreScore: true,
//		},
//		{
//			name: "test case for ScoringStrategy MostAllocated",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 67}, {Name: "runner2", Score: 36}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.MostAllocated,
//					Resources: defaultResources,
//				},
//			},
//			runPreScore: true,
//		},
//		{
//			name: "test case for ScoringStrategy LeastAllocated",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 32}, {Name: "runner2", Score: 63}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.LeastAllocated,
//					Resources: defaultResources,
//				},
//			},
//			runPreScore: true,
//		},
//		{
//			name: "test case for ScoringStrategy RequestedToCapacityRatio case1 if PreScore is not called",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "3000", "memory": "5000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 10}, {Name: "runner2", Score: 32}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: defaultResources,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 10},
//							{Utilization: 100, Score: 0},
//						},
//					},
//				},
//			},
//			runPreScore: false,
//		},
//		{
//			name: "test case for ScoringStrategy MostAllocated if PreScore is not called",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 67}, {Name: "runner2", Score: 36}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.MostAllocated,
//					Resources: defaultResources,
//				},
//			},
//			runPreScore: false,
//		},
//		{
//			name: "test case for ScoringStrategy LeastAllocated if PreScore is not called",
//			requestedRegion: st.MakeRegion().
//				Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).
//				Obj(),
//			runners: []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//				st.MakeRunner().Name("runner2").Capacity(map[corev1.ResourceName]string{"cpu": "6000", "memory": "10000"}).Obj(),
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//				st.MakeRegion().Runner("runner2").Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj(),
//			},
//			expectedPriorities: []framework.RunnerScore{{Name: "runner1", Score: 32}, {Name: "runner2", Score: 63}},
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.LeastAllocated,
//					Resources: defaultResources,
//				},
//			},
//			runPreScore: false,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			state := framework.NewCycleState()
//			snapshot := cache.NewSnapshot(test.existingRegions, test.runners)
//			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(snapshot))
//			args := test.runnerResourcesFitArgs
//			p, err := NewFit(ctx, &args, fh, plfeature.Features{})
//			if err != nil {
//				t.Fatalf("unexpected error: %v", err)
//			}
//
//			var gotPriorities framework.RunnerScoreList
//			for _, n := range test.runners {
//				if test.runPreScore {
//					status := p.(framework.PreScorePlugin).PreScore(ctx, state, test.requestedRegion, tf.BuildRunnerInfos(test.runners))
//					if !status.IsSuccess() {
//						t.Errorf("PreScore is expected to return success, but didn't. Got status: %v", status)
//					}
//				}
//				score, status := p.(framework.ScorePlugin).Score(ctx, state, test.requestedRegion, n.Name)
//				if !status.IsSuccess() {
//					t.Errorf("Score is expected to return success, but didn't. Got status: %v", status)
//				}
//				gotPriorities = append(gotPriorities, framework.RunnerScore{Name: n.Name, Score: score})
//			}
//
//			if !reflect.DeepEqual(test.expectedPriorities, gotPriorities) {
//				t.Errorf("expected:\n\t%+v,\ngot:\n\t%+v", test.expectedPriorities, gotPriorities)
//			}
//		})
//	}
//}
//
//var benchmarkResourceSet = []config.ResourceSpec{
//	{Name: string(corev1.ResourceCPU), Weight: 1},
//	{Name: string(corev1.ResourceMemory), Weight: 1},
//	{Name: string(corev1.ResourceRegions), Weight: 1},
//	{Name: string(corev1.ResourceStorage), Weight: 1},
//	{Name: string(corev1.ResourceEphemeralStorage), Weight: 1},
//	{Name: string(extendedResourceA), Weight: 1},
//	{Name: string(extendedResourceB), Weight: 1},
//	{Name: string(kubernetesIOResourceA), Weight: 1},
//	{Name: string(kubernetesIOResourceB), Weight: 1},
//	{Name: string(hugePageResourceA), Weight: 1},
//}
//
//func BenchmarkTestFitScore(b *testing.B) {
//	tests := []struct {
//		name                   string
//		runnerResourcesFitArgs config.RunnerResourcesFitArgs
//	}{
//		{
//			name: "RequestedToCapacityRatio with defaultResources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: defaultResources,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 10},
//							{Utilization: 100, Score: 0},
//						},
//					},
//				},
//			},
//		},
//		{
//			name: "RequestedToCapacityRatio with 10 resources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.RequestedToCapacityRatio,
//					Resources: benchmarkResourceSet,
//					RequestedToCapacityRatio: &config.RequestedToCapacityRatioParam{
//						Shape: []config.UtilizationShapePoint{
//							{Utilization: 0, Score: 10},
//							{Utilization: 100, Score: 0},
//						},
//					},
//				},
//			},
//		},
//		{
//			name: "MostAllocated with defaultResources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.MostAllocated,
//					Resources: defaultResources,
//				},
//			},
//		},
//		{
//			name: "MostAllocated with 10 resources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.MostAllocated,
//					Resources: benchmarkResourceSet,
//				},
//			},
//		},
//		{
//			name: "LeastAllocated with defaultResources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.LeastAllocated,
//					Resources: defaultResources,
//				},
//			},
//		},
//		{
//			name: "LeastAllocated with 10 resources",
//			runnerResourcesFitArgs: config.RunnerResourcesFitArgs{
//				ScoringStrategy: &config.ScoringStrategy{
//					Type:      config.LeastAllocated,
//					Resources: benchmarkResourceSet,
//				},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		b.Run(test.name, func(b *testing.B) {
//			_, ctx := ktesting.NewTestContext(b)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			existingRegions := []*corev1.Region{
//				st.MakeRegion().Runner("runner1").Req(map[corev1.ResourceName]string{"cpu": "2000", "memory": "4000"}).Obj(),
//			}
//			runners := []*corev1.Runner{
//				st.MakeRunner().Name("runner1").Capacity(map[corev1.ResourceName]string{"cpu": "4000", "memory": "10000"}).Obj(),
//			}
//			state := framework.NewCycleState()
//			var runnerResourcesFunc = runtime.FactoryAdapter(plfeature.Features{}, NewFit)
//			pl := plugintesting.SetupPlugin(ctx, b, runnerResourcesFunc, &test.runnerResourcesFitArgs, cache.NewSnapshot(existingRegions, runners))
//			p := pl.(*Fit)
//
//			b.ResetTimer()
//
//			requestedRegion := st.MakeRegion().Req(map[corev1.ResourceName]string{"cpu": "1000", "memory": "2000"}).Obj()
//			for i := 0; i < b.N; i++ {
//				_, status := p.Score(ctx, state, requestedRegion, runners[0].Name)
//				if !status.IsSuccess() {
//					b.Errorf("unexpected status: %v", status)
//				}
//			}
//		})
//	}
//}
//
//func TestEventsToRegister(t *testing.T) {
//	tests := []struct {
//		name                                string
//		inPlaceRegionVerticalScalingEnabled bool
//		expectedClusterEvents               []framework.ClusterEventWithHint
//	}{
//		{
//			"Register events with InPlaceRegionVerticalScaling feature enabled",
//			true,
//			[]framework.ClusterEventWithHint{
//				{Event: framework.ClusterEvent{Resource: "Region", ActionType: framework.Update | framework.Delete}},
//				{Event: framework.ClusterEvent{Resource: "Runner", ActionType: framework.Add | framework.Update}},
//			},
//		},
//		{
//			"Register events with InPlaceRegionVerticalScaling feature disabled",
//			false,
//			[]framework.ClusterEventWithHint{
//				{Event: framework.ClusterEvent{Resource: "Region", ActionType: framework.Delete}},
//				{Event: framework.ClusterEvent{Resource: "Runner", ActionType: framework.Add | framework.Update}},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			fp := &Fit{enableInPlaceRegionVerticalScaling: test.inPlaceRegionVerticalScalingEnabled}
//			actualClusterEvents := fp.EventsToRegister()
//			for i := range actualClusterEvents {
//				actualClusterEvents[i].QueueingHintFn = nil
//			}
//			if diff := cmp.Diff(test.expectedClusterEvents, actualClusterEvents); diff != "" {
//				t.Error("Cluster Events doesn't match extected events (-expected +actual):\n", diff)
//			}
//		})
//	}
//}
//
//func Test_isSchedulableAfterRegionChange(t *testing.T) {
//	testcases := map[string]struct {
//		region                             *corev1.Region
//		oldObj, newObj                     interface{}
//		enableInPlaceRegionVerticalScaling bool
//		expectedHint                       framework.QueueingHint
//		expectedErr                        bool
//	}{
//		"backoff-wrong-old-object": {
//			region:                             &corev1.Region{},
//			oldObj:                             "not-a-region",
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.Queue,
//			expectedErr:                        true,
//		},
//		"backoff-wrong-new-object": {
//			region:                             &corev1.Region{},
//			newObj:                             "not-a-region",
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.Queue,
//			expectedErr:                        true,
//		},
//		"queue-on-deleted": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Runner("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.Queue,
//		},
//		"skip-queue-on-unscheduled-region-deleted": {
//			region:                             &corev1.Region{},
//			oldObj:                             &corev1.Region{},
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"skip-queue-on-disable-inplace-region-vertical-scaling": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Runners("fake").Obj(),
//			newObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Runners("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: false,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"skip-queue-on-unscheduled-region": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Obj(),
//			newObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"skip-queue-on-non-resource-changes": {
//			region:                             &corev1.Region{},
//			oldObj:                             st.MakeRegion().Label("k", "v").Runner("fake").Obj(),
//			newObj:                             st.MakeRegion().Label("foo", "bar").Runner("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"skip-queue-on-unrelated-resource-changes": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceMemory: "2"}).Runner("fake").Obj(),
//			newObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceMemory: "1"}).Runner("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"skip-queue-on-resource-scale-up": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Runner("fake").Obj(),
//			newObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Runner("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.QueueSkip,
//		},
//		"queue-on-some-resource-scale-down": {
//			region:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			oldObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Runner("fake").Obj(),
//			newObj:                             st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Runner("fake").Obj(),
//			enableInPlaceRegionVerticalScaling: true,
//			expectedHint:                       framework.Queue,
//		},
//	}
//
//	for name, tc := range testcases {
//		t.Run(name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{
//				EnableInPlaceRegionVerticalScaling: tc.enableInPlaceRegionVerticalScaling,
//			})
//			if err != nil {
//				t.Fatal(err)
//			}
//			actualHint, err := p.(*Fit).isSchedulableAfterRegionChange(logger, tc.region, tc.oldObj, tc.newObj)
//			if tc.expectedErr {
//				require.Error(t, err)
//				return
//			}
//			require.NoError(t, err)
//			require.Equal(t, tc.expectedHint, actualHint)
//		})
//	}
//}
//
//func Test_isSchedulableAfterRunnerChange(t *testing.T) {
//	testcases := map[string]struct {
//		region         *corev1.Region
//		oldObj, newObj interface{}
//		expectedHint   framework.QueueingHint
//		expectedErr    bool
//	}{
//		"backoff-wrong-new-object": {
//			region:       &corev1.Region{},
//			newObj:       "not-a-runner",
//			expectedHint: framework.Queue,
//			expectedErr:  true,
//		},
//		"backoff-wrong-old-object": {
//			region:       &corev1.Region{},
//			oldObj:       "not-a-runner",
//			newObj:       &corev1.Runner{},
//			expectedHint: framework.Queue,
//			expectedErr:  true,
//		},
//		"skip-queue-on-runner-add-without-sufficient-resources": {
//			region: newResourceRegion(framework.Resource{Memory: 2}),
//			newObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "1",
//			}).Obj(),
//			expectedHint: framework.QueueSkip,
//		},
//		"skip-queue-on-runner-add-without-required-resource-type": {
//			region: newResourceRegion(framework.Resource{
//				ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1}},
//			),
//			newObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				extendedResourceB: "1",
//			}).Obj(),
//			expectedHint: framework.QueueSkip,
//		},
//		"queue-on-runner-add-with-sufficient-resources": {
//			region: newResourceRegion(framework.Resource{
//				Memory:          2,
//				ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1},
//			}),
//			newObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "4",
//				extendedResourceA: "2",
//			}).Obj(),
//			expectedHint: framework.Queue,
//		},
//		// uncomment this case when the isSchedulableAfterRunnerChange also check the
//		// original runner's resources.
//		// "skip-queue-on-runner-unrelated-changes": {
//		// 	region:          &corev1.Region{},
//		// 	oldObj:       st.MakeRunner().Obj(),
//		// 	newObj:       st.MakeRunner().Label("foo", "bar").Obj(),
//		// 	expectedHint: framework.QueueSkip,
//		// },
//		"skip-queue-on-runner-changes-from-suitable-to-unsuitable": {
//			region: newResourceRegion(framework.Resource{
//				Memory:          2,
//				ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1},
//			}),
//			oldObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "4",
//				extendedResourceA: "2",
//			}).Obj(),
//			newObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "1",
//				extendedResourceA: "2",
//			}).Obj(),
//			expectedHint: framework.QueueSkip,
//		},
//		"queue-on-runner-changes-from-unsuitable-to-suitable": {
//			region: newResourceRegion(framework.Resource{
//				Memory:          2,
//				ScalarResources: map[corev1.ResourceName]int64{extendedResourceA: 1},
//			}),
//			oldObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "1",
//				extendedResourceA: "2",
//			}).Obj(),
//			newObj: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceMemory: "4",
//				extendedResourceA: "2",
//			}).Obj(),
//			expectedHint: framework.Queue,
//		},
//	}
//
//	for name, tc := range testcases {
//		t.Run(name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			p, err := NewFit(ctx, &config.RunnerResourcesFitArgs{ScoringStrategy: defaultScoringStrategy}, nil, plfeature.Features{})
//			if err != nil {
//				t.Fatal(err)
//			}
//			actualHint, err := p.(*Fit).isSchedulableAfterRunnerChange(logger, tc.region, tc.oldObj, tc.newObj)
//			if tc.expectedErr {
//				require.Error(t, err)
//				return
//			}
//			require.NoError(t, err)
//			require.Equal(t, tc.expectedHint, actualHint)
//		})
//	}
//}
//
//func TestIsFit(t *testing.T) {
//	testCases := map[string]struct {
//		region   *corev1.Region
//		runner   *corev1.Runner
//		expected bool
//	}{
//		"nil runner": {
//			region:   &corev1.Region{},
//			expected: false,
//		},
//		"insufficient resource": {
//			region:   st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Obj(),
//			runner:   st.MakeRunner().Capacity(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			expected: false,
//		},
//		"sufficient resource": {
//			region:   st.MakeRegion().Req(map[corev1.ResourceName]string{corev1.ResourceCPU: "1"}).Obj(),
//			runner:   st.MakeRunner().Capacity(map[corev1.ResourceName]string{corev1.ResourceCPU: "2"}).Obj(),
//			expected: true,
//		},
//	}
//
//	for name, tc := range testCases {
//		t.Run(name, func(t *testing.T) {
//			if got := isFit(tc.region, tc.runner); got != tc.expected {
//				t.Errorf("expected: %v, got: %v", tc.expected, got)
//			}
//		})
//	}
//}
