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

package cache

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/ktesting"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var allRunners = []*corev1.Runner{
	// Runner 0: a runner without any region-zone label
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-0",
		},
	},
	// Runner 1: a runner with region label only
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-1",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-1",
			},
		},
	},
	// Runner 2: a runner with zone label only
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-2",
			Labels: map[string]string{
				v1.LabelTopologyZone: "zone-2",
			},
		},
	},
	// Runner 3: a runner with proper region and zone labels
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-3",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-1",
				v1.LabelTopologyZone:   "zone-2",
			},
		},
	},
	// Runner 4: a runner with proper region and zone labels
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-4",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-1",
				v1.LabelTopologyZone:   "zone-2",
			},
		},
	},
	// Runner 5: a runner with proper region and zone labels in a different zone, same region as above
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-5",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-1",
				v1.LabelTopologyZone:   "zone-3",
			},
		},
	},
	// Runner 6: a runner with proper region and zone labels in a new region and zone
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-6",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-2",
				v1.LabelTopologyZone:   "zone-2",
			},
		},
	},
	// Runner 7: a runner with proper region and zone labels in a region and zone as runner-6
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-7",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-2",
				v1.LabelTopologyZone:   "zone-2",
			},
		},
	},
	// Runner 8: a runner with proper region and zone labels in a region and zone as runner-6
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-8",
			Labels: map[string]string{
				v1.LabelTopologyRegion: "region-2",
				v1.LabelTopologyZone:   "zone-2",
			},
		},
	},
	// Runner 9: a runner with zone + region label and the deprecated zone + region label
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-9",
			Labels: map[string]string{
				v1.LabelTopologyRegion:          "region-2",
				v1.LabelTopologyZone:            "zone-2",
				v1.LabelFailureDomainBetaRegion: "region-2",
				v1.LabelFailureDomainBetaZone:   "zone-2",
			},
		},
	},
	// Runner 10: a runner with only the deprecated zone + region labels
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "runner-10",
			Labels: map[string]string{
				v1.LabelFailureDomainBetaRegion: "region-2",
				v1.LabelFailureDomainBetaZone:   "zone-3",
			},
		},
	},
}

func verifyRunnerTree(t *testing.T, nt *runnerTree, expectedTree map[string][]string) {
	expectedNumRunners := int(0)
	for _, na := range expectedTree {
		expectedNumRunners += len(na)
	}
	if numRunners := nt.numRunners; numRunners != expectedNumRunners {
		t.Errorf("unexpected runnerTree.numRunners. Expected: %v, Got: %v", expectedNumRunners, numRunners)
	}
	if diff := cmp.Diff(expectedTree, nt.tree); diff != "" {
		t.Errorf("Unexpected runner tree (-want, +got):\n%s", diff)
	}
	if len(nt.zones) != len(expectedTree) {
		t.Errorf("Number of zones in runnerTree.zones is not expected. Expected: %v, Got: %v", len(expectedTree), len(nt.zones))
	}
	for _, z := range nt.zones {
		if _, ok := expectedTree[z]; !ok {
			t.Errorf("zone %v is not expected to exist in runnerTree.zones", z)
		}
	}
}

