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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

var (
	ignoreBadValue = cmpopts.IgnoreFields(field.Error{}, "BadValue")
)

func TestRunnerSelectorMatch(t *testing.T) {
	tests := []struct {
		name         string
		nodeSelector corev1.RunnerSelector
		node         *corev1.Runner
		wantErr      error
		wantMatch    bool
	}{
		{
			name:      "nil node",
			wantMatch: false,
		},
		{
			name: "invalid field selector and label selector",
			nodeSelector: corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1", "host_2"},
					}},
				},
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "invalid key",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_value"},
					}},
				},
			}},
			wantErr: field.ErrorList{
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "nodeSelectorTerms[0].matchFields[0].values",
					Detail: "must have one element",
				},
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "nodeSelectorTerms[2].matchExpressions[0].key",
					Detail: `name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
				},
			}.ToAggregate(),
		},
		{
			name: "node matches field selector, but not labels",
			nodeSelector: corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
		},
		{
			name: "node matches field selector and label selector",
			nodeSelector: corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node:      &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1", Labels: map[string]string{"label_1": "label_1_val"}}},
			wantMatch: true,
		},
		{
			name: "second term matches",
			nodeSelector: corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val"},
					}},
				},
				{
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1"},
					}},
				},
			}},
			node:      &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			wantMatch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeSelector, err := NewRunnerSelector(&tt.nodeSelector)
			if diff := cmp.Diff(tt.wantErr, err, ignoreBadValue); diff != "" {
				t.Errorf("NewRunnerSelector returned unexpected error (-want,+got):\n%s", diff)
			}
			if tt.wantErr != nil {
				return
			}
			match := nodeSelector.Match(tt.node)
			if match != tt.wantMatch {
				t.Errorf("RunnerSelector.Match returned %t, want %t", match, tt.wantMatch)
			}
		})
	}
}

func TestPreferredSchedulingTermsScore(t *testing.T) {
	tests := []struct {
		name           string
		prefSchedTerms []corev1.PreferredSchedulingTerm
		node           *corev1.Runner
		wantErr        error
		wantScore      int64
	}{
		{
			name: "invalid field selector and label selector",
			prefSchedTerms: []corev1.PreferredSchedulingTerm{
				{
					Weight: 1,
					Preference: corev1.RunnerSelectorTerm{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				},
				{
					Weight: 1,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val"},
						}},
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				},
				{
					Weight: 1,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "invalid key",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_value"},
						}},
					},
				},
			},
			wantErr: field.ErrorList{
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "[0].matchFields[0].values",
					Detail: "must have one element",
				},
				&field.Error{
					Type:   field.ErrorTypeInvalid,
					Field:  "[2].matchExpressions[0].key",
					Detail: `name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]')`,
				},
			}.ToAggregate(),
		},
		{
			name: "invalid field selector but no weight, error not reported",
			prefSchedTerms: []corev1.PreferredSchedulingTerm{
				{
					Weight: 0,
					Preference: corev1.RunnerSelectorTerm{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				},
			},
			node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
		},
		{
			name: "first and third term match",
			prefSchedTerms: []corev1.PreferredSchedulingTerm{
				{
					Weight: 5,
					Preference: corev1.RunnerSelectorTerm{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				},
				{
					Weight: 7,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "unknown_label",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"unknown_label_val"},
						}},
					},
				},
				{
					Weight: 11,
					Preference: corev1.RunnerSelectorTerm{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val"},
						}},
					},
				},
			},
			node:      &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1", Labels: map[string]string{"label_1": "label_1_val"}}},
			wantScore: 16,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefSchedTerms, err := NewPreferredSchedulingTerms(tt.prefSchedTerms)
			if diff := cmp.Diff(tt.wantErr, err, ignoreBadValue); diff != "" {
				t.Errorf("NewPreferredSchedulingTerms returned unexpected error (-want,+got):\n%s", diff)
			}
			if tt.wantErr != nil {
				return
			}
			score := prefSchedTerms.Score(tt.node)
			if score != tt.wantScore {
				t.Errorf("PreferredSchedulingTerms.Score returned %d, want %d", score, tt.wantScore)
			}
		})
	}
}

func TestRunnerSelectorRequirementsAsSelector(t *testing.T) {
	matchExpressions := []corev1.RunnerSelectorRequirement{{
		Key:      "foo",
		Operator: corev1.RunnerSelectorOpIn,
		Values:   []string{"bar", "baz"},
	}}
	mustParse := func(s string) labels.Selector {
		out, e := labels.Parse(s)
		if e != nil {
			panic(e)
		}
		return out
	}
	tc := []struct {
		in      []corev1.RunnerSelectorRequirement
		out     labels.Selector
		wantErr []error
	}{
		{in: nil, out: labels.Nothing()},
		{in: []corev1.RunnerSelectorRequirement{}, out: labels.Nothing()},
		{
			in:  matchExpressions,
			out: mustParse("foo in (baz,bar)"),
		},
		{
			in: []corev1.RunnerSelectorRequirement{{
				Key:      "foo",
				Operator: corev1.RunnerSelectorOpExists,
				Values:   []string{"bar", "baz"},
			}},
			wantErr: []error{
				field.ErrorList{
					field.Invalid(field.NewPath("root").Index(0).Child("values"), nil, "values set must be empty for exists and does not exist"),
				}.ToAggregate(),
			},
		},
		{
			in: []corev1.RunnerSelectorRequirement{{
				Key:      "foo",
				Operator: corev1.RunnerSelectorOpGt,
				Values:   []string{"1"},
			}},
			out: mustParse("foo>1"),
		},
		{
			in: []corev1.RunnerSelectorRequirement{{
				Key:      "bar",
				Operator: corev1.RunnerSelectorOpLt,
				Values:   []string{"7"},
			}},
			out: mustParse("bar<7"),
		},
		{
			in: []corev1.RunnerSelectorRequirement{{
				Key:      "baz",
				Operator: "invalid",
				Values:   []string{"5"},
			}},
			wantErr: []error{
				field.NotSupported(field.NewPath("root").Index(0).Child("operator"), corev1.RunnerSelectorOperator("invalid"), validSelectorOperators),
			},
		},
	}

	for i, tc := range tc {
		out, err := nodeSelectorRequirementsAsSelector(tc.in, field.NewPath("root"))
		if diff := cmp.Diff(tc.wantErr, err, ignoreBadValue); diff != "" {
			t.Errorf("nodeSelectorRequirementsAsSelector returned unexpected error (-want,+got):\n%s", diff)
		}
		if !reflect.DeepEqual(out, tc.out) {
			t.Errorf("[%v]expected:\n\t%+v\nbut got:\n\t%+v", i, tc.out, out)
		}
	}
}

