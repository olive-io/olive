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

package v1

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// TestCompatibility verifies that the types in extender/v1 can be successfully encoded to json and decoded back, even when lowercased,
// since these types were written around JSON tags and we need to enforce consistency on them now.
// @TODO(88634): v2 of these types should be defined with proper JSON tags to enforce field casing to a single approach
func TestCompatibility(t *testing.T) {
	testcases := []struct {
		emptyObj   interface{}
		obj        interface{}
		expectJSON string
	}{
		{
			emptyObj: &ExtenderPreemptionResult{},
			obj: &ExtenderPreemptionResult{
				RunnerNameToMetaVictims: map[string]*MetaVictims{"foo": {Regions: []*MetaRegion{{UID: "myuid"}}, NumPDBViolations: 1}},
			},
			expectJSON: `{"RunnerNameToMetaVictims":{"foo":{"Regions":[{"UID":"myuid"}],"NumPDBViolations":1}}}`,
		},
		{
			emptyObj: &ExtenderPreemptionArgs{},
			obj: &ExtenderPreemptionArgs{
				Region:                  &corev1.Region{ObjectMeta: metav1.ObjectMeta{Name: "podname"}},
				RunnerNameToVictims:     map[string]*Victims{"foo": {Regions: []*corev1.Region{{ObjectMeta: metav1.ObjectMeta{Name: "podname"}}}, NumPDBViolations: 1}},
				RunnerNameToMetaVictims: map[string]*MetaVictims{"foo": {Regions: []*MetaRegion{{UID: "myuid"}}, NumPDBViolations: 1}},
			},
			expectJSON: `{"Region":{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}},"RunnerNameToVictims":{"foo":{"Regions":[{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}}],"NumPDBViolations":1}},"RunnerNameToMetaVictims":{"foo":{"Regions":[{"UID":"myuid"}],"NumPDBViolations":1}}}`,
		},
		{
			emptyObj: &ExtenderArgs{},
			obj: &ExtenderArgs{
				Region:      &corev1.Region{ObjectMeta: metav1.ObjectMeta{Name: "podname"}},
				Runners:     &corev1.RunnerList{Items: []corev1.Runner{{ObjectMeta: metav1.ObjectMeta{Name: "nodename"}}}},
				RunnerNames: &[]string{"node1"},
			},
			expectJSON: `{"Region":{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}},"Runners":{"metadata":{},"items":[{"metadata":{"name":"nodename","creationTimestamp":null},"spec":{},"status":{"daemonEndpoints":{"kubeletEndpoint":{"Port":0}},"nodeInfo":{"machineID":"","systemUUID":"","bootID":"","kernelVersion":"","osImage":"","containerRuntimeVersion":"","kubeletVersion":"","kubeProxyVersion":"","operatingSystem":"","architecture":""}}}]},"RunnerNames":["node1"]}`,
		},
		{
			emptyObj: &ExtenderFilterResult{},
			obj: &ExtenderFilterResult{
				Runners:                      &corev1.RunnerList{Items: []corev1.Runner{{ObjectMeta: metav1.ObjectMeta{Name: "nodename"}}}},
				RunnerNames:                  &[]string{"node1"},
				FailedRunners:                FailedRunnersMap{"foo": "bar"},
				FailedAndUnresolvableRunners: FailedRunnersMap{"baz": "qux"},
				Error:                        "myerror",
			},
			expectJSON: `{"Runners":{"metadata":{},"items":[{"metadata":{"name":"nodename","creationTimestamp":null},"spec":{},"status":{"daemonEndpoints":{"kubeletEndpoint":{"Port":0}},"nodeInfo":{"machineID":"","systemUUID":"","bootID":"","kernelVersion":"","osImage":"","containerRuntimeVersion":"","kubeletVersion":"","kubeProxyVersion":"","operatingSystem":"","architecture":""}}}]},"RunnerNames":["node1"],"FailedRunners":{"foo":"bar"},"FailedAndUnresolvableRunners":{"baz":"qux"},"Error":"myerror"}`,
		},
		{
			emptyObj: &ExtenderBindingArgs{},
			obj: &ExtenderBindingArgs{
				RegionName:      "mypodname",
				RegionNamespace: "mypodnamespace",
				RegionUID:       types.UID("mypoduid"),
				Runner:          "mynode",
			},
			expectJSON: `{"RegionName":"mypodname","RegionNamespace":"mypodnamespace","RegionUID":"mypoduid","Runner":"mynode"}`,
		},
		{
			emptyObj:   &ExtenderBindingResult{},
			obj:        &ExtenderBindingResult{Error: "myerror"},
			expectJSON: `{"Error":"myerror"}`,
		},
		{
			emptyObj:   &HostPriority{},
			obj:        &HostPriority{Host: "myhost", Score: 1},
			expectJSON: `{"Host":"myhost","Score":1}`,
		},
	}

	for _, tc := range testcases {
		t.Run(reflect.TypeOf(tc.obj).String(), func(t *testing.T) {
			data, err := json.Marshal(tc.obj)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != tc.expectJSON {
				t.Fatalf("expected %s, got %s", tc.expectJSON, string(data))
			}
			if err := json.Unmarshal([]byte(strings.ToLower(string(data))), tc.emptyObj); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.emptyObj, tc.obj) {
				t.Fatalf("round-tripped case-insensitive diff: %s", cmp.Diff(tc.obj, tc.emptyObj))
			}
		})
	}
}
