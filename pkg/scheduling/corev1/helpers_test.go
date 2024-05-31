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

package corev1

import (
	"reflect"
	"testing"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "github.com/olive-io/olive/apis/core/v1"
)

// TestRegionPriority tests RegionPriority function.
func TestRegionPriority(t *testing.T) {
	p := int32(20)
	tests := []struct {
		name             string
		pod              *corev1.Region
		expectedPriority int32
	}{
		{
			name: "no priority pod resolves to static default priority",
			pod: &corev1.Region{
				Spec: corev1.RegionSpec{},
			},
			expectedPriority: 0,
		},
		{
			name: "pod with priority resolves correctly",
			pod: &corev1.Region{
				Spec: corev1.RegionSpec{
					Priority: &p,
				},
			},
			expectedPriority: p,
		},
	}
	for _, test := range tests {
		if RegionPriority(test.pod) != test.expectedPriority {
			t.Errorf("expected pod priority: %v, got %v", test.expectedPriority, RegionPriority(test.pod))
		}

	}
}

func TestMatchRunnerSelectorTerms(t *testing.T) {
	type args struct {
		nodeSelector *corev1.RunnerSelector
		node         *corev1.Runner
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil terms",
			args: args{
				nodeSelector: nil,
				node:         nil,
			},
			want: false,
		},
		{
			name: "node label matches matchExpressions terms",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"label_1": "label_1_val"}}},
			},
			want: true,
		},
		{
			name: "node field matches matchFields terms",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			},
			want: true,
		},
		{
			name: "invalid node field requirement",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			},
			want: false,
		},
		{
			name: "fieldSelectorTerm with node labels",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "not_host_1", Labels: map[string]string{
					"metadata.name": "host_1",
				}}},
			},
			want: false,
		},
		{
			name: "labelSelectorTerm with node fields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			},
			want: false,
		},
		{
			name: "labelSelectorTerm and fieldSelectorTerm was set, but only node fields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
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
			want: false,
		},
		{
			name: "labelSelectorTerm and fieldSelectorTerm was set, both node fields and labels (both matched)",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
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
					}},
				},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val",
					}}},
			},
			want: true,
		},
		{
			name: "labelSelectorTerm and fieldSelectorTerm was set, both node fields and labels (one mismatched)",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
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
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val-failed",
					}}},
			},
			want: false,
		},
		{
			name: "multi-selector was set, both node fields and labels (one mismatched)",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
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
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val-failed",
					}}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := MatchRunnerSelectorTerms(tt.args.node, tt.args.nodeSelector); got != tt.want {
				t.Errorf("MatchRunnerSelectorTermsORed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMatchRunnerSelectorTermsStateless ensures MatchRunnerSelectorTerms()
// is invoked in a "stateless" manner, i.e. nodeSelector should NOT
// be deeply modified after invoking
func TestMatchRunnerSelectorTermsStateless(t *testing.T) {
	type args struct {
		nodeSelector *corev1.RunnerSelector
		node         *corev1.Runner
	}

	tests := []struct {
		name string
		args args
		want *corev1.RunnerSelector
	}{
		{
			name: "nil terms",
			args: args{
				nodeSelector: nil,
				node:         nil,
			},
			want: nil,
		},
		{
			name: "nodeLabels: preordered matchExpressions and nil matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val", "label_2_val"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"label_1": "label_1_val"}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val", "label_2_val"},
					}},
				},
			}},
		},
		{
			name: "nodeLabels: unordered matchExpressions and nil matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_2_val", "label_1_val"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"label_1": "label_1_val"}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_2_val", "label_1_val"},
					}},
				},
			}},
		},
		{
			name: "nodeFields: nil matchExpressions and preordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1", "host_2"},
					}},
				},
			}},
		},
		{
			name: "nodeFields: nil matchExpressions and unordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_2", "host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1"}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_2", "host_1"},
					}},
				},
			}},
		},
		{
			name: "nodeLabels and nodeFields: ordered matchExpressions and ordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val", "label_2_val"},
						}},
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val",
					}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val", "label_2_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1", "host_2"},
					}},
				},
			}},
		},
		{
			name: "nodeLabels and nodeFields: ordered matchExpressions and unordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_1_val", "label_2_val"},
						}},
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_2", "host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val",
					}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_1_val", "label_2_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_2", "host_1"},
					}},
				},
			}},
		},
		{
			name: "nodeLabels and nodeFields: unordered matchExpressions and ordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_2_val", "label_1_val"},
						}},
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_1", "host_2"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val",
					}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_2_val", "label_1_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_1", "host_2"},
					}},
				},
			}},
		},
		{
			name: "nodeLabels and nodeFields: unordered matchExpressions and unordered matchFields",
			args: args{
				nodeSelector: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
					{
						MatchExpressions: []corev1.RunnerSelectorRequirement{{
							Key:      "label_1",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"label_2_val", "label_1_val"},
						}},
						MatchFields: []corev1.RunnerSelectorRequirement{{
							Key:      "metadata.name",
							Operator: corev1.RunnerSelectorOpIn,
							Values:   []string{"host_2", "host_1"},
						}},
					},
				}},
				node: &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "host_1",
					Labels: map[string]string{
						"label_1": "label_1_val",
					}}},
			},
			want: &corev1.RunnerSelector{RunnerSelectorTerms: []corev1.RunnerSelectorTerm{
				{
					MatchExpressions: []corev1.RunnerSelectorRequirement{{
						Key:      "label_1",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"label_2_val", "label_1_val"},
					}},
					MatchFields: []corev1.RunnerSelectorRequirement{{
						Key:      "metadata.name",
						Operator: corev1.RunnerSelectorOpIn,
						Values:   []string{"host_2", "host_1"},
					}},
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = MatchRunnerSelectorTerms(tt.args.node, tt.args.nodeSelector)
			if !apiequality.Semantic.DeepEqual(tt.args.nodeSelector, tt.want) {
				// fail when tt.args.nodeSelector is deeply modified
				t.Errorf("MatchRunnerSelectorTerms() got = %v, want %v", tt.args.nodeSelector, tt.want)
			}
		})
	}
}

