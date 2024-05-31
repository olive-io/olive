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

package runneraffinity

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2/ktesting"

	config "github.com/olive-io/olive/apis/config/v1"
	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/mon/scheduler/framework"
	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
	"github.com/olive-io/olive/mon/scheduler/internal/cache"
	st "github.com/olive-io/olive/mon/scheduler/testing"
	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
)

// TODO: Add test case for RequiredDuringSchedulingRequiredDuringExecution after it's implemented.
func TestRunnerAffinity(t *testing.T) {
	tests := []struct {
		name                string
		region              *corev1.Region
		labels              map[string]string
		runnerName          string
		wantStatus          *framework.Status
		wantPreFilterStatus *framework.Status
		wantPreFilterResult *framework.PreFilterResult
		args                config.RunnerAffinityArgs
		runPreFilter        bool
	}{
		{
			name: "missing labels",
			region: st.MakeRegion().RunnerSelector(map[string]string{
				"foo": "bar",
			}).Obj(),
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "same labels",
			region: st.MakeRegion().RunnerSelector(map[string]string{
				"foo": "bar",
			}).Obj(),
			labels: map[string]string{
				"foo": "bar",
			},
			runPreFilter: true,
		},
		{
			name: "runner labels are superset",
			region: st.MakeRegion().RunnerSelector(map[string]string{
				"foo": "bar",
			}).Obj(),
			labels: map[string]string{
				"foo": "bar",
				"baz": "blah",
			},
			runPreFilter: true,
		},
		{
			name: "runner labels are subset",
			region: st.MakeRegion().RunnerSelector(map[string]string{
				"foo": "bar",
				"baz": "blah",
			}).Obj(),
			labels: map[string]string{
				"foo": "bar",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with matchExpressions using In operator that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar", "value2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			runPreFilter: true,
		},
		{
			name: "Region with matchExpressions using Gt operator that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "kernel-version",
												Operator: corev1.RunnerSelectorOpGt,
												Values:   []string{"0204"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				// We use two digit to denote major version and two digit for minor version.
				"kernel-version": "0206",
			},
			runPreFilter: true,
		},
		{
			name: "Region with matchExpressions using NotIn operator that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "mem-type",
												Operator: corev1.RunnerSelectorOpNotIn,
												Values:   []string{"DDR", "DDR2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"mem-type": "DDR3",
			},
			runPreFilter: true,
		},
		{
			name: "Region with matchExpressions using Exists operator that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpExists,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"GPU": "NVIDIA-GRID-K1",
			},
			runPreFilter: true,
		},
		{
			name: "Region with affinity that don't match runner's labels won't schedule onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"value1", "value2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with empty MatchExpressions is not a valid value will match no objects and won't schedule onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name:   "Region with no Affinity will schedule onto a runner",
			region: &corev1.Region{},
			labels: map[string]string{
				"foo": "bar",
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
			runPreFilter:        true,
		},
		{
			name: "Region with Affinity but nil RunnerSelector will schedule onto a runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: nil,
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			wantPreFilterStatus: framework.NewStatus(framework.Skip),
			runPreFilter:        true,
		},
		{
			name: "Region with multiple matchExpressions ANDed that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpExists,
											}, {
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpNotIn,
												Values:   []string{"AMD", "INTER"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"GPU": "NVIDIA-GRID-K1",
			},
			runPreFilter: true,
		},
		{
			name: "Region with multiple matchExpressions ANDed that doesn't match the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpExists,
											}, {
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"AMD", "INTER"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"GPU": "NVIDIA-GRID-K1",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with multiple RunnerSelectorTerms ORed in affinity, matches the runner's labels and will schedule onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar", "value2"},
											},
										},
									},
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "diffkey",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"wrong", "value2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			runPreFilter: true,
		},
		{
			name: "Region with an Affinity and a RegionSpec.RunnerSelector(the old thing that we are deprecating) " +
				"both are satisfied, will schedule onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
					},
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpExists,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			runPreFilter: true,
		},
		{
			name: "Region with an Affinity matches runner's labels but the RegionSpec.RunnerSelector(the old thing that we are deprecating) " +
				"is not satisfied, won't schedule onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
					},
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpExists,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "barrrrrr",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with an invalid value in Affinity term won't be scheduled onto the runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpNotIn,
												Values:   []string{"invalid value: ___@#$%^"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with matchFields using In operator that matches the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner1",
			wantPreFilterResult: &framework.PreFilterResult{RunnerNames: sets.New("runner1")},
			runPreFilter:        true,
		},
		{
			name: "Region with matchFields using In operator that does not match the existing runner",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner2",
			wantStatus:          framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			wantPreFilterResult: &framework.PreFilterResult{RunnerNames: sets.New("runner1")},
			runPreFilter:        true,
		},
		{
			name: "Region with two terms: matchFields does not match, but matchExpressions matches",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner2"},
											},
										},
									},
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:   "runner2",
			labels:       map[string]string{"foo": "bar"},
			runPreFilter: true,
		},
		{
			name: "Region with one term: matchFields does not match, but matchExpressions matches",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner2",
			labels:              map[string]string{"foo": "bar"},
			wantPreFilterResult: &framework.PreFilterResult{RunnerNames: sets.New("runner1")},
			wantStatus:          framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter:        true,
		},
		{
			name: "Region with one term: both matchFields and matchExpressions match",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner1",
			labels:              map[string]string{"foo": "bar"},
			wantPreFilterResult: &framework.PreFilterResult{RunnerNames: sets.New("runner1")},
			runPreFilter:        true,
		},
		{
			name: "Region with two terms: both matchFields and matchExpressions do not match",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
									},
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "foo",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"not-match-to-bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:   "runner2",
			labels:       map[string]string{"foo": "bar"},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name: "Region with two terms of runner.Name affinity",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
										},
									},
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner2",
			wantPreFilterResult: &framework.PreFilterResult{RunnerNames: sets.New("runner1", "runner2")},
			runPreFilter:        true,
		},
		{
			name: "Region with two conflicting mach field requirements",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchFields: []corev1.RunnerSelectorRequirement{
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner1"},
											},
											{
												Key:      metav1.ObjectNameField,
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"runner2"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName:          "runner2",
			labels:              map[string]string{"foo": "bar"},
			wantPreFilterStatus: framework.NewStatus(framework.UnschedulableAndUnresolvable, errReasonConflict),
			wantStatus:          framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter:        true,
		},
		{
			name: "Matches added affinity and Region's runner affinity",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "zone",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"foo"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName: "runner2",
			labels:     map[string]string{"zone": "foo"},
			args: config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
						RunnerSelectorTerms: []corev1.RunnerSelectorTerm{{
							MatchFields: []corev1.RunnerSelectorRequirement{{
								Key:      metav1.ObjectNameField,
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"runner2"},
							}},
						}},
					},
				},
			},
			runPreFilter: true,
		},
		{
			name: "Matches added affinity but not Region's runner affinity",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "zone",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runnerName: "runner2",
			labels:     map[string]string{"zone": "foo"},
			args: config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
						RunnerSelectorTerms: []corev1.RunnerSelectorTerm{{
							MatchFields: []corev1.RunnerSelectorRequirement{{
								Key:      metav1.ObjectNameField,
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"runner2"},
							}},
						}},
					},
				},
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, ErrReasonRegion),
			runPreFilter: true,
		},
		{
			name:       "Doesn't match added affinity",
			region:     &corev1.Region{},
			runnerName: "runner2",
			labels:     map[string]string{"zone": "foo"},
			args: config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
						RunnerSelectorTerms: []corev1.RunnerSelectorTerm{{
							MatchExpressions: []corev1.RunnerSelectorRequirement{
								{
									Key:      "zone",
									Operator: corev1.RunnerSelectorOpIn,
									Values:   []string{"bar"},
								},
							},
						}},
					},
				},
			},
			wantStatus:   framework.NewStatus(framework.UnschedulableAndUnresolvable, errReasonEnforced),
			runPreFilter: true,
		},
		{
			name: "Matches runner selector correctly even if PreFilter is not called",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
				"baz": "blah",
			},
			runPreFilter: false,
		},
		{
			name: "Matches runner affinity correctly even if PreFilter is not called",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
									{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpExists,
											}, {
												Key:      "GPU",
												Operator: corev1.RunnerSelectorOpNotIn,
												Values:   []string{"AMD", "INTER"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"GPU": "NVIDIA-GRID-K1",
			},
			runPreFilter: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			runner := corev1.Runner{ObjectMeta: metav1.ObjectMeta{
				Name:   test.runnerName,
				Labels: test.labels,
			}}
			runnerInfo := framework.NewRunnerInfo()
			runnerInfo.SetRunner(&runner)

			p, err := New(ctx, &test.args, nil)
			if err != nil {
				t.Fatalf("Creating plugin: %v", err)
			}

			state := framework.NewCycleState()
			var gotStatus *framework.Status
			if test.runPreFilter {
				gotPreFilterResult, gotStatus := p.(framework.PreFilterPlugin).PreFilter(context.Background(), state, test.region)
				if diff := cmp.Diff(test.wantPreFilterStatus, gotStatus); diff != "" {
					t.Errorf("unexpected PreFilter Status (-want,+got):\n%s", diff)
				}
				if diff := cmp.Diff(test.wantPreFilterResult, gotPreFilterResult); diff != "" {
					t.Errorf("unexpected PreFilterResult (-want,+got):\n%s", diff)
				}
			}
			gotStatus = p.(framework.FilterPlugin).Filter(context.Background(), state, test.region, runnerInfo)
			if diff := cmp.Diff(test.wantStatus, gotStatus); diff != "" {
				t.Errorf("unexpected Filter Status (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestRunnerAffinityPriority(t *testing.T) {
	label1 := map[string]string{"foo": "bar"}
	label2 := map[string]string{"key": "value"}
	label3 := map[string]string{"az": "az1"}
	label4 := map[string]string{"abc": "az11", "def": "az22"}
	label5 := map[string]string{"foo": "bar", "key": "value", "az": "az1"}

	affinity1 := &corev1.Affinity{
		RunnerAffinity: &corev1.RunnerAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{{
				Weight: 2,
				Preference: corev1.RunnerSelectorTerm{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "foo",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"bar"},
					}},
				},
			}},
		},
	}

	affinity2 := &corev1.Affinity{
		RunnerAffinity: &corev1.RunnerAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 2,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{
							{
								Key:      "foo",
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"bar"},
							},
						},
					},
				},
				{
					Weight: 4,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{
							{
								Key:      "key",
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"value"},
							},
						},
					},
				},
				{
					Weight: 5,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{
							{
								Key:      "foo",
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"bar"},
							},
							{
								Key:      "key",
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"value"},
							},
							{
								Key:      "az",
								Operator: corev1.RunnerSelectorOpIn,
								Values:   []string{"az1"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name               string
		region             *corev1.Region
		runners            []*corev1.Runner
		expectedList       framework.RunnerScoreList
		args               config.RunnerAffinityArgs
		runPreScore        bool
		wantPreScoreStatus *framework.Status
	}{
		{
			name: "all runners are same priority as RunnerAffinity is nil",
			region: &corev1.Region{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner3", Labels: label3}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 0}, {Name: "runner2", Score: 0}, {Name: "runner3", Score: 0}},
		},
		{
			// PreScore returns Skip.
			name: "Skip is returned in PreScore when RunnerAffinity is nil",
			region: &corev1.Region{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
			},
			runPreScore:        true,
			wantPreScoreStatus: framework.NewStatus(framework.Skip),
		},
		{
			name: "PreScore returns error when an incoming Region has a broken affinity",
			region: &corev1.Region{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
								{
									Weight: 2,
									Preference: corev1.RunnerSelectorTerm{
										MatchExpressions: []corev1.RunnerSelectorRequirement{
											{
												Key:      "invalid key",
												Operator: corev1.RunnerSelectorOpIn,
												Values:   []string{"bar"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
			},
			runPreScore:        true,
			wantPreScoreStatus: framework.AsStatus(fmt.Errorf(`[0].matchExpressions[0].key: Invalid value: "invalid key": name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`)),
		},
		{
			name: "no runner matches preferred scheduling requirements in RunnerAffinity of region so all runners' priority is zero",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: affinity1,
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label4}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner3", Labels: label3}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 0}, {Name: "runner2", Score: 0}, {Name: "runner3", Score: 0}},
			runPreScore:  true,
		},
		{
			name: "only runner1 matches the preferred scheduling requirements of region",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: affinity1,
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner3", Labels: label3}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: 0}, {Name: "runner3", Score: 0}},
			runPreScore:  true,
		},
		{
			name: "all runners matches the preferred scheduling requirements of region but with different priorities ",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: affinity2,
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner5", Labels: label5}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 18}, {Name: "runner5", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: 36}},
			runPreScore:  true,
		},
		{
			name:   "added affinity",
			region: &corev1.Region{},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: 0}},
			args: config.RunnerAffinityArgs{
				AddedAffinity: affinity1.RunnerAffinity,
			},
			runPreScore: true,
		},
		{
			name: "added affinity and region has default affinity",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: affinity1,
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner3", Labels: label5}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 40}, {Name: "runner2", Score: 60}, {Name: "runner3", Score: framework.MaxRunnerScore}},
			args: config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Weight: 3,
							Preference: corev1.RunnerSelectorTerm{
								MatchExpressions: []corev1.RunnerSelectorRequirement{
									{
										Key:      "key",
										Operator: corev1.RunnerSelectorOpIn,
										Values:   []string{"value"},
									},
								},
							},
						},
					},
				},
			},
			runPreScore: true,
		},
		{
			name: "calculate the priorities correctly even if PreScore is not called",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: affinity2,
				},
			},
			runners: []*corev1.Runner{
				{ObjectMeta: metav1.ObjectMeta{Name: "runner1", Labels: label1}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner5", Labels: label5}},
				{ObjectMeta: metav1.ObjectMeta{Name: "runner2", Labels: label2}},
			},
			expectedList: []framework.RunnerScore{{Name: "runner1", Score: 18}, {Name: "runner5", Score: framework.MaxRunnerScore}, {Name: "runner2", Score: 36}},
			runPreScore:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, ctx := ktesting.NewTestContext(t)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			state := framework.NewCycleState()
			fh, _ := runtime.NewFramework(ctx, nil, nil, runtime.WithSnapshotSharedLister(cache.NewSnapshot(nil, test.runners)))
			p, err := New(ctx, &test.args, fh)
			if err != nil {
				t.Fatalf("Creating plugin: %v", err)
			}
			var status *framework.Status
			if test.runPreScore {
				status = p.(framework.PreScorePlugin).PreScore(ctx, state, test.region, tf.BuildRunnerInfos(test.runners))
				if status.Code() != test.wantPreScoreStatus.Code() {
					t.Errorf("unexpected status code from PreScore: want: %v got: %v", test.wantPreScoreStatus.Code().String(), status.Code().String())
				}
				if status.Message() != test.wantPreScoreStatus.Message() {
					t.Errorf("unexpected status message from PreScore: want: %v got: %v", test.wantPreScoreStatus.Message(), status.Message())
				}
				if !status.IsSuccess() {
					// no need to proceed.
					return
				}
			}
			var gotList framework.RunnerScoreList
			for _, n := range test.runners {
				runnerName := n.ObjectMeta.Name
				score, status := p.(framework.ScorePlugin).Score(ctx, state, test.region, runnerName)
				if !status.IsSuccess() {
					t.Errorf("unexpected error: %v", status)
				}
				gotList = append(gotList, framework.RunnerScore{Name: runnerName, Score: score})
			}

			status = p.(framework.ScorePlugin).ScoreExtensions().NormalizeScore(ctx, state, test.region, gotList)
			if !status.IsSuccess() {
				t.Errorf("unexpected error: %v", status)
			}

			if diff := cmp.Diff(test.expectedList, gotList); diff != "" {
				t.Errorf("obtained scores (-want,+got):\n%s", diff)
			}
		})
	}
}

