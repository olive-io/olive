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
//	"github.com/google/go-cmp/cmp"
//	"k8s.io/apimachinery/pkg/api/resource"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/apimachinery/pkg/runtime/schema"
//	"k8s.io/client-go/dynamic/dynamicinformer"
//	dyfake "k8s.io/client-go/dynamic/fake"
//	"k8s.io/klog/v2/ktesting"
//
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	"github.com/olive-io/olive/client-go/generated/clientset/versioned/fake"
//	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	"github.com/olive-io/olive/mon/scheduler/internal/cache"
//	"github.com/olive-io/olive/mon/scheduler/internal/queue"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//)
//
//func TestRunnerAllocatableChanged(t *testing.T) {
//	newQuantity := func(value int64) resource.Quantity {
//		return *resource.NewQuantity(value, resource.BinarySI)
//	}
//	for _, test := range []struct {
//		Name           string
//		Changed        bool
//		OldAllocatable corev1.ResourceList
//		NewAllocatable corev1.ResourceList
//	}{
//		{
//			Name:           "no allocatable resources changed",
//			Changed:        false,
//			OldAllocatable: corev1.ResourceList{corev1.ResourceMemory: newQuantity(1024)},
//			NewAllocatable: corev1.ResourceList{corev1.ResourceMemory: newQuantity(1024)},
//		},
//		{
//			Name:           "new runner has more allocatable resources",
//			Changed:        true,
//			OldAllocatable: corev1.ResourceList{corev1.ResourceMemory: newQuantity(1024)},
//			NewAllocatable: corev1.ResourceList{corev1.ResourceMemory: newQuantity(1024), corev1.ResourceStorage: newQuantity(1024)},
//		},
//	} {
//		t.Run(test.Name, func(t *testing.T) {
//			oldRunner := &corev1.Runner{Status: corev1.RunnerStatus{Allocatable: test.OldAllocatable}}
//			newRunner := &corev1.Runner{Status: corev1.RunnerStatus{Allocatable: test.NewAllocatable}}
//			changed := runnerAllocatableChanged(newRunner, oldRunner)
//			if changed != test.Changed {
//				t.Errorf("runnerAllocatableChanged should be %t, got %t", test.Changed, changed)
//			}
//		})
//	}
//}
//
//func TestRunnerLabelsChanged(t *testing.T) {
//	for _, test := range []struct {
//		Name      string
//		Changed   bool
//		OldLabels map[string]string
//		NewLabels map[string]string
//	}{
//		{
//			Name:      "no labels changed",
//			Changed:   false,
//			OldLabels: map[string]string{"foo": "bar"},
//			NewLabels: map[string]string{"foo": "bar"},
//		},
//		// Labels changed.
//		{
//			Name:      "new runner has more labels",
//			Changed:   true,
//			OldLabels: map[string]string{"foo": "bar"},
//			NewLabels: map[string]string{"foo": "bar", "test": "value"},
//		},
//	} {
//		t.Run(test.Name, func(t *testing.T) {
//			oldRunner := &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Labels: test.OldLabels}}
//			newRunner := &corev1.Runner{ObjectMeta: metav1.ObjectMeta{Labels: test.NewLabels}}
//			changed := runnerLabelsChanged(newRunner, oldRunner)
//			if changed != test.Changed {
//				t.Errorf("Test case %q failed: should be %t, got %t", test.Name, test.Changed, changed)
//			}
//		})
//	}
//}
//
//func TestRunnerTaintsChanged(t *testing.T) {
//	for _, test := range []struct {
//		Name      string
//		Changed   bool
//		OldTaints []corev1.Taint
//		NewTaints []corev1.Taint
//	}{
//		{
//			Name:      "no taint changed",
//			Changed:   false,
//			OldTaints: []corev1.Taint{{Key: "key", Value: "value"}},
//			NewTaints: []corev1.Taint{{Key: "key", Value: "value"}},
//		},
//		{
//			Name:      "taint value changed",
//			Changed:   true,
//			OldTaints: []corev1.Taint{{Key: "key", Value: "value1"}},
//			NewTaints: []corev1.Taint{{Key: "key", Value: "value2"}},
//		},
//	} {
//		t.Run(test.Name, func(t *testing.T) {
//			oldRunner := &corev1.Runner{Spec: corev1.RunnerSpec{Taints: test.OldTaints}}
//			newRunner := &corev1.Runner{Spec: corev1.RunnerSpec{Taints: test.NewTaints}}
//			changed := runnerTaintsChanged(newRunner, oldRunner)
//			if changed != test.Changed {
//				t.Errorf("Test case %q failed: should be %t, not %t", test.Name, test.Changed, changed)
//			}
//		})
//	}
//}
//
//func TestRunnerConditionsChanged(t *testing.T) {
//	runnerConditionType := reflect.TypeOf(corev1.RunnerCondition{})
//	if runnerConditionType.NumField() != 6 {
//		t.Errorf("RunnerCondition type has changed. The runnerConditionsChanged() function must be reevaluated.")
//	}
//
//	for _, test := range []struct {
//		Name          string
//		Changed       bool
//		OldConditions []corev1.RunnerCondition
//		NewConditions []corev1.RunnerCondition
//	}{
//		{
//			Name:          "no condition changed",
//			Changed:       false,
//			OldConditions: []corev1.RunnerCondition{{Type: corev1.RunnerDiskPressure, Status: corev1.ConditionTrue}},
//			NewConditions: []corev1.RunnerCondition{{Type: corev1.RunnerDiskPressure, Status: corev1.ConditionTrue}},
//		},
//		{
//			Name:          "only LastHeartbeatTime changed",
//			Changed:       false,
//			OldConditions: []corev1.RunnerCondition{{Type: corev1.RunnerDiskPressure, Status: corev1.ConditionTrue, LastHeartbeatTime: metav1.Unix(1, 0)}},
//			NewConditions: []corev1.RunnerCondition{{Type: corev1.RunnerDiskPressure, Status: corev1.ConditionTrue, LastHeartbeatTime: metav1.Unix(2, 0)}},
//		},
//		{
//			Name:          "new runner has more healthy conditions",
//			Changed:       true,
//			OldConditions: []corev1.RunnerCondition{},
//			NewConditions: []corev1.RunnerCondition{{Type: corev1.RunnerReady, Status: corev1.ConditionTrue}},
//		},
//		{
//			Name:          "new runner has less unhealthy conditions",
//			Changed:       true,
//			OldConditions: []corev1.RunnerCondition{{Type: corev1.RunnerDiskPressure, Status: corev1.ConditionTrue}},
//			NewConditions: []corev1.RunnerCondition{},
//		},
//		{
//			Name:          "condition status changed",
//			Changed:       true,
//			OldConditions: []corev1.RunnerCondition{{Type: corev1.RunnerReady, Status: corev1.ConditionFalse}},
//			NewConditions: []corev1.RunnerCondition{{Type: corev1.RunnerReady, Status: corev1.ConditionTrue}},
//		},
//	} {
//		t.Run(test.Name, func(t *testing.T) {
//			oldRunner := &corev1.Runner{Status: corev1.RunnerStatus{Conditions: test.OldConditions}}
//			newRunner := &corev1.Runner{Status: corev1.RunnerStatus{Conditions: test.NewConditions}}
//			changed := runnerConditionsChanged(newRunner, oldRunner)
//			if changed != test.Changed {
//				t.Errorf("Test case %q failed: should be %t, got %t", test.Name, test.Changed, changed)
//			}
//		})
//	}
//}
//
//func TestUpdateRegionInCache(t *testing.T) {
//	ttl := 10 * time.Second
//	runnerName := "runner"
//
//	tests := []struct {
//		name   string
//		oldObj interface{}
//		newObj interface{}
//	}{
//		{
//			name:   "region updated with the same UID",
//			oldObj: withRegionName(regionWithPort("oldUID", runnerName, 80), "region"),
//			newObj: withRegionName(regionWithPort("oldUID", runnerName, 8080), "region"),
//		},
//		{
//			name:   "region updated with different UIDs",
//			oldObj: withRegionName(regionWithPort("oldUID", runnerName, 80), "region"),
//			newObj: withRegionName(regionWithPort("newUID", runnerName, 8080), "region"),
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			sched := &Scheduler{
//				Cache:           cache.New(ctx, ttl),
//				SchedulingQueue: queue.NewTestQueue(ctx, nil),
//				logger:          logger,
//			}
//			sched.addRegionToCache(tt.oldObj)
//			sched.updateRegionInCache(tt.oldObj, tt.newObj)
//
//			if tt.oldObj.(*corev1.Region).UID != tt.newObj.(*corev1.Region).UID {
//				if region, err := sched.Cache.GetRegion(tt.oldObj.(*corev1.Region)); err == nil {
//					t.Errorf("Get region UID %v from cache but it should not happen", region.UID)
//				}
//			}
//			region, err := sched.Cache.GetRegion(tt.newObj.(*corev1.Region))
//			if err != nil {
//				t.Errorf("Failed to get region from scheduler: %v", err)
//			}
//			if region.UID != tt.newObj.(*corev1.Region).UID {
//				t.Errorf("Want region UID %v, got %v", tt.newObj.(*corev1.Region).UID, region.UID)
//			}
//		})
//	}
//}
//
//func withRegionName(region *corev1.Region, name string) *corev1.Region {
//	region.Name = name
//	return region
//}
//
//func TestPreCheckForRunner(t *testing.T) {
//	tests := []struct {
//		name                     string
//		runnerFn                 func() *corev1.Runner
//		existingRegions, regions []*corev1.Region
//		want                     []bool
//	}{
//		{
//			name: "regular runner, regions with a single constraint",
//			runnerFn: func() *corev1.Runner {
//				return st.MakeRunner().Name("fake-runner").Label("hostname", "fake-runner").Obj()
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Name("r").Obj(),
//			},
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("r1").Obj(),
//				st.MakeRegion().Name("r2").Obj(),
//				st.MakeRegion().Name("r3").Obj(),
//				st.MakeRegion().Name("r4").RunnerAffinityIn("hostname", []string{"fake-runner"}).Obj(),
//				st.MakeRegion().Name("r5").RunnerAffinityNotIn("hostname", []string{"fake-runner"}).Obj(),
//				st.MakeRegion().Name("r6").Obj(),
//				st.MakeRegion().Name("r7").Runners("invalid-runner").Obj(),
//				st.MakeRegion().Name("r8").Obj(),
//				st.MakeRegion().Name("r9").Obj(),
//			},
//			want: []bool{true, false, false, true, false, true, false, true, false},
//		},
//		{
//			name: "tainted runner, regions with a single constraint",
//			runnerFn: func() *corev1.Runner {
//				runner := st.MakeRunner().Name("fake-runner").Obj()
//				runner.Spec.Taints = []corev1.Taint{
//					{Key: "foo", Effect: corev1.TaintEffectNoSchedule},
//					{Key: "bar", Effect: corev1.TaintEffectPreferNoSchedule},
//				}
//				return runner
//			},
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("p1").Obj(),
//				st.MakeRegion().Name("p2").Toleration("foo").Obj(),
//				st.MakeRegion().Name("p3").Toleration("bar").Obj(),
//				st.MakeRegion().Name("p4").Toleration("bar").Toleration("foo").Obj(),
//			},
//			want: []bool{false, true, false, true},
//		},
//		{
//			name: "regular runner, regions with multiple constraints",
//			runnerFn: func() *corev1.Runner {
//				return st.MakeRunner().Name("fake-runner").Label("hostname", "fake-runner").Obj()
//			},
//			existingRegions: []*corev1.Region{
//				st.MakeRegion().Name("r").Obj(),
//			},
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("r1").RunnerAffinityNotIn("hostname", []string{"fake-runner"}).Obj(),
//				st.MakeRegion().Name("r2").RunnerAffinityIn("hostname", []string{"fake-runner"}).Obj(),
//				st.MakeRegion().Name("r3").RunnerAffinityIn("hostname", []string{"fake-runner"}).Obj(),
//				st.MakeRegion().Name("r4").Runners("invalid-runner").Obj(),
//				st.MakeRegion().Name("r5").RunnerAffinityIn("hostname", []string{"fake-runner"}).Obj(),
//			},
//			want: []bool{false, false, true, false, false},
//		},
//		{
//			name: "tainted runner, regions with multiple constraints",
//			runnerFn: func() *corev1.Runner {
//				runner := st.MakeRunner().Name("fake-runner").Label("hostname", "fake-runner").Obj()
//				runner.Spec.Taints = []corev1.Taint{
//					{Key: "foo", Effect: corev1.TaintEffectNoSchedule},
//					{Key: "bar", Effect: corev1.TaintEffectPreferNoSchedule},
//				}
//				return runner
//			},
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("p1").Toleration("bar").Obj(),
//				st.MakeRegion().Name("p2").Toleration("bar").Toleration("foo").Obj(),
//				st.MakeRegion().Name("p3").Toleration("foo").Obj(),
//				st.MakeRegion().Name("p3").Toleration("bar").Obj(),
//			},
//			want: []bool{false, true, false, false},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			runnerInfo := framework.NewRunnerInfo(tt.existingRegions...)
//			runnerInfo.SetRunner(tt.runnerFn())
//			preCheckFn := preCheckForRunner(runnerInfo)
//
//			var got []bool
//			for _, region := range tt.regions {
//				got = append(got, preCheckFn(region))
//			}
//
//			if diff := cmp.Diff(tt.want, got); diff != "" {
//				t.Errorf("Unexpected diff (-want, +got):\n%s", diff)
//			}
//		})
//	}
//}
//
//// test for informers of resources we care about is registered
//func TestAddAllEventHandlers(t *testing.T) {
//	tests := []struct {
//		name                   string
//		gvkMap                 map[framework.GVK]framework.ActionType
//		expectStaticInformers  map[reflect.Type]bool
//		expectDynamicInformers map[schema.GroupVersionResource]bool
//	}{
//		{
//			name:   "default handlers in framework",
//			gvkMap: map[framework.GVK]framework.ActionType{},
//			expectStaticInformers: map[reflect.Type]bool{
//				reflect.TypeOf(&corev1.Region{}):    true,
//				reflect.TypeOf(&corev1.Runner{}):    true,
//				reflect.TypeOf(&corev1.Namespace{}): true,
//			},
//			expectDynamicInformers: map[schema.GroupVersionResource]bool{},
//		},
//		{
//			name: "add GVKs handlers defined in framework dynamically",
//			gvkMap: map[framework.GVK]framework.ActionType{
//				"Region":                            framework.Add | framework.Delete,
//				"PersistentVolume":                  framework.Delete,
//				"storage.k8s.io/CSIStorageCapacity": framework.Update,
//			},
//			expectStaticInformers: map[reflect.Type]bool{
//				reflect.TypeOf(&corev1.Region{}):    true,
//				reflect.TypeOf(&corev1.Runner{}):    true,
//				reflect.TypeOf(&corev1.Namespace{}): true,
//			},
//			expectDynamicInformers: map[schema.GroupVersionResource]bool{},
//		},
//		{
//			name: "add GVKs handlers defined in plugins dynamically",
//			gvkMap: map[framework.GVK]framework.ActionType{
//				"daemonsets.corecorev1.apps": framework.Add | framework.Delete,
//				"cronjobs.corecorev1.batch":  framework.Delete,
//			},
//			expectStaticInformers: map[reflect.Type]bool{
//				reflect.TypeOf(&corev1.Region{}):    true,
//				reflect.TypeOf(&corev1.Runner{}):    true,
//				reflect.TypeOf(&corev1.Namespace{}): true,
//			},
//			expectDynamicInformers: map[schema.GroupVersionResource]bool{
//				{Group: "apps", Version: "v1", Resource: "daemonsets"}: true,
//				{Group: "batch", Version: "v1", Resource: "cronjobs"}:  true,
//			},
//		},
//		{
//			name: "add GVKs handlers defined in plugins dynamically, with one illegal GVK form",
//			gvkMap: map[framework.GVK]framework.ActionType{
//				"daemonsets.v1.apps":    framework.Add | framework.Delete,
//				"custommetrics.v1beta1": framework.Update,
//			},
//			expectStaticInformers: map[reflect.Type]bool{
//				reflect.TypeOf(&corev1.Region{}):    true,
//				reflect.TypeOf(&corev1.Runner{}):    true,
//				reflect.TypeOf(&corev1.Namespace{}): true,
//			},
//			expectDynamicInformers: map[schema.GroupVersionResource]bool{
//				{Group: "apps", Version: "v1", Resource: "daemonsets"}: true,
//			},
//		},
//	}
//
//	scheme := runtime.NewScheme()
//	var localSchemeBuilder = runtime.SchemeBuilder{}
//	localSchemeBuilder.AddToScheme(scheme)
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			informerFactory := informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0)
//			schedulingQueue := queue.NewTestQueueWithInformerFactory(ctx, nil, informerFactory)
//			testSched := Scheduler{
//				StopEverything:  ctx.Done(),
//				SchedulingQueue: schedulingQueue,
//				logger:          logger,
//			}
//
//			dynclient := dyfake.NewSimpleDynamicClient(scheme)
//			dynInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynclient, 0)
//
//			if err := addAllEventHandlers(&testSched, informerFactory, dynInformerFactory, tt.gvkMap); err != nil {
//				t.Fatalf("Add event handlers failed, error = %v", err)
//			}
//
//			informerFactory.Start(testSched.StopEverything)
//			dynInformerFactory.Start(testSched.StopEverything)
//			staticInformers := informerFactory.WaitForCacheSync(testSched.StopEverything)
//			dynamicInformers := dynInformerFactory.WaitForCacheSync(testSched.StopEverything)
//
//			if diff := cmp.Diff(tt.expectStaticInformers, staticInformers); diff != "" {
//				t.Errorf("Unexpected diff (-want, +got):\n%s", diff)
//			}
//			if diff := cmp.Diff(tt.expectDynamicInformers, dynamicInformers); diff != "" {
//				t.Errorf("Unexpected diff (-want, +got):\n%s", diff)
//			}
//		})
//	}
//}
//
//func TestRunnerSchedulingPropertiesChange(t *testing.T) {
//	testCases := []struct {
//		name       string
//		newRunner  *corev1.Runner
//		oldRunner  *corev1.Runner
//		wantEvents []framework.ClusterEvent
//	}{
//		{
//			name:       "no specific changed applied",
//			newRunner:  st.MakeRunner().Unschedulable(false).Obj(),
//			oldRunner:  st.MakeRunner().Unschedulable(false).Obj(),
//			wantEvents: nil,
//		},
//		{
//			name:       "only runner spec unavailable changed",
//			newRunner:  st.MakeRunner().Unschedulable(false).Obj(),
//			oldRunner:  st.MakeRunner().Unschedulable(true).Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerSpecUnschedulableChange},
//		},
//		{
//			name: "only runner allocatable changed",
//			newRunner: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceCPU:                     "1000m",
//				corev1.ResourceMemory:                  "100m",
//				corev1.ResourceName("example.com/foo"): "1"},
//			).Obj(),
//			oldRunner: st.MakeRunner().Capacity(map[corev1.ResourceName]string{
//				corev1.ResourceCPU:                     "1000m",
//				corev1.ResourceMemory:                  "100m",
//				corev1.ResourceName("example.com/foo"): "2"},
//			).Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerAllocatableChange},
//		},
//		{
//			name:       "only runner label changed",
//			newRunner:  st.MakeRunner().Label("foo", "bar").Obj(),
//			oldRunner:  st.MakeRunner().Label("foo", "fuz").Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerLabelChange},
//		},
//		{
//			name: "only runner taint changed",
//			newRunner: st.MakeRunner().Taints([]corev1.Taint{
//				{Key: corev1.TaintRunnerUnschedulable, Value: "", Effect: corev1.TaintEffectNoSchedule},
//			}).Obj(),
//			oldRunner: st.MakeRunner().Taints([]corev1.Taint{
//				{Key: corev1.TaintRunnerUnschedulable, Value: "foo", Effect: corev1.TaintEffectNoSchedule},
//			}).Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerTaintChange},
//		},
//		{
//			name:       "only runner annotation changed",
//			newRunner:  st.MakeRunner().Annotation("foo", "bar").Obj(),
//			oldRunner:  st.MakeRunner().Annotation("foo", "fuz").Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerAnnotationChange},
//		},
//		{
//			name:      "only runner condition changed",
//			newRunner: st.MakeRunner().Obj(),
//			oldRunner: st.MakeRunner().Condition(
//				corev1.RunnerReady,
//				corev1.ConditionTrue,
//				"Ready",
//				"Ready",
//			).Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerConditionChange},
//		},
//		{
//			name: "both runner label and runner taint changed",
//			newRunner: st.MakeRunner().
//				Label("foo", "bar").
//				Taints([]corev1.Taint{
//					{Key: corev1.TaintRunnerUnschedulable, Value: "", Effect: corev1.TaintEffectNoSchedule},
//				}).Obj(),
//			oldRunner: st.MakeRunner().Taints([]corev1.Taint{
//				{Key: corev1.TaintRunnerUnschedulable, Value: "foo", Effect: corev1.TaintEffectNoSchedule},
//			}).Obj(),
//			wantEvents: []framework.ClusterEvent{queue.RunnerLabelChange, queue.RunnerTaintChange},
//		},
//	}
//
//	for _, tc := range testCases {
//		gotEvents := runnerSchedulingPropertiesChange(tc.newRunner, tc.oldRunner)
//		if diff := cmp.Diff(tc.wantEvents, gotEvents); diff != "" {
//			t.Errorf("unexpected event (-want, +got):\n%s", diff)
//		}
//	}
//}
