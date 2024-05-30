/*
Copyright 2020 The Kubernetes Authors.

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
	monv1 "github.com/olive-io/olive/apis/mon/v1"
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
				RunnerNameToMetaVictims: map[string]*MetaVictims{"foo": {Definitions: []*MetaDefinition{{UID: "myuid"}}, NumPDBViolations: 1}},
			},
			expectJSON: `{"RunnerNameToMetaVictims":{"foo":{"Definitions":[{"UID":"myuid"}],"NumPDBViolations":1}}}`,
		},
		{
			emptyObj: &ExtenderPreemptionArgs{},
			obj: &ExtenderPreemptionArgs{
				Definition:              &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "podname"}},
				RunnerNameToVictims:     map[string]*Victims{"foo": {Definitions: []*corev1.Definition{{ObjectMeta: metav1.ObjectMeta{Name: "podname"}}}, NumPDBViolations: 1}},
				RunnerNameToMetaVictims: map[string]*MetaVictims{"foo": {Definitions: []*MetaDefinition{{UID: "myuid"}}, NumPDBViolations: 1}},
			},
			expectJSON: `{"Definition":{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}},"RunnerNameToVictims":{"foo":{"Definitions":[{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}}],"NumPDBViolations":1}},"RunnerNameToMetaVictims":{"foo":{"Definitions":[{"UID":"myuid"}],"NumPDBViolations":1}}}`,
		},
		{
			emptyObj: &ExtenderArgs{},
			obj: &ExtenderArgs{
				Definition:  &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "podname"}},
				Runners:     &monv1.RunnerList{Items: []monv1.Runner{{ObjectMeta: metav1.ObjectMeta{Name: "nodename"}}}},
				RunnerNames: &[]string{"node1"},
			},
			expectJSON: `{"Definition":{"metadata":{"name":"podname","creationTimestamp":null},"spec":{"containers":null},"status":{}},"Runners":{"metadata":{},"items":[{"metadata":{"name":"nodename","creationTimestamp":null},"spec":{},"status":{"daemonEndpoints":{"kubeletEndpoint":{"Port":0}},"nodeInfo":{"machineID":"","systemUUID":"","bootID":"","kernelVersion":"","osImage":"","containerRuntimeVersion":"","kubeletVersion":"","kubeProxyVersion":"","operatingSystem":"","architecture":""}}}]},"RunnerNames":["node1"]}`,
		},
		{
			emptyObj: &ExtenderFilterResult{},
			obj: &ExtenderFilterResult{
				Runners:                      &monv1.RunnerList{Items: []monv1.Runner{{ObjectMeta: metav1.ObjectMeta{Name: "nodename"}}}},
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
				DefinitionName:      "mypodname",
				DefinitionNamespace: "mypodnamespace",
				DefinitionUID:       types.UID("mypoduid"),
				Runner:              "mynode",
			},
			expectJSON: `{"DefinitionName":"mypodname","DefinitionNamespace":"mypodnamespace","DefinitionUID":"mypoduid","Runner":"mynode"}`,
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
