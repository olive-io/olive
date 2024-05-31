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

// Package resources provides a metrics collector that reports the
// resource consumption (requests and limits) of the regions in the cluster
// as the scheduler and olive-runner would interpret it.
package resources

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/testutil"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

type fakeRegionLister struct {
	regions []*corev1.Region
}

func (l *fakeRegionLister) List(selector labels.Selector) (ret []*corev1.Region, err error) {
	return l.regions, nil
}

func (l *fakeRegionLister) Get(name string) (*corev1.Region, error) {
	panic("implement me")
}

func Test_regionResourceCollector_Handler(t *testing.T) {
	h := Handler(&fakeRegionLister{regions: []*corev1.Region{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
			Spec:       corev1.RegionSpec{},
			Status:     corev1.RegionStatus{},
		},
	}})

	r := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/metrics/resources", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(r, req)

	expected := `# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
# TYPE olive_region_resource_limit gauge
olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 6
olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2.68435456e+09
# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
# TYPE olive_region_resource_request gauge
olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 3
`
	out := r.Body.String()
	if expected != out {
		t.Fatal(out)
	}
}

func Test_regionResourceCollector_CollectWithStability(t *testing.T) {
	tests := []struct {
		name string

		regions  []*corev1.Region
		expected string
	}{
		{},
		{
			name: "no containers",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
				},
			},
		},
		{
			name: "no resources",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
		},
		{
			name: "request only",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "limits only",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `      
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "terminal regions are excluded",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-unscheduled-succeeded"},
					Spec:       corev1.RegionSpec{},
					// until runner name is set, phase is ignored
					Status: corev1.RegionStatus{Phase: corev1.RegionSucceeded},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-succeeded"},
					Spec:       corev1.RegionSpec{},
					Status:     corev1.RegionStatus{Phase: corev1.RegionSucceeded},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-failed"},
					Spec:       corev1.RegionSpec{},
					Status:     corev1.RegionStatus{Phase: corev1.RegionFailed},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-pending"},
					Spec:       corev1.RegionSpec{},
					Status:     corev1.RegionStatus{},
				},
			},
			expected: `
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="",region="foo-unscheduled-succeeded",priority="",resource="cpu",scheduler="",unit="cores"} 1
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo-pending",priority="",resource="cpu",scheduler="",unit="cores"} 1
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo-unknown",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "zero resource should be excluded",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: ``,
		},
		{
			name: "optional field labels",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="0",resource="cpu",scheduler="default-scheduler",unit="cores"} 1
				`,
		},
		{
			name: "init containers and regular containers when initialized",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
					Status:     corev1.RegionStatus{},
				},
			},
			expected: `
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 6
				olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2e+09
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 3
				`,
		},
		{
			name: "init containers and regular containers when initializing",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
					Status:     corev1.RegionStatus{},
				},
			},
			expected: `
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 6
				olive_region_resource_limit{namespace="test",runner="runner-one",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2e+09
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
				olive_region_resource_request{namespace="test",runner="runner-one",region="foo",priority="",resource="custom",scheduler="",unit=""} 3
				`,
		},
		{
			name: "aggregate container requests and limits",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `            
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 3.25
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 4e+09
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1.5
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 1e+09
				`,
		},
		{
			name: "overhead added to requests and limits",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="custom",scheduler="",unit=""} 6.5
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2.75e+09
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2.25
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="custom",scheduler="",unit=""} 3.5
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="memory",scheduler="",unit="bytes"} 7.5e+08
				`,
		},
		{
			name: "units for standard resources",
			regions: []*corev1.Region{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.RegionSpec{},
				},
			},
			expected: `
				# HELP olive_region_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_limit gauge
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="attachable-volumes-",scheduler="",unit="integer"} 4
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="attachable-volumes-aws",scheduler="",unit="integer"} 3
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="hugepages-",scheduler="",unit="bytes"} 2
				olive_region_resource_limit{namespace="test",runner="",region="foo",priority="",resource="hugepages-x",scheduler="",unit="bytes"} 1
				# HELP olive_region_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by region. This shows the resource usage the scheduler and olivelet expect per region for resources along with the unit for the resource if any.
				# TYPE olive_region_resource_request gauge
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="ephemeral-storage",scheduler="",unit="bytes"} 6
				olive_region_resource_request{namespace="test",runner="",region="foo",priority="",resource="storage",scheduler="",unit="bytes"} 5
				`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRegionResourcesMetricsCollector(&fakeRegionLister{regions: tt.regions})
			registry := metrics.NewKubeRegistry()
			registry.CustomMustRegister(c)
			err := testutil.GatherAndCompare(registry, strings.NewReader(tt.expected))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