func TestRunnerTree_AddRunner(t *testing.T) {
	tests := []struct {
		name         string
		runnersToAdd []*corev1.Runner
		expectedTree map[string][]string
	}{
		{
			name:         "single runner no labels",
			runnersToAdd: allRunners[:1],
			expectedTree: map[string][]string{"": {"runner-0"}},
		},
		{
			name:         "same runner specified twice",
			runnersToAdd: []*corev1.Runner{allRunners[0], allRunners[0]},
			expectedTree: map[string][]string{"": {"runner-0"}},
		},
		{
			name:         "mix of runners with and without proper labels",
			runnersToAdd: allRunners[:4],
			expectedTree: map[string][]string{
				"":                     {"runner-0"},
				"region-1:\x00:":       {"runner-1"},
				":\x00:zone-2":         {"runner-2"},
				"region-1:\x00:zone-2": {"runner-3"},
			},
		},
		{
			name:         "mix of runners with and without proper labels and some zones with multiple runners",
			runnersToAdd: allRunners[:7],
			expectedTree: map[string][]string{
				"":                     {"runner-0"},
				"region-1:\x00:":       {"runner-1"},
				":\x00:zone-2":         {"runner-2"},
				"region-1:\x00:zone-2": {"runner-3", "runner-4"},
				"region-1:\x00:zone-3": {"runner-5"},
				"region-2:\x00:zone-2": {"runner-6"},
			},
		},
		{
			name:         "runners also using deprecated zone/region label",
			runnersToAdd: allRunners[9:],
			expectedTree: map[string][]string{
				"region-2:\x00:zone-2": {"runner-9"},
				"region-2:\x00:zone-3": {"runner-10"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			nt := newRunnerTree(logger, nil)
			for _, n := range test.runnersToAdd {
				nt.addRunner(logger, n)
			}
			verifyRunnerTree(t, nt, test.expectedTree)
		})
	}
}

func TestRunnerTree_RemoveRunner(t *testing.T) {
	tests := []struct {
		name            string
		existingRunners []*corev1.Runner
		runnersToRemove []*corev1.Runner
		expectedTree    map[string][]string
		expectError     bool
	}{
		{
			name:            "remove a single runner with no labels",
			existingRunners: allRunners[:7],
			runnersToRemove: allRunners[:1],
			expectedTree: map[string][]string{
				"region-1:\x00:":       {"runner-1"},
				":\x00:zone-2":         {"runner-2"},
				"region-1:\x00:zone-2": {"runner-3", "runner-4"},
				"region-1:\x00:zone-3": {"runner-5"},
				"region-2:\x00:zone-2": {"runner-6"},
			},
		},
		{
			name:            "remove a few runners including one from a zone with multiple runners",
			existingRunners: allRunners[:7],
			runnersToRemove: allRunners[1:4],
			expectedTree: map[string][]string{
				"":                     {"runner-0"},
				"region-1:\x00:zone-2": {"runner-4"},
				"region-1:\x00:zone-3": {"runner-5"},
				"region-2:\x00:zone-2": {"runner-6"},
			},
		},
		{
			name:            "remove all runners",
			existingRunners: allRunners[:7],
			runnersToRemove: allRunners[:7],
			expectedTree:    map[string][]string{},
		},
		{
			name:            "remove non-existing runner",
			existingRunners: nil,
			runnersToRemove: allRunners[:5],
			expectedTree:    map[string][]string{},
			expectError:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			nt := newRunnerTree(logger, test.existingRunners)
			for _, n := range test.runnersToRemove {
				err := nt.removeRunner(logger, n)
				if test.expectError == (err == nil) {
					t.Errorf("unexpected returned error value: %v", err)
				}
			}
			verifyRunnerTree(t, nt, test.expectedTree)
		})
	}
}

func TestRunnerTree_UpdateRunner(t *testing.T) {
	tests := []struct {
		name            string
		existingRunners []*corev1.Runner
		runnerToUpdate  *corev1.Runner
		expectedTree    map[string][]string
	}{
		{
			name:            "update a runner without label",
			existingRunners: allRunners[:7],
			runnerToUpdate: &corev1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-0",
					Labels: map[string]string{
						corev1.LabelTopologyRegion: "region-1",
						corev1.LabelTopologyZone:   "zone-2",
					},
				},
			},
			expectedTree: map[string][]string{
				"region-1:\x00:":       {"runner-1"},
				":\x00:zone-2":         {"runner-2"},
				"region-1:\x00:zone-2": {"runner-3", "runner-4", "runner-0"},
				"region-1:\x00:zone-3": {"runner-5"},
				"region-2:\x00:zone-2": {"runner-6"},
			},
		},
		{
			name:            "update the only existing runner",
			existingRunners: allRunners[:1],
			runnerToUpdate: &corev1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-0",
					Labels: map[string]string{
						corev1.LabelTopologyRegion: "region-1",
						corev1.LabelTopologyZone:   "zone-2",
					},
				},
			},
			expectedTree: map[string][]string{
				"region-1:\x00:zone-2": {"runner-0"},
			},
		},
		{
			name:            "update non-existing runner",
			existingRunners: allRunners[:1],
			runnerToUpdate: &corev1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-new",
					Labels: map[string]string{
						corev1.LabelTopologyRegion: "region-1",
						corev1.LabelTopologyZone:   "zone-2",
					},
				},
			},
			expectedTree: map[string][]string{
				"":                     {"runner-0"},
				"region-1:\x00:zone-2": {"runner-new"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			nt := newRunnerTree(logger, test.existingRunners)
			var oldRunner *corev1.Runner
			for _, n := range allRunners {
				if n.Name == test.runnerToUpdate.Name {
					oldRunner = n
					break
				}
			}
			if oldRunner == nil {
				oldRunner = &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "nonexisting-runner"}}
			}
			nt.updateRunner(logger, oldRunner, test.runnerToUpdate)
			verifyRunnerTree(t, nt, test.expectedTree)
		})
	}
}

