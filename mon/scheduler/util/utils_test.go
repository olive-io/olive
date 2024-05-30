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
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/net"
	clienttesting "k8s.io/client-go/testing"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
)

func TestGetDefinitionFullName(t *testing.T) {
	definition := &corev1.Definition{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test",
			Name:      "definition",
		},
	}
	got := GetDefinitionFullName(definition)
	expected := fmt.Sprintf("%s_%s", definition.Name, definition.Namespace)
	if got != expected {
		t.Errorf("Got wrong full name, got: %s, expected: %s", got, expected)
	}
}

func newPriorityDefinitionWithStartTime(name string, priority int64, startTime time.Time) *corev1.Definition {
	return &corev1.Definition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.DefinitionSpec{
			Priority: &priority,
		},
		Status: corev1.DefinitionStatus{},
	}
}

func TestGetEarliestDefinitionStartTime(t *testing.T) {
	var priority int32 = 1
	currentTime := time.Now()
	tests := []struct {
		name              string
		definitions       []*corev1.Definition
		expectedStartTime *metav1.Time
	}{
		{
			name:              "Definitions length is 0",
			definitions:       []*corev1.Definition{},
			expectedStartTime: nil,
		},
		{
			name: "generate new startTime",
			definitions: []*corev1.Definition{
				newPriorityDefinitionWithStartTime("definition1", 1, currentTime.Add(-time.Second)),
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "definition2",
					},
					Spec: corev1.DefinitionSpec{
						Priority: &priority,
					},
				},
			},
			expectedStartTime: &metav1.Time{Time: currentTime.Add(-time.Second)},
		},
		{
			name: "Definition with earliest start time last in the list",
			definitions: []*corev1.Definition{
				newPriorityDefinitionWithStartTime("definition1", 1, currentTime.Add(time.Second)),
				newPriorityDefinitionWithStartTime("definition2", 2, currentTime.Add(time.Second)),
				newPriorityDefinitionWithStartTime("definition3", 2, currentTime),
			},
			expectedStartTime: &metav1.Time{Time: currentTime},
		},
		{
			name: "Definition with earliest start time first in the list",
			definitions: []*corev1.Definition{
				newPriorityDefinitionWithStartTime("definition1", 2, currentTime),
				newPriorityDefinitionWithStartTime("definition2", 2, currentTime.Add(time.Second)),
				newPriorityDefinitionWithStartTime("definition3", 2, currentTime.Add(2*time.Second)),
			},
			expectedStartTime: &metav1.Time{Time: currentTime},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			startTime := GetEarliestDefinitionStartTime(&extenderv1.Victims{Definitions: test.definitions})
			if !startTime.Equal(test.expectedStartTime) {
				t.Errorf("startTime is not the expected result,got %v, expected %v", startTime, test.expectedStartTime)
			}
		})
	}
}

func TestMoreImportantDefinition(t *testing.T) {
	currentTime := time.Now()
	definition1 := newPriorityDefinitionWithStartTime("definition1", 1, currentTime)
	definition2 := newPriorityDefinitionWithStartTime("definition2", 2, currentTime.Add(time.Second))
	definition3 := newPriorityDefinitionWithStartTime("definition3", 2, currentTime)

	tests := map[string]struct {
		p1       *corev1.Definition
		p2       *corev1.Definition
		expected bool
	}{
		"Definition with higher priority": {
			p1:       definition1,
			p2:       definition2,
			expected: false,
		},
		"Definition with older created time": {
			p1:       definition2,
			p2:       definition3,
			expected: false,
		},
		"Definitions with same start time": {
			p1:       definition3,
			p2:       definition1,
			expected: true,
		},
	}

	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			got := MoreImportantDefinition(v.p1, v.p2)
			if got != v.expected {
				t.Errorf("expected %t but got %t", v.expected, got)
			}
		})
	}
}