func TestGetAvoidRegionsFromRunner(t *testing.T) {
	controllerFlag := true
	testCases := []struct {
		node        *corev1.Runner
		expectValue corev1.AvoidRegions
		expectErr   bool
	}{
		{
			node:        &corev1.Runner{},
			expectValue: corev1.AvoidRegions{},
			expectErr:   false,
		},
		{
			node: &corev1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						corev1.PreferAvoidRegionsAnnotationKey: `
							{
							    "preferAvoidRegions": [
							        {
							            "podSignature": {
							                "podController": {
						                            "apiVersion": "corev1",
						                            "kind": "ReplicationController",
						                            "name": "foo",
						                            "uid": "abcdef123456",
						                            "controller": true
							                }
							            },
							            "reason": "some reason",
							            "message": "some message"
							        }
							    ]
							}`,
					},
				},
			},
			expectValue: corev1.AvoidRegions{
				PreferAvoidRegions: []corev1.PreferAvoidRegionsEntry{
					{
						RegionSignature: corev1.RegionSignature{
							RegionController: &metav1.OwnerReference{
								APIVersion: "corev1",
								Kind:       "ReplicationController",
								Name:       "foo",
								UID:        "abcdef123456",
								Controller: &controllerFlag,
							},
						},
						Reason:  "some reason",
						Message: "some message",
					},
				},
			},
			expectErr: false,
		},
		{
			node: &corev1.Runner{
				// Missing end symbol of "podController" and "podSignature"
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						corev1.PreferAvoidRegionsAnnotationKey: `
							{
							    "preferAvoidRegions": [
							        {
							            "podSignature": {
							                "podController": {
							                    "kind": "ReplicationController",
							                    "apiVersion": "corev1"
							            "reason": "some reason",
							            "message": "some message"
							        }
							    ]
							}`,
					},
				},
			},
			expectValue: corev1.AvoidRegions{},
			expectErr:   true,
		},
	}

	for i, tc := range testCases {
		v, err := GetAvoidRegionsFromRunnerAnnotations(tc.node.Annotations)
		if err == nil && tc.expectErr {
			t.Errorf("[%v]expected error but got none.", i)
		}
		if err != nil && !tc.expectErr {
			t.Errorf("[%v]did not expect error but got: %v", i, err)
		}
		if !reflect.DeepEqual(tc.expectValue, v) {
			t.Errorf("[%v]expect value %v but got %v with %v", i, tc.expectValue, v, v.PreferAvoidRegions[0].RegionSignature.RegionController.Controller)
		}
	}
}

func TestFindMatchingUntoleratedTaint(t *testing.T) {
	testCases := []struct {
		description     string
		tolerations     []corev1.Toleration
		taints          []corev1.Taint
		applyFilter     taintsFilterFunc
		expectTolerated bool
	}{
		{
			description:     "empty tolerations tolerate empty taints",
			tolerations:     []corev1.Toleration{},
			taints:          []corev1.Taint{},
			applyFilter:     func(t *corev1.Taint) bool { return true },
			expectTolerated: true,
		},
		{
			description: "non-empty tolerations tolerate empty taints",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: "Exists",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			taints:          []corev1.Taint{},
			applyFilter:     func(t *corev1.Taint) bool { return true },
			expectTolerated: true,
		},
		{
			description: "tolerations match all taints, expect tolerated",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: "Exists",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			taints: []corev1.Taint{
				{
					Key:    "foo",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
			applyFilter:     func(t *corev1.Taint) bool { return true },
			expectTolerated: true,
		},
		{
			description: "tolerations don't match taints, but no taints apply to the filter, expect tolerated",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: "Exists",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			taints: []corev1.Taint{
				{
					Key:    "bar",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
			applyFilter:     func(t *corev1.Taint) bool { return false },
			expectTolerated: true,
		},
		{
			description: "no filterFunc indicated, means all taints apply to the filter, tolerations don't match taints, expect untolerated",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: "Exists",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			taints: []corev1.Taint{
				{
					Key:    "bar",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
			applyFilter:     nil,
			expectTolerated: false,
		},
		{
			description: "tolerations match taints, expect tolerated",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: "Exists",
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
			taints: []corev1.Taint{
				{
					Key:    "foo",
					Effect: corev1.TaintEffectNoExecute,
				},
				{
					Key:    "bar",
					Effect: corev1.TaintEffectNoSchedule,
				},
			},
			applyFilter:     func(t *corev1.Taint) bool { return t.Effect == corev1.TaintEffectNoExecute },
			expectTolerated: true,
		},
	}

	for _, tc := range testCases {
		_, untolerated := FindMatchingUntoleratedTaint(tc.taints, tc.tolerations, tc.applyFilter)
		if tc.expectTolerated != !untolerated {
			filteredTaints := []corev1.Taint{}
			for _, taint := range tc.taints {
				if tc.applyFilter != nil && !tc.applyFilter(&taint) {
					continue
				}
				filteredTaints = append(filteredTaints, taint)
			}
			t.Errorf("[%s] expect tolerations %+v tolerate filtered taints %+v in taints %+v", tc.description, tc.tolerations, filteredTaints, tc.taints)
		}
	}
}