func TestRunnerTree_List(t *testing.T) {
	tests := []struct {
		name           string
		runnersToAdd   []*corev1.Runner
		expectedOutput []string
	}{
		{
			name:           "empty tree",
			runnersToAdd:   nil,
			expectedOutput: nil,
		},
		{
			name:           "one runner",
			runnersToAdd:   allRunners[:1],
			expectedOutput: []string{"runner-0"},
		},
		{
			name:           "four runners",
			runnersToAdd:   allRunners[:4],
			expectedOutput: []string{"runner-0", "runner-1", "runner-2", "runner-3"},
		},
		{
			name:           "all runners",
			runnersToAdd:   allRunners[:9],
			expectedOutput: []string{"runner-0", "runner-1", "runner-2", "runner-3", "runner-5", "runner-6", "runner-4", "runner-7", "runner-8"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			nt := newRunnerTree(logger, test.runnersToAdd)

			output, err := nt.list()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.expectedOutput, output); diff != "" {
				t.Errorf("Unexpected output (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestRunnerTree_List_Exhausted(t *testing.T) {
	logger, _ := ktesting.NewTestContext(t)
	nt := newRunnerTree(logger, allRunners[:9])
	nt.numRunners++
	_, err := nt.list()
	if err == nil {
		t.Fatal("Expected an error from zone exhaustion")
	}
}

func TestRunnerTreeMultiOperations(t *testing.T) {
	tests := []struct {
		name            string
		runnersToAdd    []*corev1.Runner
		runnersToRemove []*corev1.Runner
		operations      []string
		expectedOutput  []string
	}{
		{
			name:            "add and remove all runners",
			runnersToAdd:    allRunners[2:9],
			runnersToRemove: allRunners[2:9],
			operations:      []string{"add", "add", "add", "remove", "remove", "remove"},
			expectedOutput:  nil,
		},
		{
			name:            "add and remove some runners",
			runnersToAdd:    allRunners[2:9],
			runnersToRemove: allRunners[2:9],
			operations:      []string{"add", "add", "add", "remove"},
			expectedOutput:  []string{"runner-3", "runner-4"},
		},
		{
			name:            "remove three runners",
			runnersToAdd:    allRunners[2:9],
			runnersToRemove: allRunners[2:9],
			operations:      []string{"add", "add", "add", "remove", "remove", "remove", "add"},
			expectedOutput:  []string{"runner-5"},
		},
		{
			name:            "add more runners to an exhausted zone",
			runnersToAdd:    append(allRunners[4:9:9], allRunners[3]),
			runnersToRemove: nil,
			operations:      []string{"add", "add", "add", "add", "add", "add"},
			expectedOutput:  []string{"runner-4", "runner-5", "runner-6", "runner-3", "runner-7", "runner-8"},
		},
		{
			name:            "remove zone and add new",
			runnersToAdd:    append(allRunners[3:5:5], allRunners[6:8]...),
			runnersToRemove: allRunners[3:5],
			operations:      []string{"add", "add", "remove", "add", "add", "remove"},
			expectedOutput:  []string{"runner-6", "runner-7"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, _ := ktesting.NewTestContext(t)
			nt := newRunnerTree(logger, nil)
			addIndex := 0
			removeIndex := 0
			for _, op := range test.operations {
				switch op {
				case "add":
					if addIndex >= len(test.runnersToAdd) {
						t.Error("more add operations than runnersToAdd")
					} else {
						nt.addRunner(logger, test.runnersToAdd[addIndex])
						addIndex++
					}
				case "remove":
					if removeIndex >= len(test.runnersToRemove) {
						t.Error("more remove operations than runnersToRemove")
					} else {
						nt.removeRunner(logger, test.runnersToRemove[removeIndex])
						removeIndex++
					}
				default:
					t.Errorf("unknown operation: %v", op)
				}
			}
			output, err := nt.list()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(test.expectedOutput, output); diff != "" {
				t.Errorf("Unexpected output (-want, +got):\n%s", diff)
			}
		})
	}
}