func TestRemoveNominatedNodeName(t *testing.T) {
	tests := []struct {
		name                     string
		currentNominatedNodeName string
		newNominatedNodeName     string
		expectedPatchRequests    int
		expectedPatchData        string
	}{
		{
			name:                     "Should make patch request to clear node name",
			currentNominatedNodeName: "node1",
			expectedPatchRequests:    1,
			expectedPatchData:        `{"status":{"nominatedNodeName":null}}`,
		},
		{
			name:                     "Should not make patch request if nominated node is already cleared",
			currentNominatedNodeName: "",
			expectedPatchRequests:    0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualPatchRequests := 0
			var actualPatchData string
			cs := &clientsetfake.Clientset{}
			cs.AddReactor("patch", "definitions", func(action clienttesting.Action) (bool, runtime.Object, error) {
				actualPatchRequests++
				patch := action.(clienttesting.PatchAction)
				actualPatchData = string(patch.GetPatch())
				// For this test, we don't care about the result of the patched definition, just that we got the expected
				// patch request, so just returning &corev1.Definition{} here is OK because scheduler doesn't use the response.
				return true, &corev1.Definition{}, nil
			})

			definition := &corev1.Definition{
				ObjectMeta: metav1.ObjectMeta{Name: "foo"},
				Status:     corev1.DefinitionStatus{NominatedNodeName: test.currentNominatedNodeName},
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := ClearNominatedNodeName(ctx, cs, definition); err != nil {
				t.Fatalf("Error calling removeNominatedNodeName: %v", err)
			}

			if actualPatchRequests != test.expectedPatchRequests {
				t.Fatalf("Actual patch requests (%d) dos not equal expected patch requests (%d)", actualPatchRequests, test.expectedPatchRequests)
			}

			if test.expectedPatchRequests > 0 && actualPatchData != test.expectedPatchData {
				t.Fatalf("Patch data mismatch: Actual was %v, but expected %v", actualPatchData, test.expectedPatchData)
			}
		})
	}
}

func TestPatchDefinitionStatus(t *testing.T) {
	tests := []struct {
		name       string
		definition corev1.Definition
		client     *clientsetfake.Clientset
		// validateErr checks if error returned from PatchDefinitionStatus is expected one or not.
		// (true means error is expected one.)
		validateErr    func(goterr error) bool
		statusToUpdate corev1.DefinitionStatus
	}{
		{
			name:   "Should update definition conditions successfully",
			client: clientsetfake.NewSimpleClientset(),
			definition: corev1.Definition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "definition1",
				},
				Spec: corev1.DefinitionSpec{
					ImagePullSecrets: []v1.LocalObjectReference{{Name: "foo"}},
				},
			},
			statusToUpdate: corev1.DefinitionStatus{
				Conditions: []corev1.DefinitionCondition{
					{
						Type:   corev1.DefinitionScheduled,
						Status: v1.ConditionFalse,
					},
				},
			},
		},
		{
			// ref: #101697, #94626 - ImagePullSecrets are allowed to have empty secret names
			// which would fail the 2-way merge patch generation on Definition patches
			// due to the mergeKey being the name field
			name:   "Should update definition conditions successfully on a definition Spec with secrets with empty name",
			client: clientsetfake.NewSimpleClientset(),
			definition: corev1.Definition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "definition1",
				},
				Spec: corev1.DefinitionSpec{
					// this will serialize to imagePullSecrets:[{}]
					ImagePullSecrets: make([]v1.LocalObjectReference, 1),
				},
			},
			statusToUpdate: corev1.DefinitionStatus{
				Conditions: []corev1.DefinitionCondition{
					{
						Type:   corev1.DefinitionScheduled,
						Status: v1.ConditionFalse,
					},
				},
			},
		},
		{
			name: "retry patch request when an 'connection refused' error is returned",
			client: func() *clientsetfake.Clientset {
				client := clientsetfake.NewSimpleClientset()

				reqcount := 0
				client.PrependReactor("patch", "definitions", func(action clienttesting.Action) (bool, runtime.Object, error) {
					defer func() { reqcount++ }()
					if reqcount == 0 {
						// return an connection refused error for the first patch request.
						return true, &corev1.Definition{}, fmt.Errorf("connection refused: %w", syscall.ECONNREFUSED)
					}
					if reqcount == 1 {
						// not return error for the second patch request.
						return false, &corev1.Definition{}, nil
					}

					// return error if requests comes in more than three times.
					return true, nil, errors.New("requests comes in more than three times.")
				})

				return client
			}(),
			definition: corev1.Definition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "definition1",
				},
				Spec: corev1.DefinitionSpec{
					ImagePullSecrets: []v1.LocalObjectReference{{Name: "foo"}},
				},
			},
			statusToUpdate: corev1.DefinitionStatus{
				Conditions: []corev1.DefinitionCondition{
					{
						Type:   corev1.DefinitionScheduled,
						Status: v1.ConditionFalse,
					},
				},
			},
		},
		{
			name: "only 4 retries at most",
			client: func() *clientsetfake.Clientset {
				client := clientsetfake.NewSimpleClientset()

				reqcount := 0
				client.PrependReactor("patch", "definitions", func(action clienttesting.Action) (bool, runtime.Object, error) {
					defer func() { reqcount++ }()
					if reqcount >= 4 {
						// return error if requests comes in more than four times.
						return true, nil, errors.New("requests comes in more than four times.")
					}

					// return an connection refused error for the first patch request.
					return true, &corev1.Definition{}, fmt.Errorf("connection refused: %w", syscall.ECONNREFUSED)
				})

				return client
			}(),
			definition: corev1.Definition{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
					Name:      "definition1",
				},
				Spec: corev1.DefinitionSpec{
					ImagePullSecrets: []v1.LocalObjectReference{{Name: "foo"}},
				},
			},
			validateErr: net.IsConnectionRefused,
			statusToUpdate: corev1.DefinitionStatus{
				Conditions: []corev1.DefinitionCondition{
					{
						Type:   corev1.DefinitionScheduled,
						Status: v1.ConditionFalse,
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := tc.client
			_, err := client.CoreV1().Definitions(tc.definition.Namespace).Create(context.TODO(), &tc.definition, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			err = PatchDefinitionStatus(ctx, client, &tc.definition, &tc.statusToUpdate)
			if err != nil && tc.validateErr == nil {
				// shouldn't be error
				t.Fatal(err)
			}
			if tc.validateErr != nil {
				if !tc.validateErr(err) {
					t.Fatalf("Returned unexpected error: %v", err)
				}
				return
			}

			retrievedDefinition, err := client.CoreV1().Definitions(tc.definition.Namespace).Get(ctx, tc.definition.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.statusToUpdate, retrievedDefinition.Status); diff != "" {
				t.Errorf("unexpected definition status (-want,+got):\n%s", diff)
			}
		})
	}
}

