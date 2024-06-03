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

package helper

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/olive-io/olive/apis/core"
)

func TestSemantic(t *testing.T) {
	table := []struct {
		a, b        interface{}
		shouldEqual bool
	}{
		{resource.MustParse("0"), resource.Quantity{}, true},
		{resource.Quantity{}, resource.MustParse("0"), true},
		{resource.Quantity{}, resource.MustParse("1m"), false},
		{
			resource.NewQuantity(5, resource.BinarySI),
			resource.NewQuantity(5, resource.DecimalSI),
			true,
		},
		{resource.MustParse("2m"), resource.MustParse("1m"), false},
	}

	for index, item := range table {
		if e, a := item.shouldEqual, Semantic.DeepEqual(item.a, item.b); e != a {
			t.Errorf("case[%d], expected %v, got %v.", index, e, a)
		}
	}
}

func TestIsStandardContainerResource(t *testing.T) {
	testCases := []struct {
		input  string
		output bool
	}{
		{"cpu", true},
		{"memory", true},
		{"disk", false},
		{"hugepages-2Mi", true},
	}
	for i, tc := range testCases {
		if IsStandardContainerResourceName(core.ResourceName(tc.input)) != tc.output {
			t.Errorf("case[%d], input: %s, expected: %t, got: %t", i, tc.input, tc.output, !tc.output)
		}
	}
}

func TestIsHugePageResourceName(t *testing.T) {
	testCases := []struct {
		name   core.ResourceName
		result bool
	}{
		{
			name:   core.ResourceName("hugepages-2Mi"),
			result: true,
		},
		{
			name:   core.ResourceName("hugepages-1Gi"),
			result: true,
		},
		{
			name:   core.ResourceName("cpu"),
			result: false,
		},
		{
			name:   core.ResourceName("memory"),
			result: false,
		},
	}
	for _, testCase := range testCases {
		if testCase.result != IsHugePageResourceName(testCase.name) {
			t.Errorf("resource: %v expected result: %v", testCase.name, testCase.result)
		}
	}
}

func TestIsHugePageResourceValueDivisible(t *testing.T) {
	testCases := []struct {
		name     core.ResourceName
		quantity resource.Quantity
		result   bool
	}{
		{
			name:     core.ResourceName("hugepages-2Mi"),
			quantity: resource.MustParse("4Mi"),
			result:   true,
		},
		{
			name:     core.ResourceName("hugepages-2Mi"),
			quantity: resource.MustParse("5Mi"),
			result:   false,
		},
		{
			name:     core.ResourceName("hugepages-1Gi"),
			quantity: resource.MustParse("2Gi"),
			result:   true,
		},
		{
			name:     core.ResourceName("hugepages-1Gi"),
			quantity: resource.MustParse("2.1Gi"),
			result:   false,
		},
		{
			name:     core.ResourceName("hugepages-1Mi"),
			quantity: resource.MustParse("2.1Mi"),
			result:   false,
		},
		{
			name:     core.ResourceName("hugepages-64Ki"),
			quantity: resource.MustParse("128Ki"),
			result:   true,
		},
		{
			name:     core.ResourceName("hugepages-"),
			quantity: resource.MustParse("128Ki"),
			result:   false,
		},
		{
			name:     core.ResourceName("hugepages"),
			quantity: resource.MustParse("128Ki"),
			result:   false,
		},
	}
	for _, testCase := range testCases {
		if testCase.result != IsHugePageResourceValueDivisible(testCase.name, testCase.quantity) {
			t.Errorf("resource: %v storage:%v expected result: %v", testCase.name, testCase.quantity, testCase.result)
		}
	}
}

func TestHugePageSizeFromResourceName(t *testing.T) {
	testCases := []struct {
		name      core.ResourceName
		expectErr bool
		pageSize  resource.Quantity
	}{
		{
			name:      core.ResourceName("hugepages-2Mi"),
			pageSize:  resource.MustParse("2Mi"),
			expectErr: false,
		},
		{
			name:      core.ResourceName("hugepages-1Gi"),
			pageSize:  resource.MustParse("1Gi"),
			expectErr: false,
		},
		{
			name:      core.ResourceName("hugepages-bad"),
			expectErr: true,
		},
	}
	for _, testCase := range testCases {
		value, err := HugePageSizeFromResourceName(testCase.name)
		if testCase.expectErr && err == nil {
			t.Errorf("Expected an error for %v", testCase.name)
		} else if !testCase.expectErr && err != nil {
			t.Errorf("Unexpected error for %v, got %v", testCase.name, err)
		} else if testCase.pageSize.Value() != value.Value() {
			t.Errorf("Unexpected pageSize for resource %v got %v", testCase.name, value.String())
		}
	}
}

func TestIsOvercommitAllowed(t *testing.T) {
	testCases := []struct {
		name    core.ResourceName
		allowed bool
	}{
		{
			name:    core.ResourceCPU,
			allowed: true,
		},
		{
			name:    core.ResourceMemory,
			allowed: true,
		},
		{
			name:    HugePageResourceName(resource.MustParse("2Mi")),
			allowed: false,
		},
	}
	for _, testCase := range testCases {
		if testCase.allowed != IsOvercommitAllowed(testCase.name) {
			t.Errorf("Unexpected result for %v", testCase.name)
		}
	}
}