func TestRegionMatchesRunnerSelectorAndAffinityTerms(t *testing.T) {
	tests := []struct {
		name     string
		region   *corev1.Region
		labels   map[string]string
		nodeName string
		want     bool
	}{
		{
			name:   "no selector",
			region: &corev1.Region{},
			want:   true,
		},
		{
			name: "missing labels",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
					},
				},
			},
			want: false,
		},
		{
			name: "same labels",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			want: true,
		},
		{
			name: "node labels are superset",
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
			want: true,
		},
		{
			name: "node labels are subset",
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					RunnerSelector: map[string]string{
						"foo": "bar",
						"baz": "blah",
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			want: false,
		},
		{
			name: "Region with matchExpressions using In operator that matches the existing node",
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
			want: true,
		},
		{
			name: "Region with matchExpressions using Gt operator that matches the existing node",
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
			want: true,
		},
		{
			name: "Region with matchExpressions using NotIn operator that matches the existing node",
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
			want: true,
		},
		{
			name: "Region with matchExpressions using Exists operator that matches the existing node",
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
			want: true,
		},
		{
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
			want: false,
			name: "Region with affinity that don't match node's labels won't schedule onto the node",
		},
		{
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: nil,
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			want: false,
			name: "Region with a nil []RunnerSelectorTerm in affinity, can't match the node's labels and won't schedule onto the node",
		},
		{
			region: &corev1.Region{
				Spec: corev1.RegionSpec{
					Affinity: &corev1.Affinity{
						RunnerAffinity: &corev1.RunnerAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.RunnerSelector{
								RunnerSelectorTerms: []corev1.RunnerSelectorTerm{},
							},
						},
					},
				},
			},
			labels: map[string]string{
				"foo": "bar",
			},
			want: false,
			name: "Region with an empty []RunnerSelectorTerm in affinity, can't match the node's labels and won't schedule onto the node",
		},
		{
			name: "Region with empty MatchExpressions is not a valid value will match no objects and won't schedule onto the node",
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
			want: false,
		},
		{
			name:   "Region with no Affinity will schedule onto a node",
			region: &corev1.Region{},
			labels: map[string]string{
				"foo": "bar",
			},
			want: true,
		},
		{
			name: "Region with Affinity but nil RunnerSelector will schedule onto a node",
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
			want: true,
		},
		{
			name: "Region with multiple matchExpressions ANDed that matches the existing node",
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
			want: true,
		},
		{
			name: "Region with multiple matchExpressions ANDed that doesn't match the existing node",
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
			want: false,
		},
		{
			name: "Region with multiple RunnerSelectorTerms ORed in affinity, matches the node's labels and will schedule onto the node",
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
			want: true,
		},
		{
			name: "Region with an Affinity and a RegionSpec.RunnerSelector(the old thing that we are deprecating) " +
				"both are satisfied, will schedule onto the node",
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
			want: true,
		},
		{
			name: "Region with an Affinity matches node's labels but the RegionSpec.RunnerSelector(the old thing that we are deprecating) " +
				"is not satisfied, won't schedule onto the node",
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
			want: false,
		},
		{
			name: "Region with an invalid value in Affinity term won't be scheduled onto the node",
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
			want: false,
		},
		{
			name: "Region with matchFields using In operator that matches the existing node",
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
												Values:   []string{"node_1"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			nodeName: "node_1",
			want:     true,
		},
		{
			name: "Region with matchFields using In operator that does not match the existing node",
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
												Values:   []string{"node_1"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			nodeName: "node_2",
			want:     false,
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
												Values:   []string{"node_1"},
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
			nodeName: "node_2",
			labels:   map[string]string{"foo": "bar"},
			want:     true,
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
												Values:   []string{"node_1"},
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
			nodeName: "node_2",
			labels:   map[string]string{"foo": "bar"},
			want:     false,
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
												Values:   []string{"node_1"},
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
			nodeName: "node_1",
			labels:   map[string]string{"foo": "bar"},
			want:     true,
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
												Values:   []string{"node_1"},
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
			nodeName: "node_2",
			labels:   map[string]string{"foo": "bar"},
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := corev1.Runner{ObjectMeta: metav1.ObjectMeta{
				Name:   test.nodeName,
				Labels: test.labels,
			}}
			got, _ := GetRequiredRunnerAffinity(test.region).Match(&node)
			if test.want != got {
				t.Errorf("expected: %v got %v", test.want, got)
			}
		})
	}
}
