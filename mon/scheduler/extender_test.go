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

package scheduler

//
//import (
//	"context"
//	"reflect"
//	"testing"
//	"time"
//
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/util/sets"
//	"k8s.io/klog/v2/ktesting"
//
//	configv1 "github.com/olive-io/olive/apis/config/v1"
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	clientsetfake "github.com/olive-io/olive/client-go/generated/clientset/versioned/fake"
//	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
//	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/defaultbinder"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/queuesort"
//	"github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
//	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//)
//
//func TestSchedulerWithExtenders(t *testing.T) {
//	tests := []struct {
//		name            string
//		registerPlugins []tf.RegisterPluginFunc
//		extenders       []tf.FakeExtender
//		runners         []string
//		expectedResult  ScheduleResult
//		expectsErr      bool
//	}{
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.ErrorPredicateExtender},
//				},
//			},
//			runners:    []string{"runner1", "runner2"},
//			expectsErr: true,
//			name:       "test 1",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.FalsePredicateExtender},
//				},
//			},
//			runners:    []string{"runner1", "runner2"},
//			expectsErr: true,
//			name:       "test 2",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.Runner1PredicateExtender},
//				},
//			},
//			runners: []string{"runner1", "runner2"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner1",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  1,
//			},
//			name: "test 3",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.Runner2PredicateExtender},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.Runner1PredicateExtender},
//				},
//			},
//			runners:    []string{"runner1", "runner2"},
//			expectsErr: true,
//			name:       "test 4",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//					Prioritizers: []tf.PriorityConfig{{Function: tf.ErrorPrioritizerExtender, Weight: 10}},
//					Weight:       1,
//				},
//			},
//			runners: []string{"runner1"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner1",
//				EvaluatedRunners: 1,
//				FeasibleRunners:  1,
//			},
//			name: "test 5",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//					Prioritizers: []tf.PriorityConfig{{Function: tf.Runner1PrioritizerExtender, Weight: 10}},
//					Weight:       1,
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//					Prioritizers: []tf.PriorityConfig{{Function: tf.Runner2PrioritizerExtender, Weight: 10}},
//					Weight:       5,
//				},
//			},
//			runners: []string{"runner1", "runner2"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner2",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  2,
//			},
//			name: "test 6",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("Runner2Prioritizer", tf.NewRunner2PrioritizerPlugin(), 20),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//					Prioritizers: []tf.PriorityConfig{{Function: tf.Runner1PrioritizerExtender, Weight: 10}},
//					Weight:       1,
//				},
//			},
//			runners: []string{"runner1", "runner2"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner2",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  2,
//			}, // runner2 has higher score
//			name: "test 7",
//		},
//		{
//			// Scheduler is expected to not send region to extender in
//			// Filter/Prioritize phases if the extender is not interested in
//			// the region.
//			//
//			// If scheduler sends the region by mistake, the test would fail
//			// because of the errors from errorPredicateExtender and/or
//			// errorPrioritizerExtender.
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("Runner2Prioritizer", tf.NewRunner2PrioritizerPlugin(), 1),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.ErrorPredicateExtender},
//					Prioritizers: []tf.PriorityConfig{{Function: tf.ErrorPrioritizerExtender, Weight: 10}},
//					UnInterested: true,
//				},
//			},
//			runners:    []string{"runner1", "runner2"},
//			expectsErr: false,
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner2",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  2,
//			}, // runner2 has higher score
//			name: "test 8",
//		},
//		{
//			// Scheduling is expected to not fail in
//			// Filter/Prioritize phases if the extender is not available and ignorable.
//			//
//			// If scheduler did not ignore the extender, the test would fail
//			// because of the errors from errorPredicateExtender.
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.ErrorPredicateExtender},
//					Ignorable:    true,
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.Runner1PredicateExtender},
//				},
//			},
//			runners:    []string{"runner1", "runner2"},
//			expectsErr: false,
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner1",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  1,
//			},
//			name: "test 9",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Predicates:   []tf.FitPredicate{tf.Runner1PredicateExtender},
//				},
//			},
//			runners: []string{"runner1", "runner2"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner1",
//				EvaluatedRunners: 2,
//				FeasibleRunners:  1,
//			},
//			name: "test 10 - no scoring, extender filters configured, multiple feasible runners are evaluated",
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Binder:       func() error { return nil },
//				},
//			},
//			runners: []string{"runner1", "runner2"},
//			expectedResult: ScheduleResult{
//				SuggestedHost:    "runner1",
//				EvaluatedRunners: 1,
//				FeasibleRunners:  1,
//			},
//			name: "test 11 - no scoring, no prefilters or  extender filters configured, a single feasible runner is evaluated",
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			client := clientsetfake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//
//			var extenders []framework.Extender
//			for ii := range test.extenders {
//				extenders = append(extenders, &test.extenders[ii])
//			}
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			cache := internalcache.New(ctx, time.Duration(0))
//			for _, name := range test.runners {
//				cache.AddRunner(logger, createRunner(name))
//			}
//			fwk, err := tf.NewFramework(
//				ctx,
//				test.registerPlugins, "",
//				runtime.WithClientSet(client),
//				runtime.WithInformerFactory(informerFactory),
//				runtime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//				runtime.WithLogger(logger),
//			)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			sched := &Scheduler{
//				Cache:                      cache,
//				runnerInfoSnapshot:         emptySnapshot,
//				percentageOfRunnersToScore: configv1.DefaultPercentageOfRunnersToScore,
//				Extenders:                  extenders,
//				logger:                     logger,
//			}
//			sched.applyDefaultHandlers()
//
//			regionIgnored := &corev1.Region{}
//			result, err := sched.ScheduleRegion(ctx, fwk, framework.NewCycleState(), regionIgnored)
//			if test.expectsErr {
//				if err == nil {
//					t.Errorf("Unexpected non-error, result %+v", result)
//				}
//			} else {
//				if err != nil {
//					t.Errorf("Unexpected error: %v", err)
//					return
//				}
//
//				if !reflect.DeepEqual(result, test.expectedResult) {
//					t.Errorf("Expected: %+v, Saw: %+v", test.expectedResult, result)
//				}
//			}
//		})
//	}
//}
//
//func createRunner(name string) *corev1.Runner {
//	return &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: name}}
//}
//
//func TestIsInterested(t *testing.T) {
//	mem := &HTTPExtender{
//		managedResources: sets.New[string](),
//	}
//	mem.managedResources.Insert("memory")
//
//	for _, tc := range []struct {
//		label    string
//		extender *HTTPExtender
//		region   *corev1.Region
//		want     bool
//	}{
//		{
//			label: "Empty managed resources",
//			extender: &HTTPExtender{
//				managedResources: sets.New[string](),
//			},
//			region: &corev1.Region{},
//			want:   true,
//		},
//		{
//			label:    "Managed memory, empty resources",
//			extender: mem,
//			region:   st.MakeRegion().Obj(),
//			want:     false,
//		},
//		{
//			label:    "Managed memory, container memory with Requests",
//			extender: mem,
//			region: st.MakeRegion().Req(map[corev1.ResourceName]string{
//				"memory": "0",
//			}).Obj(),
//			want: true,
//		},
//		{
//			label:    "Managed memory, container memory with Limits",
//			extender: mem,
//			region: st.MakeRegion().Lim(map[corev1.ResourceName]string{
//				"memory": "0",
//			}).Obj(),
//			want: true,
//		},
//		{
//			label:    "Managed memory, init container memory",
//			extender: mem,
//			region: st.MakeRegion().Container("app").InitReq(map[corev1.ResourceName]string{
//				"memory": "0",
//			}).Obj(),
//			want: true,
//		},
//	} {
//		t.Run(tc.label, func(t *testing.T) {
//			if got := tc.extender.IsInterested(tc.region); got != tc.want {
//				t.Fatalf("IsInterested(%v) = %v, wanted %v", tc.region, got, tc.want)
//			}
//		})
//	}
//}
//
//func TestConvertToMetaVictims(t *testing.T) {
//	tests := []struct {
//		name                string
//		runnerNameToVictims map[string]*extenderv1.Victims
//		want                map[string]*extenderv1.MetaVictims
//	}{
//		{
//			name: "test NumPDBViolations is transferred from runnerNameToVictims to runnerNameToMetaVictims",
//			runnerNameToVictims: map[string]*extenderv1.Victims{
//				"runner1": {
//					Regions: []*corev1.Region{
//						st.MakeRegion().Name("region1").UID("uid1").Obj(),
//						st.MakeRegion().Name("region3").UID("uid3").Obj(),
//					},
//					NumPDBViolations: 1,
//				},
//				"runner2": {
//					Regions: []*corev1.Region{
//						st.MakeRegion().Name("region2").UID("uid2").Obj(),
//						st.MakeRegion().Name("region4").UID("uid4").Obj(),
//					},
//					NumPDBViolations: 2,
//				},
//			},
//			want: map[string]*extenderv1.MetaVictims{
//				"runner1": {
//					Regions: []*extenderv1.MetaRegion{
//						{UID: "uid1"},
//						{UID: "uid3"},
//					},
//					NumPDBViolations: 1,
//				},
//				"runner2": {
//					Regions: []*extenderv1.MetaRegion{
//						{UID: "uid2"},
//						{UID: "uid4"},
//					},
//					NumPDBViolations: 2,
//				},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := convertToMetaVictims(tt.runnerNameToVictims); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("convertToMetaVictims() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestConvertToVictims(t *testing.T) {
//	tests := []struct {
//		name                    string
//		httpExtender            *HTTPExtender
//		runnerNameToMetaVictims map[string]*extenderv1.MetaVictims
//		runnerNames             []string
//		regionsInRunnerList     []*corev1.Region
//		runnerInfos             framework.RunnerInfoLister
//		want                    map[string]*extenderv1.Victims
//		wantErr                 bool
//	}{
//		{
//			name:         "test NumPDBViolations is transferred from RunnerNameToMetaVictims to newRunnerNameToVictims",
//			httpExtender: &HTTPExtender{},
//			runnerNameToMetaVictims: map[string]*extenderv1.MetaVictims{
//				"runner1": {
//					Regions: []*extenderv1.MetaRegion{
//						{UID: "uid1"},
//						{UID: "uid3"},
//					},
//					NumPDBViolations: 1,
//				},
//				"runner2": {
//					Regions: []*extenderv1.MetaRegion{
//						{UID: "uid2"},
//						{UID: "uid4"},
//					},
//					NumPDBViolations: 2,
//				},
//			},
//			runnerNames: []string{"runner1", "runner2"},
//			regionsInRunnerList: []*corev1.Region{
//				st.MakeRegion().Name("region1").UID("uid1").Obj(),
//				st.MakeRegion().Name("region2").UID("uid2").Obj(),
//				st.MakeRegion().Name("region3").UID("uid3").Obj(),
//				st.MakeRegion().Name("region4").UID("uid4").Obj(),
//			},
//			runnerInfos: nil,
//			want: map[string]*extenderv1.Victims{
//				"runner1": {
//					Regions: []*corev1.Region{
//						st.MakeRegion().Name("region1").UID("uid1").Obj(),
//						st.MakeRegion().Name("region3").UID("uid3").Obj(),
//					},
//					NumPDBViolations: 1,
//				},
//				"runner2": {
//					Regions: []*corev1.Region{
//						st.MakeRegion().Name("region2").UID("uid2").Obj(),
//						st.MakeRegion().Name("region4").UID("uid4").Obj(),
//					},
//					NumPDBViolations: 2,
//				},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// runnerInfos instantiations
//			runnerInfoList := make([]*framework.RunnerInfo, 0, len(tt.runnerNames))
//			for i, nm := range tt.runnerNames {
//				runnerInfo := framework.NewRunnerInfo()
//				runner := createRunner(nm)
//				runnerInfo.SetRunner(runner)
//				runnerInfo.AddRegion(tt.regionsInRunnerList[i])
//				runnerInfo.AddRegion(tt.regionsInRunnerList[i+2])
//				runnerInfoList = append(runnerInfoList, runnerInfo)
//			}
//			tt.runnerInfos = tf.RunnerInfoLister(runnerInfoList)
//
//			got, err := tt.httpExtender.convertToVictims(tt.runnerNameToMetaVictims, tt.runnerInfos)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("convertToVictims() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("convertToVictims() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
