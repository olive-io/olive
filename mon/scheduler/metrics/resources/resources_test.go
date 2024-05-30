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
// resource consumption (requests and limits) of the definitions in the cluster
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
	corelisters "github.com/olive-io/olive/client/generated/listers/core/v1"
)

type fakeDefinitionLister struct {
	definitions []*corev1.Definition
}

func (l *fakeDefinitionLister) List(selector labels.Selector) (ret []*corev1.Definition, err error) {
	return l.definitions, nil
}

func (l *fakeDefinitionLister) Definitions(namespace string) corelisters.DefinitionNamespaceLister {
	panic("not implemented")
}

func Test_definitionResourceCollector_Handler(t *testing.T) {
	h := Handler(&fakeDefinitionLister{definitions: []*corev1.Definition{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
			Spec:       corev1.DefinitionSpec{},
			Status:     corev1.DefinitionStatus{},
		},
	}})

	r := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/metrics/resources", nil)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(r, req)

	expected := `# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
# TYPE olive_definition_resource_limit gauge
olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 6
olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2.68435456e+09
# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
# TYPE olive_definition_resource_request gauge
olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 3
`
	out := r.Body.String()
	if expected != out {
		t.Fatal(out)
	}
}

func Test_definitionResourceCollector_CollectWithStability(t *testing.T) {
	tests := []struct {
		name string

		definitions []*corev1.Definition
		expected    string
	}{
		{},
		{
			name: "no containers",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
				},
			},
		},
		{
			name: "no resources",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
		},
		{
			name: "request only",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "limits only",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `      
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "terminal definitions are excluded",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-unscheduled-succeeded"},
					Spec:       corev1.DefinitionSpec{},
					// until node name is set, phase is ignored
					Status: corev1.DefinitionStatus{Phase: corev1.DefSucceeded},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-succeeded"},
					Spec:       corev1.DefinitionSpec{},
					Status:     corev1.DefinitionStatus{Phase: corev1.DefSucceeded},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-failed"},
					Spec:       corev1.DefinitionSpec{},
					Status:     corev1.DefinitionStatus{Phase: corev1.DefFailed},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo-pending"},
					Spec:       corev1.DefinitionSpec{},
					Status:     corev1.DefinitionStatus{},
				},
			},
			expected: `
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="",definition="foo-unscheduled-succeeded",priority="",resource="cpu",scheduler="",unit="cores"} 1
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo-pending",priority="",resource="cpu",scheduler="",unit="cores"} 1
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo-unknown",priority="",resource="cpu",scheduler="",unit="cores"} 1
				`,
		},
		{
			name: "zero resource should be excluded",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: ``,
		},
		{
			name: "optional field labels",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="0",resource="cpu",scheduler="default-scheduler",unit="cores"} 1
				`,
		},
		{
			name: "init containers and regular containers when initialized",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
					Status:     corev1.DefinitionStatus{},
				},
			},
			expected: `
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 6
				olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2e+09
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 3
				`,
		},
		{
			name: "init containers and regular containers when initializing",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
					Status:     corev1.DefinitionStatus{},
				},
			},
			expected: `
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 6
				olive_definition_resource_limit{namespace="test",node="node-one",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2e+09
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2
				olive_definition_resource_request{namespace="test",node="node-one",definition="foo",priority="",resource="custom",scheduler="",unit=""} 3
				`,
		},
		{
			name: "aggregate container requests and limits",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `            
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 3.25
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 4e+09
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 1.5
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 1e+09
				`,
		},
		{
			name: "overhead added to requests and limits",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="custom",scheduler="",unit=""} 6.5
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 2.75e+09
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="cpu",scheduler="",unit="cores"} 2.25
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="custom",scheduler="",unit=""} 3.5
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="memory",scheduler="",unit="bytes"} 7.5e+08
				`,
		},
		{
			name: "units for standard resources",
			definitions: []*corev1.Definition{
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "foo"},
					Spec:       corev1.DefinitionSpec{},
				},
			},
			expected: `
				# HELP olive_definition_resource_limit [STABLE] Resources limit for workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_limit gauge
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="attachable-volumes-",scheduler="",unit="integer"} 4
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="attachable-volumes-aws",scheduler="",unit="integer"} 3
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="hugepages-",scheduler="",unit="bytes"} 2
				olive_definition_resource_limit{namespace="test",node="",definition="foo",priority="",resource="hugepages-x",scheduler="",unit="bytes"} 1
				# HELP olive_definition_resource_request [STABLE] Resources requested by workloads on the cluster, broken down by definition. This shows the resource usage the scheduler and olivelet expect per definition for resources along with the unit for the resource if any.
				# TYPE olive_definition_resource_request gauge
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="ephemeral-storage",scheduler="",unit="bytes"} 6
				olive_definition_resource_request{namespace="test",node="",definition="foo",priority="",resource="storage",scheduler="",unit="bytes"} 5
				`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewDefinitionResourcesMetricsCollector(&fakeDefinitionLister{definitions: tt.definitions})
			registry := metrics.NewKubeRegistry()
			registry.CustomMustRegister(c)
			err := testutil.GatherAndCompare(registry, strings.NewReader(tt.expected))
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
