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

//import (
//	"context"
//	"fmt"
//	"sort"
//	"strings"
//	"testing"
//	"time"
//
//	"github.com/google/go-cmp/cmp"
//	"github.com/olive-io/olive/mon/scheduler/apis/config/testing/defaults"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/queuesort"
//	configv1 "github.com/olive-io/olive/apis/config/v1"
//	"github.com/olive-io/olive/client-go/generated/clientset/versioned/fake"
//	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//	v1 "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	utilfeature "k8s.io/apiserver/pkg/util/feature"
//	"k8s.io/client-go/kubernetes"
//	"k8s.io/client-go/tools/cache"
//	"k8s.io/client-go/tools/events"
//	featuregatetesting "k8s.io/component-base/featuregate/testing"
//	"k8s.io/klog/v2"
//	"k8s.io/klog/v2/ktesting"
//	testingclock "k8s.io/utils/clock/testing"
//	"k8s.io/utils/ptr"
//
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/defaultbinder"
//
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	"github.com/olive-io/olive/mon/features"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins"
//	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
//	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
//	"github.com/olive-io/olive/mon/scheduler/profile"
//)
//
//func TestSchedulerCreation(t *testing.T) {
//	invalidRegistry := map[string]frameworkruntime.PluginFactory{
//		defaultbinder.Name: defaultbinder.New,
//	}
//	validRegistry := map[string]frameworkruntime.PluginFactory{
//		"Foo": defaultbinder.New,
//	}
//	cases := []struct {
//		name          string
//		opts          []Option
//		wantErr       string
//		wantProfiles  []string
//		wantExtenders []string
//	}{
//		{
//			name: "valid out-of-tree registry",
//			opts: []Option{
//				WithFrameworkOutOfTreeRegistry(validRegistry),
//				WithProfiles(
//					configv1.SchedulerProfile{
//						SchedulerName: "default-scheduler",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//				)},
//			wantProfiles: []string{"default-scheduler"},
//		},
//		{
//			name: "repeated plugin name in out-of-tree plugin",
//			opts: []Option{
//				WithFrameworkOutOfTreeRegistry(invalidRegistry),
//				WithProfiles(
//					configv1.SchedulerProfile{
//						SchedulerName: "default-scheduler",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//				)},
//			wantProfiles: []string{"default-scheduler"},
//			wantErr:      "a plugin named DefaultBinder already exists",
//		},
//		{
//			name: "multiple profiles",
//			opts: []Option{
//				WithProfiles(
//					configv1.SchedulerProfile{
//						SchedulerName: "foo",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//					configv1.SchedulerProfile{
//						SchedulerName: "bar",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//				)},
//			wantProfiles: []string{"bar", "foo"},
//		},
//		{
//			name: "Repeated profiles",
//			opts: []Option{
//				WithProfiles(
//					configv1.SchedulerProfile{
//						SchedulerName: "foo",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//					configv1.SchedulerProfile{
//						SchedulerName: "bar",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//					configv1.SchedulerProfile{
//						SchedulerName: "foo",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//				)},
//			wantErr: "duplicate profile with scheduler name \"foo\"",
//		},
//		{
//			name: "With extenders",
//			opts: []Option{
//				WithProfiles(
//					configv1.SchedulerProfile{
//						SchedulerName: "default-scheduler",
//						Plugins: &configv1.Plugins{
//							QueueSort: configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "PrioritySort"}}},
//							Bind:      configv1.PluginSet{Enabled: []configv1.Plugin{{Name: "DefaultBinder"}}},
//						},
//					},
//				),
//				WithExtenders(
//					configv1.Extender{
//						URLPrefix: "http://extender.kube-system/",
//					},
//				),
//			},
//			wantProfiles:  []string{"default-scheduler"},
//			wantExtenders: []string{"http://extender.kube-system/"},
//		},
//	}
//
//	for _, tc := range cases {
//		t.Run(tc.name, func(t *testing.T) {
//			client := fake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//
//			eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			s, err := New(
//				ctx,
//				client,
//				informerFactory,
//				nil,
//				profile.NewRecorderFactory(eventBroadcaster),
//				tc.opts...,
//			)
//
//			// Errors
//			if len(tc.wantErr) != 0 {
//				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
//					t.Errorf("got error %q, want %q", err, tc.wantErr)
//				}
//				return
//			}
//			if err != nil {
//				t.Fatalf("Failed to create scheduler: %v", err)
//			}
//
//			// Profiles
//			profiles := make([]string, 0, len(s.Profiles))
//			for name := range s.Profiles {
//				profiles = append(profiles, name)
//			}
//			sort.Strings(profiles)
//			if diff := cmp.Diff(tc.wantProfiles, profiles); diff != "" {
//				t.Errorf("unexpected profiles (-want, +got):\n%s", diff)
//			}
//
//			// Extenders
//			if len(tc.wantExtenders) != 0 {
//				// Scheduler.Extenders
//				extenders := make([]string, 0, len(s.Extenders))
//				for _, e := range s.Extenders {
//					extenders = append(extenders, e.Name())
//				}
//				if diff := cmp.Diff(tc.wantExtenders, extenders); diff != "" {
//					t.Errorf("unexpected extenders (-want, +got):\n%s", diff)
//				}
//
//				// framework.Handle.Extenders()
//				for _, p := range s.Profiles {
//					extenders := make([]string, 0, len(p.Extenders()))
//					for _, e := range p.Extenders() {
//						extenders = append(extenders, e.Name())
//					}
//					if diff := cmp.Diff(tc.wantExtenders, extenders); diff != "" {
//						t.Errorf("unexpected extenders (-want, +got):\n%s", diff)
//					}
//				}
//			}
//		})
//	}
//}
//
//func TestFailureHandler(t *testing.T) {
//	testRegion := st.MakeRegion().Name("test-region").Namespace(v1.NamespaceDefault).Obj()
//	testRegionUpdated := testRegion.DeepCopy()
//	testRegionUpdated.Labels = map[string]string{"foo": ""}
//
//	tests := []struct {
//		name                          string
//		regionUpdatedDuringScheduling bool // region is updated during a scheduling cycle
//		regionDeletedDuringScheduling bool // region is deleted during a scheduling cycle
//		expect                        *corev1.Region
//	}{
//		{
//			name:                          "region is updated during a scheduling cycle",
//			regionUpdatedDuringScheduling: true,
//			expect:                        testRegionUpdated,
//		},
//		{
//			name:   "region is not updated during a scheduling cycle",
//			expect: testRegion,
//		},
//		{
//			name:                          "region is deleted during a scheduling cycle",
//			regionDeletedDuringScheduling: true,
//			expect:                        nil,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			client := fake.NewSimpleClientset(&corev1.RegionList{Items: []corev1.Region{*testRegion}})
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//			regionInformer := informerFactory.Core().V1().Regions()
//			// Need to add/update/delete testRegion to the store.
//			regionInformer.Informer().GetStore().Add(testRegion)
//
//			queue := internalqueue.NewPriorityQueue(nil, informerFactory, internalqueue.WithClock(testingclock.NewFakeClock(time.Now())))
//			schedulerCache := internalcache.New(ctx, 30*time.Second)
//
//			if err := queue.Add(logger, testRegion); err != nil {
//				t.Fatalf("Add failed: %v", err)
//			}
//
//			if _, err := queue.Pop(logger); err != nil {
//				t.Fatalf("Pop failed: %v", err)
//			}
//
//			if tt.regionUpdatedDuringScheduling {
//				regionInformer.Informer().GetStore().Update(testRegionUpdated)
//				queue.Update(logger, testRegion, testRegionUpdated)
//			}
//			if tt.regionDeletedDuringScheduling {
//				regionInformer.Informer().GetStore().Delete(testRegion)
//				queue.Delete(testRegion)
//			}
//
//			s, fwk, err := initScheduler(ctx, schedulerCache, queue, client, informerFactory)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			testRegionInfo := &framework.QueuedRegionInfo{RegionInfo: mustNewRegionInfo(t, testRegion)}
//			s.FailureHandler(ctx, fwk, testRegionInfo, framework.NewStatus(framework.Unschedulable), nil, time.Now())
//
//			var got *corev1.Region
//			if tt.regionUpdatedDuringScheduling {
//				head, e := queue.Pop(logger)
//				if e != nil {
//					t.Fatalf("Cannot pop region from the activeQ: %v", e)
//				}
//				got = head.Region
//			} else {
//				got = getRegionFromPriorityQueue(queue, testRegion)
//			}
//
//			if diff := cmp.Diff(tt.expect, got); diff != "" {
//				t.Errorf("Unexpected region (-want, +got): %s", diff)
//			}
//		})
//	}
//}
//
//func TestFailureHandler_RegionAlreadyBound(t *testing.T) {
//	logger, ctx := ktesting.NewTestContext(t)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//
//	runnerFoo := v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "foo"}}
//	testRegion := st.MakeRegion().Name("test-region").Namespace(v1.NamespaceDefault).Runner("foo").Obj()
//
//	client := fake.NewSimpleClientset(&corev1.RegionList{Items: []corev1.Region{*testRegion}}, &v1.RunnerList{Items: []v1.Runner{runnerFoo}})
//	informerFactory := informers.NewSharedInformerFactory(client, 0)
//	regionInformer := informerFactory.Core().V1().Regions()
//	// Need to add testRegion to the store.
//	regionInformer.Informer().GetStore().Add(testRegion)
//
//	queue := internalqueue.NewPriorityQueue(nil, informerFactory, internalqueue.WithClock(testingclock.NewFakeClock(time.Now())))
//	schedulerCache := internalcache.New(ctx, 30*time.Second)
//
//	// Add runner to schedulerCache no matter it's deleted in API server or not.
//	schedulerCache.AddRunner(logger, &runnerFoo)
//
//	s, fwk, err := initScheduler(ctx, schedulerCache, queue, client, informerFactory)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	testRegionInfo := &framework.QueuedRegionInfo{RegionInfo: mustNewRegionInfo(t, testRegion)}
//	s.FailureHandler(ctx, fwk, testRegionInfo, framework.NewStatus(framework.Unschedulable).WithError(fmt.Errorf("binding rejected: timeout")), nil, time.Now())
//
//	region := getRegionFromPriorityQueue(queue, testRegion)
//	if region != nil {
//		t.Fatalf("Unexpected region: %v should not be in PriorityQueue when the RunnerName of region is not empty", region.Name)
//	}
//}
//
//// TestWithPercentageOfRunnersToScore tests scheduler's PercentageOfRunnersToScore is set correctly.
//func TestWithPercentageOfRunnersToScore(t *testing.T) {
//	tests := []struct {
//		name                             string
//		percentageOfRunnersToScoreConfig *int32
//		wantedPercentageOfRunnersToScore int32
//	}{
//		{
//			name:                             "percentageOfRunnersScore is nil",
//			percentageOfRunnersToScoreConfig: nil,
//			wantedPercentageOfRunnersToScore: configv1.DefaultPercentageOfRunnersToScore,
//		},
//		{
//			name:                             "percentageOfRunnersScore is not nil",
//			percentageOfRunnersToScoreConfig: ptr.To[int32](10),
//			wantedPercentageOfRunnersToScore: 10,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			client := fake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//			eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			sched, err := New(
//				ctx,
//				client,
//				informerFactory,
//				nil,
//				profile.NewRecorderFactory(eventBroadcaster),
//				WithPercentageOfRunnersToScore(tt.percentageOfRunnersToScoreConfig),
//			)
//			if err != nil {
//				t.Fatalf("Failed to create scheduler: %v", err)
//			}
//			if sched.percentageOfRunnersToScore != tt.wantedPercentageOfRunnersToScore {
//				t.Errorf("scheduler.percercentageOfRunnersToScore = %v, want %v", sched.percentageOfRunnersToScore, tt.wantedPercentageOfRunnersToScore)
//			}
//		})
//	}
//}
//
//// getRegionFromPriorityQueue is the function used in the TestDefaultErrorFunc test to get
//// the specific region from the given priority queue. It returns the found region in the priority queue.
//func getRegionFromPriorityQueue(queue *internalqueue.PriorityQueue, region *corev1.Region) *corev1.Region {
//	regionList, _ := queue.PendingRegions()
//	if len(regionList) == 0 {
//		return nil
//	}
//
//	queryRegionKey, err := cache.MetaNamespaceKeyFunc(region)
//	if err != nil {
//		return nil
//	}
//
//	for _, foundRegion := range regionList {
//		foundRegionKey, err := cache.MetaNamespaceKeyFunc(foundRegion)
//		if err != nil {
//			return nil
//		}
//
//		if foundRegionKey == queryRegionKey {
//			return foundRegion
//		}
//	}
//
//	return nil
//}
//
//func initScheduler(ctx context.Context, cache internalcache.Cache, queue internalqueue.SchedulingQueue,
//	client kubernetes.Interface, informerFactory informers.SharedInformerFactory) (*Scheduler, framework.Framework, error) {
//	logger := klog.FromContext(ctx)
//	registerPluginFuncs := []tf.RegisterPluginFunc{
//		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//	}
//	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//	fwk, err := tf.NewFramework(ctx,
//		registerPluginFuncs,
//		testSchedulerName,
//		frameworkruntime.WithClientSet(client),
//		frameworkruntime.WithInformerFactory(informerFactory),
//		frameworkruntime.WithEventRecorder(eventBroadcaster.NewRecorder(scheme.Scheme, testSchedulerName)),
//	)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	s := &Scheduler{
//		Cache:           cache,
//		client:          client,
//		StopEverything:  ctx.Done(),
//		SchedulingQueue: queue,
//		Profiles:        profile.Map{testSchedulerName: fwk},
//		logger:          logger,
//	}
//	s.applyDefaultHandlers()
//
//	return s, fwk, nil
//}
//
//func TestInitPluginsWithIndexers(t *testing.T) {
//	tests := []struct {
//		name string
//		// the plugin registration ordering must not matter, being map traversal random
//		entrypoints map[string]frameworkruntime.PluginFactory
//		wantErr     string
//	}{
//		{
//			name: "register indexer, no conflicts",
//			entrypoints: map[string]frameworkruntime.PluginFactory{
//				"AddIndexer": func(ctx context.Context, obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//					regionInformer := handle.SharedInformerFactory().Core().V1().Regions()
//					err := regionInformer.Informer().AddIndexers(cache.Indexers{
//						"runnerName": indexByRegionSpecRunnerName,
//					})
//					return &TestPlugin{name: "AddIndexer"}, err
//				},
//			},
//		},
//		{
//			name: "register the same indexer name multiple times, conflict",
//			// order of registration doesn't matter
//			entrypoints: map[string]frameworkruntime.PluginFactory{
//				"AddIndexer1": func(ctx context.Context, obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//					regionInformer := handle.SharedInformerFactory().Core().V1().Regions()
//					err := regionInformer.Informer().AddIndexers(cache.Indexers{
//						"runnerName": indexByRegionSpecRunnerName,
//					})
//					return &TestPlugin{name: "AddIndexer1"}, err
//				},
//				"AddIndexer2": func(ctx context.Context, obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//					regionInformer := handle.SharedInformerFactory().Core().V1().Regions()
//					err := regionInformer.Informer().AddIndexers(cache.Indexers{
//						"runnerName": indexByRegionAnnotationRunnerName,
//					})
//					return &TestPlugin{name: "AddIndexer1"}, err
//				},
//			},
//			wantErr: "indexer conflict",
//		},
//		{
//			name: "register the same indexer body with different names, no conflicts",
//			// order of registration doesn't matter
//			entrypoints: map[string]frameworkruntime.PluginFactory{
//				"AddIndexer1": func(ctx context.Context, obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//					regionInformer := handle.SharedInformerFactory().Core().V1().Regions()
//					err := regionInformer.Informer().AddIndexers(cache.Indexers{
//						"runnerName1": indexByRegionSpecRunnerName,
//					})
//					return &TestPlugin{name: "AddIndexer1"}, err
//				},
//				"AddIndexer2": func(ctx context.Context, obj runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//					regionInformer := handle.SharedInformerFactory().Core().V1().Regions()
//					err := regionInformer.Informer().AddIndexers(cache.Indexers{
//						"runnerName2": indexByRegionAnnotationRunnerName,
//					})
//					return &TestPlugin{name: "AddIndexer2"}, err
//				},
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			fakeInformerFactory := NewInformerFactory(&fake.Clientset{}, 0*time.Second)
//
//			var registerPluginFuncs []tf.RegisterPluginFunc
//			for name, entrypoint := range tt.entrypoints {
//				registerPluginFuncs = append(registerPluginFuncs,
//					// anything supported by TestPlugin is fine
//					tf.RegisterFilterPlugin(name, entrypoint),
//				)
//			}
//			// we always need this
//			registerPluginFuncs = append(registerPluginFuncs,
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			)
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			_, err := tf.NewFramework(ctx, registerPluginFuncs, "test", frameworkruntime.WithInformerFactory(fakeInformerFactory))
//
//			if len(tt.wantErr) > 0 {
//				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
//					t.Errorf("got error %q, want %q", err, tt.wantErr)
//				}
//				return
//			}
//			if err != nil {
//				t.Fatalf("Failed to create scheduler: %v", err)
//			}
//		})
//	}
//}
//
//func indexByRegionAnnotationRunnerName(obj interface{}) ([]string, error) {
//	region, ok := obj.(*corev1.Region)
//	if !ok {
//		return []string{}, nil
//	}
//	if len(region.Annotations) == 0 {
//		return []string{}, nil
//	}
//	runnerName, ok := region.Annotations["runner-name"]
//	if !ok {
//		return []string{}, nil
//	}
//	return []string{runnerName}, nil
//}
//
//const (
//	filterWithoutEnqueueExtensions = "filterWithoutEnqueueExtensions"
//	fakeRunner                     = "fakeRunner"
//	fakeRegion                     = "fakeRegion"
//	emptyEventsToRegister          = "emptyEventsToRegister"
//	queueSort                      = "no-op-queue-sort-plugin"
//	fakeBind                       = "bind-plugin"
//	emptyEventExtensions           = "emptyEventExtensions"
//)
//
//func Test_buildQueueingHintMap(t *testing.T) {
//	tests := []struct {
//		name                string
//		plugins             []framework.Plugin
//		want                map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction
//		featuregateDisabled bool
//	}{
//		{
//			name:    "filter without EnqueueExtensions plugin",
//			plugins: []framework.Plugin{&filterWithoutEnqueueExtensionsPlugin{}},
//			want: map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction{
//				{Resource: framework.Region, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.Runner, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.RegionSchedulingContext, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.ResourceClaim, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.ResourceClass, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.ResourceClaimParameters, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//				{Resource: framework.ResourceClassParameters, ActionType: framework.All}: {
//					{PluginName: filterWithoutEnqueueExtensions, QueueingHintFn: defaultQueueingHintFn},
//				},
//			},
//		},
//		{
//			name:    "runner and region plugin",
//			plugins: []framework.Plugin{&fakeRunnerPlugin{}, &fakeRegionPlugin{}},
//			want: map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction{
//				{Resource: framework.Region, ActionType: framework.Add}: {
//					{PluginName: fakeRegion, QueueingHintFn: fakeRegionPluginQueueingFn},
//				},
//				{Resource: framework.Runner, ActionType: framework.Add}: {
//					{PluginName: fakeRunner, QueueingHintFn: fakeRunnerPluginQueueingFn},
//				},
//				{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint}: {
//					{PluginName: fakeRunner, QueueingHintFn: defaultQueueingHintFn}, // When Runner/Add is registered, Runner/UpdateRunnerTaint is automatically registered.
//				},
//			},
//		},
//		{
//			name:                "runner and region plugin (featuregate is disabled)",
//			plugins:             []framework.Plugin{&fakeRunnerPlugin{}, &fakeRegionPlugin{}},
//			featuregateDisabled: true,
//			want: map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction{
//				{Resource: framework.Region, ActionType: framework.Add}: {
//					{PluginName: fakeRegion, QueueingHintFn: defaultQueueingHintFn}, // default queueing hint due to disabled feature gate.
//				},
//				{Resource: framework.Runner, ActionType: framework.Add}: {
//					{PluginName: fakeRunner, QueueingHintFn: defaultQueueingHintFn}, // default queueing hint due to disabled feature gate.
//				},
//				{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint}: {
//					{PluginName: fakeRunner, QueueingHintFn: defaultQueueingHintFn}, // When Runner/Add is registered, Runner/UpdateRunnerTaint is automatically registered.
//				},
//			},
//		},
//		{
//			name:    "register plugin with empty event",
//			plugins: []framework.Plugin{&emptyEventPlugin{}},
//			want:    map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction{},
//		},
//		{
//			name:    "register plugins including emptyEventPlugin",
//			plugins: []framework.Plugin{&emptyEventPlugin{}, &fakeRunnerPlugin{}},
//			want: map[framework.ClusterEvent][]*internalqueue.QueueingHintFunction{
//				{Resource: framework.Region, ActionType: framework.Add}: {
//					{PluginName: fakeRegion, QueueingHintFn: fakeRegionPluginQueueingFn},
//				},
//				{Resource: framework.Runner, ActionType: framework.Add}: {
//					{PluginName: fakeRunner, QueueingHintFn: fakeRunnerPluginQueueingFn},
//				},
//				{Resource: framework.Runner, ActionType: framework.UpdateRunnerTaint}: {
//					{PluginName: fakeRunner, QueueingHintFn: defaultQueueingHintFn}, // When Runner/Add is registered, Runner/UpdateRunnerTaint is automatically registered.
//				},
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			defer featuregatetesting.SetFeatureGateDuringTest(t, utilfeature.DefaultFeatureGate, features.SchedulerQueueingHints, !tt.featuregateDisabled)()
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			registry := frameworkruntime.Registry{}
//			cfgPls := &configv1.Plugins{}
//			plugins := append(tt.plugins, &fakebindPlugin{}, &fakeQueueSortPlugin{})
//			for _, pl := range plugins {
//				tmpPl := pl
//				if err := registry.Register(pl.Name(), func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//					return tmpPl, nil
//				}); err != nil {
//					t.Fatalf("fail to register filter plugin (%s)", pl.Name())
//				}
//				cfgPls.MultiPoint.Enabled = append(cfgPls.MultiPoint.Enabled, configv1.Plugin{Name: pl.Name()})
//			}
//
//			profile := configv1.SchedulerProfile{Plugins: cfgPls}
//			fwk, err := newFramework(ctx, registry, profile)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			exts := fwk.EnqueueExtensions()
//			// need to sort to make the test result stable.
//			sort.Slice(exts, func(i, j int) bool {
//				return exts[i].Name() < exts[j].Name()
//			})
//
//			got := buildQueueingHintMap(exts)
//
//			for e, fns := range got {
//				wantfns, ok := tt.want[e]
//				if !ok {
//					t.Errorf("got unexpected event %v", e)
//					continue
//				}
//				if len(fns) != len(wantfns) {
//					t.Errorf("got %v queueing hint functions, want %v", len(fns), len(wantfns))
//					continue
//				}
//				for i, fn := range fns {
//					if fn.PluginName != wantfns[i].PluginName {
//						t.Errorf("got plugin name %v, want %v", fn.PluginName, wantfns[i].PluginName)
//						continue
//					}
//					got, gotErr := fn.QueueingHintFn(logger, nil, nil, nil)
//					want, wantErr := wantfns[i].QueueingHintFn(logger, nil, nil, nil)
//					if got != want || gotErr != wantErr {
//						t.Errorf("got queueing hint function (%v) returning (%v, %v), expect it to return (%v, %v)", fn.PluginName, got, gotErr, want, wantErr)
//						continue
//					}
//				}
//			}
//		})
//	}
//}
//
//// Test_UnionedGVKs tests UnionedGVKs worked with buildQueueingHintMap.
//func Test_UnionedGVKs(t *testing.T) {
//	tests := []struct {
//		name    string
//		plugins configv1.PluginSet
//		want    map[framework.GVK]framework.ActionType
//	}{
//		{
//			name: "filter without EnqueueExtensions plugin",
//			plugins: configv1.PluginSet{
//				Enabled: []configv1.Plugin{
//					{Name: filterWithoutEnqueueExtensions},
//					{Name: queueSort},
//					{Name: fakeBind},
//				},
//				Disabled: []configv1.Plugin{{Name: "*"}}, // disable default plugins
//			},
//			want: map[framework.GVK]framework.ActionType{
//				framework.Region:                  framework.All,
//				framework.Runner:                  framework.All,
//				framework.CSIRunner:               framework.All,
//				framework.CSIDriver:               framework.All,
//				framework.CSIStorageCapacity:      framework.All,
//				framework.PersistentVolume:        framework.All,
//				framework.PersistentVolumeClaim:   framework.All,
//				framework.StorageClass:            framework.All,
//				framework.RegionSchedulingContext: framework.All,
//				framework.ResourceClaim:           framework.All,
//				framework.ResourceClass:           framework.All,
//				framework.ResourceClaimParameters: framework.All,
//				framework.ResourceClassParameters: framework.All,
//			},
//		},
//		{
//			name: "runner plugin",
//			plugins: configv1.PluginSet{
//				Enabled: []configv1.Plugin{
//					{Name: fakeRunner},
//					{Name: queueSort},
//					{Name: fakeBind},
//				},
//				Disabled: []configv1.Plugin{{Name: "*"}}, // disable default plugins
//			},
//			want: map[framework.GVK]framework.ActionType{
//				framework.Runner: framework.Add | framework.UpdateRunnerTaint, // When Runner/Add is registered, Runner/UpdateRunnerTaint is automatically registered.
//			},
//		},
//		{
//			name: "region plugin",
//			plugins: configv1.PluginSet{
//				Enabled: []configv1.Plugin{
//					{Name: fakeRegion},
//					{Name: queueSort},
//					{Name: fakeBind},
//				},
//				Disabled: []configv1.Plugin{{Name: "*"}}, // disable default plugins
//			},
//			want: map[framework.GVK]framework.ActionType{
//				framework.Region: framework.Add,
//			},
//		},
//		{
//			name: "runner and region plugin",
//			plugins: configv1.PluginSet{
//				Enabled: []configv1.Plugin{
//					{Name: fakeRegion},
//					{Name: fakeRunner},
//					{Name: queueSort},
//					{Name: fakeBind},
//				},
//				Disabled: []configv1.Plugin{{Name: "*"}}, // disable default plugins
//			},
//			want: map[framework.GVK]framework.ActionType{
//				framework.Region: framework.Add,
//				framework.Runner: framework.Add | framework.UpdateRunnerTaint, // When Runner/Add is registered, Runner/UpdateRunnerTaint is automatically registered.
//			},
//		},
//		{
//			name: "empty EventsToRegister plugin",
//			plugins: configv1.PluginSet{
//				Enabled: []configv1.Plugin{
//					{Name: emptyEventsToRegister},
//					{Name: queueSort},
//					{Name: fakeBind},
//				},
//				Disabled: []configv1.Plugin{{Name: "*"}}, // disable default plugins
//			},
//			want: map[framework.GVK]framework.ActionType{},
//		},
//		{
//			name:    "plugins with default profile",
//			plugins: configv1.PluginSet{Enabled: defaults.PluginsV1.MultiPoint.Enabled},
//			want: map[framework.GVK]framework.ActionType{
//				framework.Region: framework.All,
//				framework.Runner: framework.All,
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			registry := plugins.NewInTreeRegistry()
//
//			cfgPls := &configv1.Plugins{MultiPoint: tt.plugins}
//			plugins := []framework.Plugin{&fakeRunnerPlugin{}, &fakeRegionPlugin{}, &filterWithoutEnqueueExtensionsPlugin{}, &emptyEventsToRegisterPlugin{}, &fakeQueueSortPlugin{}, &fakebindPlugin{}}
//			for _, pl := range plugins {
//				tmpPl := pl
//				if err := registry.Register(pl.Name(), func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//					return tmpPl, nil
//				}); err != nil {
//					t.Fatalf("fail to register filter plugin (%s)", pl.Name())
//				}
//			}
//
//			profile := configv1.SchedulerProfile{Plugins: cfgPls, PluginConfig: defaults.PluginConfigsV1}
//			fwk, err := newFramework(ctx, registry, profile)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			queueingHintsPerProfile := internalqueue.QueueingHintMapPerProfile{
//				"default": buildQueueingHintMap(fwk.EnqueueExtensions()),
//			}
//			got := unionedGVKs(queueingHintsPerProfile)
//
//			if diff := cmp.Diff(tt.want, got); diff != "" {
//				t.Errorf("Unexpected eventToPlugin map (-want,+got):%s", diff)
//			}
//		})
//	}
//}
//
//func newFramework(ctx context.Context, r frameworkruntime.Registry, profile configv1.SchedulerProfile) (framework.Framework, error) {
//	return frameworkruntime.NewFramework(ctx, r, &profile,
//		frameworkruntime.WithSnapshotSharedLister(internalcache.NewSnapshot(nil, nil)),
//		frameworkruntime.WithInformerFactory(informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 0)),
//	)
//}
//
//var _ framework.QueueSortPlugin = &fakeQueueSortPlugin{}
//
//// fakeQueueSortPlugin is a no-op implementation for QueueSort extension point.
//type fakeQueueSortPlugin struct{}
//
//func (pl *fakeQueueSortPlugin) Name() string {
//	return queueSort
//}
//
//func (pl *fakeQueueSortPlugin) Less(_, _ *framework.QueuedRegionInfo) bool {
//	return false
//}
//
//var _ framework.BindPlugin = &fakebindPlugin{}
//
//// fakebindPlugin is a no-op implementation for Bind extension point.
//type fakebindPlugin struct{}
//
//func (t *fakebindPlugin) Name() string {
//	return fakeBind
//}
//
//func (t *fakebindPlugin) Bind(ctx context.Context, state *framework.CycleState, p *corev1.Region, runnerName string) *framework.Status {
//	return nil
//}
//
//// filterWithoutEnqueueExtensionsPlugin implements Filter, but doesn't implement EnqueueExtensions.
//type filterWithoutEnqueueExtensionsPlugin struct{}
//
//func (*filterWithoutEnqueueExtensionsPlugin) Name() string { return filterWithoutEnqueueExtensions }
//
//func (*filterWithoutEnqueueExtensionsPlugin) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//var hintFromFakeRunner = framework.QueueingHint(100)
//
//type fakeRunnerPlugin struct{}
//
//var fakeRunnerPluginQueueingFn = func(_ klog.Logger, _ *corev1.Region, _, _ interface{}) (framework.QueueingHint, error) {
//	return hintFromFakeRunner, nil
//}
//
//func (*fakeRunnerPlugin) Name() string { return fakeRunner }
//
//func (*fakeRunnerPlugin) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//func (pl *fakeRunnerPlugin) EventsToRegister() []framework.ClusterEventWithHint {
//	return []framework.ClusterEventWithHint{
//		{Event: framework.ClusterEvent{Resource: framework.Runner, ActionType: framework.Add}, QueueingHintFn: fakeRunnerPluginQueueingFn},
//	}
//}
//
//var hintFromFakeRegion = framework.QueueingHint(101)
//
//type fakeRegionPlugin struct{}
//
//var fakeRegionPluginQueueingFn = func(_ klog.Logger, _ *corev1.Region, _, _ interface{}) (framework.QueueingHint, error) {
//	return hintFromFakeRegion, nil
//}
//
//func (*fakeRegionPlugin) Name() string { return fakeRegion }
//
//func (*fakeRegionPlugin) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//func (pl *fakeRegionPlugin) EventsToRegister() []framework.ClusterEventWithHint {
//	return []framework.ClusterEventWithHint{
//		{Event: framework.ClusterEvent{Resource: framework.Region, ActionType: framework.Add}, QueueingHintFn: fakeRegionPluginQueueingFn},
//	}
//}
//
//type emptyEventPlugin struct{}
//
//func (*emptyEventPlugin) Name() string { return emptyEventExtensions }
//
//func (*emptyEventPlugin) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//func (pl *emptyEventPlugin) EventsToRegister() []framework.ClusterEventWithHint {
//	return nil
//}
//
//// emptyEventsToRegisterPlugin implement interface framework.EnqueueExtensions, but returns nil from EventsToRegister.
//// This can simulate a plugin registered at scheduler setup, but does nothing
//// due to some disabled feature gate.
//type emptyEventsToRegisterPlugin struct{}
//
//func (*emptyEventsToRegisterPlugin) Name() string { return emptyEventsToRegister }
//
//func (*emptyEventsToRegisterPlugin) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//func (*emptyEventsToRegisterPlugin) EventsToRegister() []framework.ClusterEventWithHint { return nil }