// Test_As tests the As function with Definition.
func Test_As_Definition(t *testing.T) {
	tests := []struct {
		name       string
		oldObj     interface{}
		newObj     interface{}
		wantOldObj *corev1.Definition
		wantNewObj *corev1.Definition
		wantErr    bool
	}{
		{
			name:       "nil old Definition",
			oldObj:     nil,
			newObj:     &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantOldObj: nil,
			wantNewObj: &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		},
		{
			name:       "nil new Definition",
			oldObj:     &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			newObj:     nil,
			wantOldObj: &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantNewObj: nil,
		},
		{
			name:    "two different kinds of objects",
			oldObj:  &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			newObj:  &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOld, gotNew, err := As[*corev1.Definition](tc.oldObj, tc.newObj)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, but got nil")
				}
				return
			}

			if diff := cmp.Diff(tc.wantOldObj, gotOld); diff != "" {
				t.Errorf("unexpected old object (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantNewObj, gotNew); diff != "" {
				t.Errorf("unexpected new object (-want,+got):\n%s", diff)
			}
		})
	}
}

// Test_As_Node tests the As function with Node.
func Test_As_Node(t *testing.T) {
	tests := []struct {
		name       string
		oldObj     interface{}
		newObj     interface{}
		wantOldObj *v1.Node
		wantNewObj *v1.Node
		wantErr    bool
	}{
		{
			name:       "nil old Node",
			oldObj:     nil,
			newObj:     &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantOldObj: nil,
			wantNewObj: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
		},
		{
			name:       "nil new Node",
			oldObj:     &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			newObj:     nil,
			wantOldObj: &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantNewObj: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotOld, gotNew, err := As[*v1.Node](tc.oldObj, tc.newObj)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, but got nil")
				}
				return
			}

			if diff := cmp.Diff(tc.wantOldObj, gotOld); diff != "" {
				t.Errorf("unexpected old object (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantNewObj, gotNew); diff != "" {
				t.Errorf("unexpected new object (-want,+got):\n%s", diff)
			}
		})
	}
}

// Test_As_KMetadata tests the As function with Definition.
func Test_As_KMetadata(t *testing.T) {
	tests := []struct {
		name    string
		oldObj  interface{}
		newObj  interface{}
		wantErr bool
	}{
		{
			name:    "nil old Definition",
			oldObj:  nil,
			newObj:  &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantErr: false,
		},
		{
			name:    "nil new Definition",
			oldObj:  &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			newObj:  nil,
			wantErr: false,
		},
		{
			name:    "two different kinds of objects",
			oldObj:  &corev1.Definition{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			newObj:  &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "foo"}},
			wantErr: false,
		},
		{
			name:    "unknown old type",
			oldObj:  "unknown type",
			wantErr: true,
		},
		{
			name:    "unknown new type",
			newObj:  "unknown type",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := As[klog.KMetadata](tc.oldObj, tc.newObj)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, but got nil")
				}
				return
			}
		})
	}
}
