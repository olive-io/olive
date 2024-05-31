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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetNonZeroRequest(t *testing.T) {
	tests := []struct {
		name           string
		requests       v1.ResourceList
		expectedCPU    int64
		expectedMemory int64
	}{
		{
			"cpu_and_memory_not_found",
			v1.ResourceList{},
			DefaultMilliCPURequest,
			DefaultMemoryRequest,
		},
		{
			"only_cpu_exist",
			v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("200m"),
			},
			200,
			DefaultMemoryRequest,
		},
		{
			"only_memory_exist",
			v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("400Mi"),
			},
			DefaultMilliCPURequest,
			400 * 1024 * 1024,
		},
		{
			"cpu_memory_exist",
			v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse("200m"),
				v1.ResourceMemory: resource.MustParse("400Mi"),
			},
			200,
			400 * 1024 * 1024,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			realCPU, realMemory := GetNonzeroRequests(&test.requests)
			assert.EqualValuesf(t, test.expectedCPU, realCPU, "Failed to test: %s", test.name)
			assert.EqualValuesf(t, test.expectedMemory, realMemory, "Failed to test: %s", test.name)
		})
	}
}

func TestGetRequestForResource(t *testing.T) {
	tests := []struct {
		name             string
		requests         v1.ResourceList
		resource         v1.ResourceName
		expectedQuantity int64
		nonZero          bool
	}{
		{
			"extended_resource_not_found",
			v1.ResourceList{},
			v1.ResourceName("intel.com/foo"),
			0,
			true,
		},
		{
			"extended_resource_found",
			v1.ResourceList{
				v1.ResourceName("intel.com/foo"): resource.MustParse("4"),
			},
			v1.ResourceName("intel.com/foo"),
			4,
			true,
		},
		{
			"cpu_not_found",
			v1.ResourceList{},
			v1.ResourceCPU,
			DefaultMilliCPURequest,
			true,
		},
		{
			"memory_not_found",
			v1.ResourceList{},
			v1.ResourceMemory,
			DefaultMemoryRequest,
			true,
		},
		{
			"cpu_exist",
			v1.ResourceList{
				v1.ResourceCPU: resource.MustParse("200m"),
			},
			v1.ResourceCPU,
			200,
			true,
		},
		{
			"memory_exist",
			v1.ResourceList{
				v1.ResourceMemory: resource.MustParse("400Mi"),
			},
			v1.ResourceMemory,
			400 * 1024 * 1024,
			true,
		},
		{
			"ephemeralStorage_exist",
			v1.ResourceList{
				v1.ResourceEphemeralStorage: resource.MustParse("400Mi"),
			},
			v1.ResourceEphemeralStorage,
			400 * 1024 * 1024,
			true,
		},
		{
			"ephemeralStorage_not_found",
			v1.ResourceList{},
			v1.ResourceEphemeralStorage,
			0,
			true,
		},
		{
			"cpu_not_found, useRequested is true",
			v1.ResourceList{},
			v1.ResourceCPU,
			0,
			false,
		},
		{
			"memory_not_found, useRequested is true",
			v1.ResourceList{},
			v1.ResourceMemory,
			0,
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			realQuantity := GetRequestForResource(test.resource, &test.requests, test.nonZero)
			var realQuantityI64 int64
			if test.resource == v1.ResourceCPU {
				realQuantityI64 = realQuantity.MilliValue()
			} else {
				realQuantityI64 = realQuantity.Value()
			}
			assert.EqualValuesf(t, test.expectedQuantity, realQuantityI64, "Failed to test: %s", test.name)
		})
	}
}