func Test_isSchedulableAfterRunnerChange(t *testing.T) {
	regionWithRunnerAffinity := st.MakeRegion().RunnerAffinityIn("foo", []string{"bar"})
	testcases := map[string]struct {
		args           *config.RunnerAffinityArgs
		region         *corev1.Region
		oldObj, newObj interface{}
		expectedHint   framework.QueueingHint
		expectedErr    bool
	}{
		"backoff-wrong-new-object": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			newObj:       "not-a-runner",
			expectedHint: framework.Queue,
			expectedErr:  true,
		},
		"backoff-wrong-old-object": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			oldObj:       "not-a-runner",
			newObj:       st.MakeRunner().Obj(),
			expectedHint: framework.Queue,
			expectedErr:  true,
		},
		"skip-queue-on-add": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			newObj:       st.MakeRunner().Obj(),
			expectedHint: framework.QueueSkip,
		},
		"queue-on-add": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			newObj:       st.MakeRunner().Label("foo", "bar").Obj(),
			expectedHint: framework.Queue,
		},
		"skip-unrelated-changes": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			oldObj:       st.MakeRunner().Obj(),
			newObj:       st.MakeRunner().Capacity(nil).Obj(),
			expectedHint: framework.QueueSkip,
		},
		"skip-unrelated-changes-on-labels": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.DeepCopy(),
			oldObj:       st.MakeRunner().Obj(),
			newObj:       st.MakeRunner().Label("k", "v").Obj(),
			expectedHint: framework.QueueSkip,
		},
		"skip-labels-changes-on-runner-from-suitable-to-unsuitable": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.DeepCopy(),
			oldObj:       st.MakeRunner().Label("foo", "bar").Obj(),
			newObj:       st.MakeRunner().Label("k", "v").Obj(),
			expectedHint: framework.QueueSkip,
		},
		"queue-on-labels-change-makes-region-schedulable": {
			args:         &config.RunnerAffinityArgs{},
			region:       regionWithRunnerAffinity.Obj(),
			oldObj:       st.MakeRunner().Obj(),
			newObj:       st.MakeRunner().Label("foo", "bar").Obj(),
			expectedHint: framework.Queue,
		},
		"skip-queue-on-add-scheduler-enforced-runner-affinity": {
			args: &config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
						RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
							{
								MatchExpressions: []corev1.RunnerSelectorRequirement{
									{
										Key:      "foo",
										Operator: corev1.RunnerSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			},
			region:       regionWithRunnerAffinity.Obj(),
			newObj:       st.MakeRunner().Obj(),
			expectedHint: framework.QueueSkip,
		},
		"queue-on-add-scheduler-enforced-runner-affinity": {
			args: &config.RunnerAffinityArgs{
				AddedAffinity: &corev1.RunnerAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
						RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
							{
								MatchExpressions: []corev1.RunnerSelectorRequirement{
									{
										Key:      "foo",
										Operator: corev1.RunnerSelectorOpIn,
										Values:   []string{"bar"},
									},
								},
							},
						},
					},
				},
			},
			region:       regionWithRunnerAffinity.Obj(),
			newObj:       st.MakeRunner().Label("foo", "bar").Obj(),
			expectedHint: framework.Queue,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			logger, ctx := ktesting.NewTestContext(t)
			p, err := New(ctx, tc.args, nil)
			if err != nil {
				t.Fatalf("Creating plugin: %v", err)
			}

			actualHint, err := p.(*RunnerAffinity).isSchedulableAfterRunnerChange(logger, tc.region, tc.oldObj, tc.newObj)
			if tc.expectedErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expectedHint, actualHint)
		})
	}
}
