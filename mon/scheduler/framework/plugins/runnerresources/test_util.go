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

package runnerresources

import (
	"github.com/google/go-cmp/cmp/cmpopts"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var (
	ignoreBadValueDetail = cmpopts.IgnoreFields(field.Error{}, "BadValue", "Detail")
	defaultResources     = []config.ResourceSpec{
		{Name: string(corev1.ResourceCPU), Weight: 1},
		{Name: string(corev1.ResourceMemory), Weight: 1},
	}
	extendedRes         = "abc.com/xyz"
	extendedResourceSet = []config.ResourceSpec{
		{Name: string(corev1.ResourceCPU), Weight: 1},
		{Name: string(corev1.ResourceMemory), Weight: 1},
		{Name: extendedRes, Weight: 1},
	}
)

func makeRunner(runner string, milliCPU, memory int64, extendedResource map[string]int64) *corev1.Runner {
	resourceList := make(map[corev1.ResourceName]resource.Quantity)
	for res, quantity := range extendedResource {
		resourceList[corev1.ResourceName(res)] = *resource.NewQuantity(quantity, resource.DecimalSI)
	}
	resourceList[corev1.ResourceCPU] = *resource.NewMilliQuantity(milliCPU, resource.DecimalSI)
	resourceList[corev1.ResourceMemory] = *resource.NewQuantity(memory, resource.BinarySI)
	return &corev1.Runner{
		ObjectMeta: metav1.ObjectMeta{Name: runner},
		Status: corev1.RunnerStatus{
			Capacity:    resourceList,
			Allocatable: resourceList,
		},
	}
}
