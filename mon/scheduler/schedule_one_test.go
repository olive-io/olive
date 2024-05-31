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
//	"errors"
//	"fmt"
//	"math"
//	"math/rand"
//	"reflect"
//	"regexp"
//	"sort"
//	"strconv"
//	"sync"
//	"testing"
//	"time"
//
//	"github.com/google/go-cmp/cmp"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/queuesort"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/regiontopologyspread"
//	eventsv1 "k8s.io/api/events/v1"
//	"k8s.io/apimachinery/pkg/api/resource"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/apimachinery/pkg/types"
//	"k8s.io/apimachinery/pkg/util/sets"
//	"k8s.io/apimachinery/pkg/util/wait"
//	"k8s.io/client-go/informers"
//	"k8s.io/client-go/kubernetes/scheme"
//	clienttesting "k8s.io/client-go/testing"
//	clientcache "k8s.io/client-go/tools/cache"
//	"k8s.io/client-go/tools/events"
//	"k8s.io/klog/v2"
//	"k8s.io/klog/v2/ktesting"
//	"k8s.io/utils/ptr"
//
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/runnerresources"
//	st "github.com/olive-io/olive/mon/scheduler/testing"
//	tf "github.com/olive-io/olive/mon/scheduler/testing/framework"
//
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/defaultbinder"
//
//	corev1 "github.com/olive-io/olive/apis/core/v1"
//	v1 "github.com/olive-io/olive/client-go/generated/applyconfiguration/core/v1"
//	clientsetfake "github.com/olive-io/olive/client-go/generated/clientset/versioned/fake"
//	schedulerapi "github.com/olive-io/olive/mon/scheduler/apis/config"
//	extenderv1 "github.com/olive-io/olive/mon/scheduler/extender/v1"
//	"github.com/olive-io/olive/mon/scheduler/framework"
//	"github.com/olive-io/olive/mon/scheduler/framework/plugins/feature"
//	frameworkruntime "github.com/olive-io/olive/mon/scheduler/framework/runtime"
//	internalcache "github.com/olive-io/olive/mon/scheduler/internal/cache"
//	fakecache "github.com/olive-io/olive/mon/scheduler/internal/cache/fake"
//	internalqueue "github.com/olive-io/olive/mon/scheduler/internal/queue"
//	"github.com/olive-io/olive/mon/scheduler/profile"
//	schedutil "github.com/olive-io/olive/mon/scheduler/util"
//)
//
//const (
//	testSchedulerName       = "test-scheduler"
//	mb                int64 = 1024 * 1024
//)
//
//var (
//	emptySnapshot            = internalcache.NewEmptySnapshot()
//	regionTopologySpreadFunc = frameworkruntime.FactoryAdapter(feature.Features{}, regiontopologyspread.New)
//	errPrioritize            = fmt.Errorf("priority map encounters an error")
//)
//
//type mockScheduleResult struct {
//	result ScheduleResult
//	err    error
//}
//
//type fakeExtender struct {
//	isBinder             bool
//	interestedRegionName string
//	ignorable            bool
//	gotBind              bool
//	errBind              bool
//	isPrioritizer        bool
//	isFilter             bool
//}
//
//func (f *fakeExtender) Name() string {
//	return "fakeExtender"
//}
//
//func (f *fakeExtender) IsIgnorable() bool {
//	return f.ignorable
//}
//
//func (f *fakeExtender) ProcessPreemption(
//	_ *corev1.Region,
//	_ map[string]*extenderv1.Victims,
//	_ framework.RunnerInfoLister,
//) (map[string]*extenderv1.Victims, error) {
//	return nil, nil
//}
//
//func (f *fakeExtender) SupportsPreemption() bool {
//	return false
//}
//
//func (f *fakeExtender) Filter(region *corev1.Region, runners []*framework.RunnerInfo) ([]*framework.RunnerInfo, extenderv1.FailedRunnersMap, extenderv1.FailedRunnersMap, error) {
//	return nil, nil, nil, nil
//}
//
//func (f *fakeExtender) Prioritize(
//	_ *corev1.Region,
//	_ []*framework.RunnerInfo,
//) (hostPriorities *extenderv1.HostPriorityList, weight int64, err error) {
//	return nil, 0, nil
//}
//
//func (f *fakeExtender) Bind(binding *v1.Binding) error {
//	if f.isBinder {
//		if f.errBind {
//			return errors.New("bind error")
//		}
//		f.gotBind = true
//		return nil
//	}
//	return errors.New("not a binder")
//}
//
//func (f *fakeExtender) IsBinder() bool {
//	return f.isBinder
//}
//
//func (f *fakeExtender) IsInterested(region *corev1.Region) bool {
//	return region != nil && region.Name == f.interestedRegionName
//}
//
//func (f *fakeExtender) IsPrioritizer() bool {
//	return f.isPrioritizer
//}
//
//func (f *fakeExtender) IsFilter() bool {
//	return f.isFilter
//}
//
//type falseMapPlugin struct{}
//
//func newFalseMapPlugin() frameworkruntime.PluginFactory {
//	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//		return &falseMapPlugin{}, nil
//	}
//}
//
//func (pl *falseMapPlugin) Name() string {
//	return "FalseMap"
//}
//
//func (pl *falseMapPlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) (int64, *framework.Status) {
//	return 0, framework.AsStatus(errPrioritize)
//}
//
//func (pl *falseMapPlugin) ScoreExtensions() framework.ScoreExtensions {
//	return nil
//}
//
//type numericMapPlugin struct{}
//
//func newNumericMapPlugin() frameworkruntime.PluginFactory {
//	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//		return &numericMapPlugin{}, nil
//	}
//}
//
//func (pl *numericMapPlugin) Name() string {
//	return "NumericMap"
//}
//
//func (pl *numericMapPlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerName string) (int64, *framework.Status) {
//	score, err := strconv.Atoi(runnerName)
//	if err != nil {
//		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("Error converting runnername to int: %+v", runnerName))
//	}
//	return int64(score), nil
//}
//
//func (pl *numericMapPlugin) ScoreExtensions() framework.ScoreExtensions {
//	return nil
//}
//
//// NewNoRegionsFilterPlugin initializes a noRegionsFilterPlugin and returns it.
//func NewNoRegionsFilterPlugin(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//	return &noRegionsFilterPlugin{}, nil
//}
//
//type reverseNumericMapPlugin struct{}
//
//func (pl *reverseNumericMapPlugin) Name() string {
//	return "ReverseNumericMap"
//}
//
//func (pl *reverseNumericMapPlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerName string) (int64, *framework.Status) {
//	score, err := strconv.Atoi(runnerName)
//	if err != nil {
//		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("Error converting runnername to int: %+v", runnerName))
//	}
//	return int64(score), nil
//}
//
//func (pl *reverseNumericMapPlugin) ScoreExtensions() framework.ScoreExtensions {
//	return pl
//}
//
//func (pl *reverseNumericMapPlugin) NormalizeScore(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerScores framework.RunnerScoreList) *framework.Status {
//	var maxScore float64
//	minScore := math.MaxFloat64
//
//	for _, hostPriority := range runnerScores {
//		maxScore = math.Max(maxScore, float64(hostPriority.Score))
//		minScore = math.Min(minScore, float64(hostPriority.Score))
//	}
//	for i, hostPriority := range runnerScores {
//		runnerScores[i] = framework.RunnerScore{
//			Name:  hostPriority.Name,
//			Score: int64(maxScore + minScore - float64(hostPriority.Score)),
//		}
//	}
//	return nil
//}
//
//func newReverseNumericMapPlugin() frameworkruntime.PluginFactory {
//	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//		return &reverseNumericMapPlugin{}, nil
//	}
//}
//
//type trueMapPlugin struct{}
//
//func (pl *trueMapPlugin) Name() string {
//	return "TrueMap"
//}
//
//func (pl *trueMapPlugin) Score(_ context.Context, _ *framework.CycleState, _ *corev1.Region, _ string) (int64, *framework.Status) {
//	return 1, nil
//}
//
//func (pl *trueMapPlugin) ScoreExtensions() framework.ScoreExtensions {
//	return pl
//}
//
//func (pl *trueMapPlugin) NormalizeScore(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerScores framework.RunnerScoreList) *framework.Status {
//	for _, host := range runnerScores {
//		if host.Name == "" {
//			return framework.NewStatus(framework.Error, "unexpected empty host name")
//		}
//	}
//	return nil
//}
//
//func newTrueMapPlugin() frameworkruntime.PluginFactory {
//	return func(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//		return &trueMapPlugin{}, nil
//	}
//}
//
//type noRegionsFilterPlugin struct{}
//
//// Name returns name of the plugin.
//func (pl *noRegionsFilterPlugin) Name() string {
//	return "NoRegionsFilter"
//}
//
//// Filter invoked at the filter extension point.
//func (pl *noRegionsFilterPlugin) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
//	if len(runnerInfo.Regions) == 0 {
//		return nil
//	}
//	return framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake)
//}
//
//type fakeRunnerSelectorArgs struct {
//	RunnerName string `json:"runnerName"`
//}
//
//type fakeRunnerSelector struct {
//	fakeRunnerSelectorArgs
//}
//
//func (s *fakeRunnerSelector) Name() string {
//	return "FakeRunnerSelector"
//}
//
//func (s *fakeRunnerSelector) Filter(_ context.Context, _ *framework.CycleState, _ *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
//	if runnerInfo.Runner().Name != s.RunnerName {
//		return framework.NewStatus(framework.UnschedulableAndUnresolvable)
//	}
//	return nil
//}
//
//func newFakeRunnerSelector(_ context.Context, args runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//	pl := &fakeRunnerSelector{}
//	if err := frameworkruntime.DecodeInto(args, &pl.fakeRunnerSelectorArgs); err != nil {
//		return nil, err
//	}
//	return pl, nil
//}
//
//const (
//	fakeSpecifiedRunnerNameAnnotation = "fake-specified-runner-name"
//)
//
//// fakeRunnerSelectorDependOnRegionAnnotation schedules region to the specified one runner from region.Annotations[fakeSpecifiedRunnerNameAnnotation].
//type fakeRunnerSelectorDependOnRegionAnnotation struct{}
//
//func (f *fakeRunnerSelectorDependOnRegionAnnotation) Name() string {
//	return "FakeRunnerSelectorDependOnRegionAnnotation"
//}
//
//// Filter selects the specified one runner and rejects other non-specified runners.
//func (f *fakeRunnerSelectorDependOnRegionAnnotation) Filter(_ context.Context, _ *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
//	resolveRunnerNameFromRegionAnnotation := func(region *corev1.Region) (string, error) {
//		if region == nil {
//			return "", fmt.Errorf("empty region")
//		}
//		runnerName, ok := region.Annotations[fakeSpecifiedRunnerNameAnnotation]
//		if !ok {
//			return "", fmt.Errorf("no specified runner name on region %s/%s annotation", region.Namespace, region.Name)
//		}
//		return runnerName, nil
//	}
//
//	runnerName, err := resolveRunnerNameFromRegionAnnotation(region)
//	if err != nil {
//		return framework.AsStatus(err)
//	}
//	if runnerInfo.Runner().Name != runnerName {
//		return framework.NewStatus(framework.UnschedulableAndUnresolvable)
//	}
//	return nil
//}
//
//func newFakeRunnerSelectorDependOnRegionAnnotation(_ context.Context, _ runtime.Object, _ framework.Handle) (framework.Plugin, error) {
//	return &fakeRunnerSelectorDependOnRegionAnnotation{}, nil
//}
//
//type TestPlugin struct {
//	name string
//}
//
//var _ framework.ScorePlugin = &TestPlugin{}
//var _ framework.FilterPlugin = &TestPlugin{}
//
//func (t *TestPlugin) Name() string {
//	return t.name
//}
//
//func (t *TestPlugin) Score(ctx context.Context, state *framework.CycleState, p *corev1.Region, runnerName string) (int64, *framework.Status) {
//	return 1, nil
//}
//
//func (t *TestPlugin) ScoreExtensions() framework.ScoreExtensions {
//	return nil
//}
//
//func (t *TestPlugin) Filter(ctx context.Context, state *framework.CycleState, region *corev1.Region, runnerInfo *framework.RunnerInfo) *framework.Status {
//	return nil
//}
//
//func TestSchedulerMultipleProfilesScheduling(t *testing.T) {
//	runners := []runtime.Object{
//		st.MakeRunner().Name("runner1").UID("runner1").Obj(),
//		st.MakeRunner().Name("runner2").UID("runner2").Obj(),
//		st.MakeRunner().Name("runner3").UID("runner3").Obj(),
//	}
//	regions := []*corev1.Region{
//		st.MakeRegion().Name("region1").UID("region1").SchedulerName("match-runner3").Obj(),
//		st.MakeRegion().Name("region2").UID("region2").SchedulerName("match-runner2").Obj(),
//		st.MakeRegion().Name("region3").UID("region3").SchedulerName("match-runner2").Obj(),
//		st.MakeRegion().Name("region4").UID("region4").SchedulerName("match-runner3").Obj(),
//	}
//	wantBindings := map[string]string{
//		"region1": "runner3",
//		"region2": "runner2",
//		"region3": "runner2",
//		"region4": "runner3",
//	}
//	wantControllers := map[string]string{
//		"region1": "match-runner3",
//		"region2": "match-runner2",
//		"region3": "match-runner2",
//		"region4": "match-runner3",
//	}
//
//	// Set up scheduler for the 3 runners.
//	// We use a fake filter that only allows one particular runner. We create two
//	// profiles, each with a different runner in the filter configuration.
//	objs := append([]runtime.Object{
//		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ""}}}, runners...)
//	client := clientsetfake.NewSimpleClientset(objs...)
//	broadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	informerFactory := informers.NewSharedInformerFactory(client, 0)
//	sched, err := New(
//		ctx,
//		client,
//		informerFactory,
//		nil,
//		profile.NewRecorderFactory(broadcaster),
//		WithProfiles(
//			schedulerapi.SchedulerProfile{SchedulerName: "match-runner2",
//				Plugins: &schedulerapi.Plugins{
//					Filter:    schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "FakeRunnerSelector"}}},
//					QueueSort: schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "PrioritySort"}}},
//					Bind:      schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "DefaultBinder"}}},
//				},
//				PluginConfig: []schedulerapi.PluginConfig{
//					{
//						Name: "FakeRunnerSelector",
//						Args: &runtime.Unknown{Raw: []byte(`{"runnerName":"runner2"}`)},
//					},
//				},
//			},
//			schedulerapi.SchedulerProfile{
//				SchedulerName: "match-runner3",
//				Plugins: &schedulerapi.Plugins{
//					Filter:    schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "FakeRunnerSelector"}}},
//					QueueSort: schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "PrioritySort"}}},
//					Bind:      schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "DefaultBinder"}}},
//				},
//				PluginConfig: []schedulerapi.PluginConfig{
//					{
//						Name: "FakeRunnerSelector",
//						Args: &runtime.Unknown{Raw: []byte(`{"runnerName":"runner3"}`)},
//					},
//				},
//			},
//		),
//		WithFrameworkOutOfTreeRegistry(frameworkruntime.Registry{
//			"FakeRunnerSelector": newFakeRunnerSelector,
//		}),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// Capture the bindings and events' controllers.
//	var wg sync.WaitGroup
//	wg.Add(2 * len(regions))
//	bindings := make(map[string]string)
//	client.PrependReactor("create", "regions", func(action clienttesting.Action) (bool, runtime.Object, error) {
//		if action.GetSubresource() != "binding" {
//			return false, nil, nil
//		}
//		binding := action.(clienttesting.CreateAction).GetObject().(*v1.Binding)
//		bindings[binding.Name] = binding.Target.Name
//		wg.Done()
//		return true, binding, nil
//	})
//	controllers := make(map[string]string)
//	stopFn, err := broadcaster.StartEventWatcher(func(obj runtime.Object) {
//		e, ok := obj.(*eventsv1.Event)
//		if !ok || e.Reason != "Scheduled" {
//			return
//		}
//		controllers[e.Regarding.Name] = e.ReportingController
//		wg.Done()
//	})
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer stopFn()
//
//	// Run scheduler.
//	informerFactory.Start(ctx.Done())
//	informerFactory.WaitForCacheSync(ctx.Done())
//	if err = sched.WaitForHandlersSync(ctx); err != nil {
//		t.Fatalf("Handlers failed to sync: %v: ", err)
//	}
//	go sched.Run(ctx)
//
//	// Send regions to be scheduled.
//	for _, p := range regions {
//		_, err := client.CoreV1().Regions("").Create(ctx, p, metav1.CreateOptions{})
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//	wg.Wait()
//
//	// Verify correct bindings and reporting controllers.
//	if diff := cmp.Diff(wantBindings, bindings); diff != "" {
//		t.Errorf("regions were scheduled incorrectly (-want, +got):\n%s", diff)
//	}
//	if diff := cmp.Diff(wantControllers, controllers); diff != "" {
//		t.Errorf("events were reported with wrong controllers (-want, +got):\n%s", diff)
//	}
//}
//
//// TestSchedulerGuaranteeNonNilRunnerInSchedulingCycle is for detecting potential panic on nil Runner when iterating Runners.
//func TestSchedulerGuaranteeNonNilRunnerInSchedulingCycle(t *testing.T) {
//	random := rand.New(rand.NewSource(time.Now().UnixNano()))
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	var (
//		initialRunnerNumber        = 1000
//		initialRegionNumber        = 500
//		waitSchedulingRegionNumber = 200
//		deleteRunnerNumberPerRound = 20
//		createRegionNumberPerRound = 50
//
//		fakeSchedulerName = "fake-scheduler"
//		fakeNamespace     = "fake-namespace"
//
//		initialRunners []runtime.Object
//		initialRegions []runtime.Object
//	)
//
//	for i := 0; i < initialRunnerNumber; i++ {
//		runnerName := fmt.Sprintf("runner%d", i)
//		initialRunners = append(initialRunners, st.MakeRunner().Name(runnerName).UID(runnerName).Obj())
//	}
//	// Randomly scatter initial regions onto runners.
//	for i := 0; i < initialRegionNumber; i++ {
//		regionName := fmt.Sprintf("scheduled-region%d", i)
//		assignedRunnerName := fmt.Sprintf("runner%d", random.Intn(initialRunnerNumber))
//		initialRegions = append(initialRegions, st.MakeRegion().Name(regionName).UID(regionName).Runner(assignedRunnerName).Obj())
//	}
//
//	objs := []runtime.Object{&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: fakeNamespace}}}
//	objs = append(objs, initialRunners...)
//	objs = append(objs, initialRegions...)
//	client := clientsetfake.NewSimpleClientset(objs...)
//	broadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//
//	informerFactory := informers.NewSharedInformerFactory(client, 0)
//	sched, err := New(
//		ctx,
//		client,
//		informerFactory,
//		nil,
//		profile.NewRecorderFactory(broadcaster),
//		WithProfiles(
//			schedulerapi.KubeSchedulerProfile{SchedulerName: fakeSchedulerName,
//				Plugins: &schedulerapi.Plugins{
//					Filter:    schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "FakeRunnerSelectorDependOnRegionAnnotation"}}},
//					QueueSort: schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "PrioritySort"}}},
//					Bind:      schedulerapi.PluginSet{Enabled: []schedulerapi.Plugin{{Name: "DefaultBinder"}}},
//				},
//			},
//		),
//		WithFrameworkOutOfTreeRegistry(frameworkruntime.Registry{
//			"FakeRunnerSelectorDependOnRegionAnnotation": newFakeRunnerSelectorDependOnRegionAnnotation,
//		}),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// Run scheduler.
//	informerFactory.Start(ctx.Done())
//	informerFactory.WaitForCacheSync(ctx.Done())
//	go sched.Run(ctx)
//
//	var deleteRunnerIndex int
//	deleteRunnersOneRound := func() {
//		for i := 0; i < deleteRunnerNumberPerRound; i++ {
//			if deleteRunnerIndex >= initialRunnerNumber {
//				// all initial runners are already deleted
//				return
//			}
//			deleteRunnerName := fmt.Sprintf("runner%d", deleteRunnerIndex)
//			if err := client.CoreV1().Runners().Delete(ctx, deleteRunnerName, metav1.DeleteOptions{}); err != nil {
//				t.Fatal(err)
//			}
//			deleteRunnerIndex++
//		}
//	}
//	var createRegionIndex int
//	createRegionsOneRound := func() {
//		if createRegionIndex > waitSchedulingRegionNumber {
//			return
//		}
//		for i := 0; i < createRegionNumberPerRound; i++ {
//			regionName := fmt.Sprintf("region%d", createRegionIndex)
//			// Note: the runner(specifiedRunnerName) may already be deleted, which leads region scheduled failed.
//			specifiedRunnerName := fmt.Sprintf("runner%d", random.Intn(initialRunnerNumber))
//
//			waitSchedulingRegion := st.MakeRegion().Namespace(fakeNamespace).Name(regionName).UID(regionName).Annotation(fakeSpecifiedRunnerNameAnnotation, specifiedRunnerName).SchedulerName(fakeSchedulerName).Obj()
//			if _, err := client.CoreV1().Regions(fakeNamespace).Create(ctx, waitSchedulingRegion, metav1.CreateOptions{}); err != nil {
//				t.Fatal(err)
//			}
//			createRegionIndex++
//		}
//	}
//
//	// Following we start 2 goroutines asynchronously to detect potential racing issues:
//	// 1) One is responsible for deleting several runners in each round;
//	// 2) Another is creating several regions in each round to trigger scheduling;
//	// Those two goroutines will stop until ctx.Done() is called, which means all waiting regions are scheduled at least once.
//	go wait.Until(deleteRunnersOneRound, 10*time.Millisecond, ctx.Done())
//	go wait.Until(createRegionsOneRound, 9*time.Millisecond, ctx.Done())
//
//	// Capture the events to wait all regions to be scheduled at least once.
//	allWaitSchedulingRegions := sets.New[string]()
//	for i := 0; i < waitSchedulingRegionNumber; i++ {
//		allWaitSchedulingRegions.Insert(fmt.Sprintf("region%d", i))
//	}
//	var wg sync.WaitGroup
//	wg.Add(waitSchedulingRegionNumber)
//	stopFn, err := broadcaster.StartEventWatcher(func(obj runtime.Object) {
//		e, ok := obj.(*eventsv1.Event)
//		if !ok || (e.Reason != "Scheduled" && e.Reason != "FailedScheduling") {
//			return
//		}
//		if allWaitSchedulingRegions.Has(e.Regarding.Name) {
//			wg.Done()
//			allWaitSchedulingRegions.Delete(e.Regarding.Name)
//		}
//	})
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer stopFn()
//
//	wg.Wait()
//}
//
//func TestSchedulerScheduleOne(t *testing.T) {
//	testRunner := corev1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "runner1", UID: types.UID("runner1")}}
//	client := clientsetfake.NewSimpleClientset(&testRunner)
//	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: client.EventsV1()})
//	errS := errors.New("scheduler")
//	errB := errors.New("binder")
//	preBindErr := errors.New("on PreBind")
//
//	table := []struct {
//		name                string
//		injectBindError     error
//		sendRegion          *corev1.Region
//		registerPluginFuncs []tf.RegisterPluginFunc
//		expectErrorRegion   *corev1.Region
//		expectForgetRegion  *corev1.Region
//		expectAssumedRegion *corev1.Region
//		expectError         error
//		expectBind          *corev1.Binding
//		eventReason         string
//		mockResult          mockScheduleResult
//	}{
//		{
//			name:       "error reserve region",
//			sendRegion: regionWithID("foo", ""),
//			mockResult: mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, nil},
//			registerPluginFuncs: []tf.RegisterPluginFunc{
//				tf.RegisterReservePlugin("FakeReserve", tf.NewFakeReservePlugin(framework.NewStatus(framework.Error, "reserve error"))),
//			},
//			expectErrorRegion:   regionWithID("foo", testRunner.Name),
//			expectForgetRegion:  regionWithID("foo", testRunner.Name),
//			expectAssumedRegion: regionWithID("foo", testRunner.Name),
//			expectError:         fmt.Errorf(`running Reserve plugin "FakeReserve": %w`, errors.New("reserve error")),
//			eventReason:         "FailedScheduling",
//		},
//		{
//			name:       "error permit region",
//			sendRegion: regionWithID("foo", ""),
//			mockResult: mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, nil},
//			registerPluginFuncs: []tf.RegisterPluginFunc{
//				tf.RegisterPermitPlugin("FakePermit", tf.NewFakePermitPlugin(framework.NewStatus(framework.Error, "permit error"), time.Minute)),
//			},
//			expectErrorRegion:   regionWithID("foo", testRunner.Name),
//			expectForgetRegion:  regionWithID("foo", testRunner.Name),
//			expectAssumedRegion: regionWithID("foo", testRunner.Name),
//			expectError:         fmt.Errorf(`running Permit plugin "FakePermit": %w`, errors.New("permit error")),
//			eventReason:         "FailedScheduling",
//		},
//		{
//			name:       "error prebind region",
//			sendRegion: regionWithID("foo", ""),
//			mockResult: mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, nil},
//			registerPluginFuncs: []tf.RegisterPluginFunc{
//				tf.RegisterPreBindPlugin("FakePreBind", tf.NewFakePreBindPlugin(framework.AsStatus(preBindErr))),
//			},
//			expectErrorRegion:   regionWithID("foo", testRunner.Name),
//			expectForgetRegion:  regionWithID("foo", testRunner.Name),
//			expectAssumedRegion: regionWithID("foo", testRunner.Name),
//			expectError:         fmt.Errorf(`running PreBind plugin "FakePreBind": %w`, preBindErr),
//			eventReason:         "FailedScheduling",
//		},
//		{
//			name:                "bind assumed region scheduled",
//			sendRegion:          regionWithID("foo", ""),
//			mockResult:          mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, nil},
//			expectBind:          &corev1.Binding{ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: types.UID("foo")}, Target: v1.ObjectReference{Kind: "Runner", Name: testRunner.Name}},
//			expectAssumedRegion: regionWithID("foo", testRunner.Name),
//			eventReason:         "Scheduled",
//		},
//		{
//			name:              "error region failed scheduling",
//			sendRegion:        regionWithID("foo", ""),
//			mockResult:        mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, errS},
//			expectError:       errS,
//			expectErrorRegion: regionWithID("foo", ""),
//			eventReason:       "FailedScheduling",
//		},
//		{
//			name:                "error bind forget region failed scheduling",
//			sendRegion:          regionWithID("foo", ""),
//			mockResult:          mockScheduleResult{ScheduleResult{SuggestedHost: testRunner.Name, EvaluatedRunners: 1, FeasibleRunners: 1}, nil},
//			expectBind:          &corev1.Binding{ObjectMeta: metav1.ObjectMeta{Name: "foo", UID: types.UID("foo")}, Target: corev1.ObjectReference{Kind: "Runner", Names: []string{testRunner.Name}}},
//			expectAssumedRegion: regionWithID("foo", testRunner.Name),
//			injectBindError:     errB,
//			expectError:         fmt.Errorf("running Bind plugin %q: %w", "DefaultBinder", errors.New("binder")),
//			expectErrorRegion:   regionWithID("foo", testRunner.Name),
//			expectForgetRegion:  regionWithID("foo", testRunner.Name),
//			eventReason:         "FailedScheduling",
//		},
//		{
//			name:        "deleting region",
//			sendRegion:  deletingRegion("foo"),
//			mockResult:  mockScheduleResult{ScheduleResult{}, nil},
//			eventReason: "FailedScheduling",
//		},
//	}
//
//	for _, item := range table {
//		t.Run(item.name, func(t *testing.T) {
//			var gotError error
//			var gotRegion *corev1.Region
//			var gotForgetRegion *corev1.Region
//			var gotAssumedRegion *corev1.Region
//			var gotBinding *v1.Binding
//			cache := &fakecache.Cache{
//				ForgetFunc: func(region *corev1.Region) {
//					gotForgetRegion = region
//				},
//				AssumeFunc: func(region *corev1.Region) {
//					gotAssumedRegion = region
//				},
//				IsAssumedRegionFunc: func(region *corev1.Region) bool {
//					if region == nil || gotAssumedRegion == nil {
//						return false
//					}
//					return region.UID == gotAssumedRegion.UID
//				},
//			}
//			client := clientsetfake.NewSimpleClientset(item.sendRegion)
//			client.PrependReactor("create", "regions", func(action clienttesting.Action) (bool, runtime.Object, error) {
//				if action.GetSubresource() != "binding" {
//					return false, nil, nil
//				}
//				gotBinding = action.(clienttesting.CreateAction).GetObject().(*corev1.Binding)
//				return true, gotBinding, item.injectBindError
//			})
//			registerPluginFuncs := append(item.registerPluginFuncs,
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			)
//			ctx, cancel := context.WithCancel(context.Background())
//			defer cancel()
//			fwk, err := tf.NewFramework(ctx,
//				registerPluginFuncs,
//				testSchedulerName,
//				frameworkruntime.WithClientSet(client),
//				frameworkruntime.WithEventRecorder(eventBroadcaster.NewRecorder(scheme.Scheme, testSchedulerName)))
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			sched := &Scheduler{
//				Cache:  cache,
//				client: client,
//				NextRegion: func(logger klog.Logger) (*framework.QueuedRegionInfo, error) {
//					return &framework.QueuedRegionInfo{RegionInfo: mustNewRegionInfo(t, item.sendRegion)}, nil
//				},
//				SchedulingQueue: internalqueue.NewTestQueue(ctx, nil),
//				Profiles:        profile.Map{testSchedulerName: fwk},
//			}
//
//			sched.ScheduleRegion = func(ctx context.Context, fwk framework.Framework, state *framework.CycleState, region *corev1.Region) (ScheduleResult, error) {
//				return item.mockResult.result, item.mockResult.err
//			}
//			sched.FailureHandler = func(_ context.Context, fwk framework.Framework, p *framework.QueuedRegionInfo, status *framework.Status, _ *framework.NominatingInfo, _ time.Time) {
//				gotRegion = p.Region
//				gotError = status.AsError()
//
//				msg := truncateMessage(gotError.Error())
//				fwk.EventRecorder().Eventf(p.Region, nil, v1.EventTypeWarning, "FailedScheduling", "Scheduling", msg)
//			}
//			called := make(chan struct{})
//			stopFunc, err := eventBroadcaster.StartEventWatcher(func(obj runtime.Object) {
//				e, _ := obj.(*eventsv1.Event)
//				if e.Reason != item.eventReason {
//					t.Errorf("got event %v, want %v", e.Reason, item.eventReason)
//				}
//				close(called)
//			})
//			if err != nil {
//				t.Fatal(err)
//			}
//			sched.ScheduleOne(ctx)
//			<-called
//			if e, a := item.expectAssumedRegion, gotAssumedRegion; !reflect.DeepEqual(e, a) {
//				t.Errorf("assumed region: wanted %v, got %v", e, a)
//			}
//			if e, a := item.expectErrorRegion, gotRegion; !reflect.DeepEqual(e, a) {
//				t.Errorf("error region: wanted %v, got %v", e, a)
//			}
//			if e, a := item.expectForgetRegion, gotForgetRegion; !reflect.DeepEqual(e, a) {
//				t.Errorf("forget region: wanted %v, got %v", e, a)
//			}
//			if e, a := item.expectError, gotError; !reflect.DeepEqual(e, a) {
//				t.Errorf("error: wanted %v, got %v", e, a)
//			}
//			if diff := cmp.Diff(item.expectBind, gotBinding); diff != "" {
//				t.Errorf("got binding diff (-want, +got): %s", diff)
//			}
//			stopFunc()
//		})
//	}
//}
//
//func TestSchedulerNoPhantomRegionAfterExpire(t *testing.T) {
//	logger, ctx := ktesting.NewTestContext(t)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	queuedRegionStore := clientcache.NewFIFO(clientcache.MetaNamespaceKeyFunc)
//	scache := internalcache.New(ctx, 100*time.Millisecond)
//	region := regionWithPort("region.Name", "", 8080)
//	runner := v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "runner1", UID: types.UID("runner1")}}
//	scache.AddRunner(logger, &runner)
//
//	fns := []tf.RegisterPluginFunc{
//		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		tf.RegisterPluginAsExtensions(runnerports.Name, runnerports.New, "Filter", "PreFilter"),
//	}
//	scheduler, bindingChan, errChan := setupTestSchedulerWithOneRegionOnRunner(ctx, t, queuedRegionStore, scache, region, &runner, fns...)
//
//	waitRegionExpireChan := make(chan struct{})
//	timeout := make(chan struct{})
//	go func() {
//		for {
//			select {
//			case <-timeout:
//				return
//			default:
//			}
//			regions, err := scache.RegionCount()
//			if err != nil {
//				errChan <- fmt.Errorf("cache.List failed: %v", err)
//				return
//			}
//			if regions == 0 {
//				close(waitRegionExpireChan)
//				return
//			}
//			time.Sleep(100 * time.Millisecond)
//		}
//	}()
//	// waiting for the assumed region to expire
//	select {
//	case err := <-errChan:
//		t.Fatal(err)
//	case <-waitRegionExpireChan:
//	case <-time.After(wait.ForeverTestTimeout):
//		close(timeout)
//		t.Fatalf("timeout timeout in waiting region expire after %v", wait.ForeverTestTimeout)
//	}
//
//	// We use conflicted region ports to incur fit predicate failure if first region not removed.
//	secondRegion := regionWithPort("bar", "", 8080)
//	queuedRegionStore.Add(secondRegion)
//	scheduler.ScheduleOne(ctx)
//	select {
//	case b := <-bindingChan:
//		expectBinding := &v1.Binding{
//			ObjectMeta: metav1.ObjectMeta{Name: "bar", UID: types.UID("bar")},
//			Target:     v1.ObjectReference{Kind: "Runner", Name: runner.Name},
//		}
//		if !reflect.DeepEqual(expectBinding, b) {
//			t.Errorf("binding want=%v, get=%v", expectBinding, b)
//		}
//	case <-time.After(wait.ForeverTestTimeout):
//		t.Fatalf("timeout in binding after %v", wait.ForeverTestTimeout)
//	}
//}
//
//func TestSchedulerNoPhantomRegionAfterDelete(t *testing.T) {
//	logger, ctx := ktesting.NewTestContext(t)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	queuedRegionStore := clientcache.NewFIFO(clientcache.MetaNamespaceKeyFunc)
//	scache := internalcache.New(ctx, 10*time.Minute)
//	firstRegion := regionWithPort("region.Name", "", 8080)
//	runner := v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "runner1", UID: types.UID("runner1")}}
//	scache.AddRunner(logger, &runner)
//	fns := []tf.RegisterPluginFunc{
//		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		tf.RegisterPluginAsExtensions(runnerports.Name, runnerports.New, "Filter", "PreFilter"),
//	}
//	scheduler, bindingChan, errChan := setupTestSchedulerWithOneRegionOnRunner(ctx, t, queuedRegionStore, scache, firstRegion, &runner, fns...)
//
//	// We use conflicted region ports to incur fit predicate failure.
//	secondRegion := regionWithPort("bar", "", 8080)
//	queuedRegionStore.Add(secondRegion)
//	// queuedRegionStore: [bar:8080]
//	// cache: [(assumed)foo:8080]
//
//	scheduler.ScheduleOne(ctx)
//	select {
//	case err := <-errChan:
//		expectErr := &framework.FitError{
//			Region:        secondRegion,
//			NumAllRunners: 1,
//			Diagnosis: framework.Diagnosis{
//				RunnerToStatusMap: framework.RunnerToStatusMap{
//					runner.Name: framework.NewStatus(framework.Unschedulable, runnerports.ErrReason).WithPlugin(runnerports.Name),
//				},
//				UnschedulablePlugins: sets.New(runnerports.Name),
//			},
//		}
//		if !reflect.DeepEqual(expectErr, err) {
//			t.Errorf("err want=%v, get=%v", expectErr, err)
//		}
//	case <-time.After(wait.ForeverTestTimeout):
//		t.Fatalf("timeout in fitting after %v", wait.ForeverTestTimeout)
//	}
//
//	// We mimic the workflow of cache behavior when a region is removed by user.
//	// Note: if the schedulerrunnerinfo timeout would be super short, the first region would expire
//	// and would be removed itself (without any explicit actions on schedulerrunnerinfo). Even in that case,
//	// explicitly AddRegion will as well correct the behavior.
//	firstRegion.Spec.RunnerName = runner.Name
//	if err := scache.AddRegion(logger, firstRegion); err != nil {
//		t.Fatalf("err: %v", err)
//	}
//	if err := scache.RemoveRegion(logger, firstRegion); err != nil {
//		t.Fatalf("err: %v", err)
//	}
//
//	queuedRegionStore.Add(secondRegion)
//	scheduler.ScheduleOne(ctx)
//	select {
//	case b := <-bindingChan:
//		expectBinding := &v1.Binding{
//			ObjectMeta: metav1.ObjectMeta{Name: "bar", UID: types.UID("bar")},
//			Target:     v1.ObjectReference{Kind: "Runner", Name: runner.Name},
//		}
//		if !reflect.DeepEqual(expectBinding, b) {
//			t.Errorf("binding want=%v, get=%v", expectBinding, b)
//		}
//	case <-time.After(wait.ForeverTestTimeout):
//		t.Fatalf("timeout in binding after %v", wait.ForeverTestTimeout)
//	}
//}
//
//func TestSchedulerFailedSchedulingReasons(t *testing.T) {
//	logger, ctx := ktesting.NewTestContext(t)
//	ctx, cancel := context.WithCancel(ctx)
//	defer cancel()
//	queuedRegionStore := clientcache.NewFIFO(clientcache.MetaNamespaceKeyFunc)
//	scache := internalcache.New(ctx, 10*time.Minute)
//
//	// Design the baseline for the regions, and we will make runners that don't fit it later.
//	var cpu = int64(4)
//	var mem = int64(500)
//	regionWithTooBigResourceRequests := regionWithResources("bar", "", v1.ResourceList{
//		v1.ResourceCPU:    *(resource.NewQuantity(cpu, resource.DecimalSI)),
//		v1.ResourceMemory: *(resource.NewQuantity(mem, resource.DecimalSI)),
//	}, v1.ResourceList{
//		v1.ResourceCPU:    *(resource.NewQuantity(cpu, resource.DecimalSI)),
//		v1.ResourceMemory: *(resource.NewQuantity(mem, resource.DecimalSI)),
//	})
//
//	// create several runners which cannot schedule the above region
//	var runners []*v1.Runner
//	var objects []runtime.Object
//	for i := 0; i < 100; i++ {
//		uid := fmt.Sprintf("runner%v", i)
//		runner := v1.Runner{
//			ObjectMeta: metav1.ObjectMeta{Name: uid, UID: types.UID(uid)},
//			Status: v1.RunnerStatus{
//				Capacity: v1.ResourceList{
//					v1.ResourceCPU:     *(resource.NewQuantity(cpu/2, resource.DecimalSI)),
//					v1.ResourceMemory:  *(resource.NewQuantity(mem/5, resource.DecimalSI)),
//					v1.ResourceRegions: *(resource.NewQuantity(10, resource.DecimalSI)),
//				},
//				Allocatable: v1.ResourceList{
//					v1.ResourceCPU:     *(resource.NewQuantity(cpu/2, resource.DecimalSI)),
//					v1.ResourceMemory:  *(resource.NewQuantity(mem/5, resource.DecimalSI)),
//					v1.ResourceRegions: *(resource.NewQuantity(10, resource.DecimalSI)),
//				}},
//		}
//		scache.AddRunner(logger, &runner)
//		runners = append(runners, &runner)
//		objects = append(objects, &runner)
//	}
//
//	// Create expected failure reasons for all the runners. Hopefully they will get rolled up into a non-spammy summary.
//	failedRunnerStatues := framework.RunnerToStatusMap{}
//	for _, runner := range runners {
//		failedRunnerStatues[runner.Name] = framework.NewStatus(
//			framework.Unschedulable,
//			fmt.Sprintf("Insufficient %v", v1.ResourceCPU),
//			fmt.Sprintf("Insufficient %v", v1.ResourceMemory),
//		).WithPlugin(runnerresources.Name)
//	}
//	fns := []tf.RegisterPluginFunc{
//		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		tf.RegisterPluginAsExtensions(runnerresources.Name, frameworkruntime.FactoryAdapter(feature.Features{}, runnerresources.NewFit), "Filter", "PreFilter"),
//	}
//
//	informerFactory := informers.NewSharedInformerFactory(clientsetfake.NewSimpleClientset(objects...), 0)
//	scheduler, _, errChan := setupTestScheduler(ctx, t, queuedRegionStore, scache, informerFactory, nil, fns...)
//
//	queuedRegionStore.Add(regionWithTooBigResourceRequests)
//	scheduler.ScheduleOne(ctx)
//	select {
//	case err := <-errChan:
//		expectErr := &framework.FitError{
//			Region:        regionWithTooBigResourceRequests,
//			NumAllRunners: len(runners),
//			Diagnosis: framework.Diagnosis{
//				RunnerToStatusMap:    failedRunnerStatues,
//				UnschedulablePlugins: sets.New(runnerresources.Name),
//			},
//		}
//		if len(fmt.Sprint(expectErr)) > 150 {
//			t.Errorf("message is too spammy ! %v ", len(fmt.Sprint(expectErr)))
//		}
//		if !reflect.DeepEqual(expectErr, err) {
//			t.Errorf("\n err \nWANT=%+v,\nGOT=%+v", expectErr, err)
//		}
//	case <-time.After(wait.ForeverTestTimeout):
//		t.Fatalf("timeout after %v", wait.ForeverTestTimeout)
//	}
//}
//
//func TestSchedulerBinding(t *testing.T) {
//	table := []struct {
//		regionName   string
//		extenders    []framework.Extender
//		wantBinderID int
//		name         string
//	}{
//		{
//			name:       "the extender is not a binder",
//			regionName: "region0",
//			extenders: []framework.Extender{
//				&fakeExtender{isBinder: false, interestedRegionName: "region0"},
//			},
//			wantBinderID: -1, // default binding.
//		},
//		{
//			name:       "one of the extenders is a binder and interested in region",
//			regionName: "region0",
//			extenders: []framework.Extender{
//				&fakeExtender{isBinder: false, interestedRegionName: "region0"},
//				&fakeExtender{isBinder: true, interestedRegionName: "region0"},
//			},
//			wantBinderID: 1,
//		},
//		{
//			name:       "one of the extenders is a binder, but not interested in region",
//			regionName: "region1",
//			extenders: []framework.Extender{
//				&fakeExtender{isBinder: false, interestedRegionName: "region1"},
//				&fakeExtender{isBinder: true, interestedRegionName: "region0"},
//			},
//			wantBinderID: -1, // default binding.
//		},
//		{
//			name:       "ignore when extender bind failed",
//			regionName: "region1",
//			extenders: []framework.Extender{
//				&fakeExtender{isBinder: true, errBind: true, interestedRegionName: "region1", ignorable: true},
//			},
//			wantBinderID: -1, // default binding.
//		},
//	}
//
//	for _, test := range table {
//		t.Run(test.name, func(t *testing.T) {
//			region := st.MakeRegion().Name(test.regionName).Obj()
//			defaultBound := false
//			client := clientsetfake.NewSimpleClientset(region)
//			client.PrependReactor("create", "regions", func(action clienttesting.Action) (bool, runtime.Object, error) {
//				if action.GetSubresource() == "binding" {
//					defaultBound = true
//				}
//				return false, nil, nil
//			})
//			_, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			fwk, err := tf.NewFramework(ctx,
//				[]tf.RegisterPluginFunc{
//					tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//					tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//				}, "", frameworkruntime.WithClientSet(client), frameworkruntime.WithEventRecorder(&events.FakeRecorder{}))
//			if err != nil {
//				t.Fatal(err)
//			}
//			sched := &Scheduler{
//				Extenders:                  test.extenders,
//				Cache:                      internalcache.New(ctx, 100*time.Millisecond),
//				runnerInfoSnapshot:         nil,
//				percentageOfRunnersToScore: 0,
//			}
//			status := sched.bind(ctx, fwk, region, "runner", nil)
//			if !status.IsSuccess() {
//				t.Error(status.AsError())
//			}
//
//			// Checking default binding.
//			if wantBound := test.wantBinderID == -1; defaultBound != wantBound {
//				t.Errorf("got bound with default binding: %v, want %v", defaultBound, wantBound)
//			}
//
//			// Checking extenders binding.
//			for i, ext := range test.extenders {
//				wantBound := i == test.wantBinderID
//				if gotBound := ext.(*fakeExtender).gotBind; gotBound != wantBound {
//					t.Errorf("got bound with extender #%d: %v, want %v", i, gotBound, wantBound)
//				}
//			}
//
//		})
//	}
//}
//
//func TestUpdateRegion(t *testing.T) {
//	tests := []struct {
//		name                       string
//		currentRegionConditions    []corev1.RegionCondition
//		newRegionCondition         *corev1.RegionCondition
//		currentNominatedRunnerName string
//		newNominatingInfo          *framework.NominatingInfo
//		expectedPatchRequests      int
//		expectedPatchDataPattern   string
//	}{
//		{
//			name:                    "Should make patch request to add region condition when there are none currently",
//			currentRegionConditions: []corev1.RegionCondition{},
//			newRegionCondition: &corev1.RegionCondition{
//				Type:               "newType",
//				Status:             "newStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 1, 1, 1, 1, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 1, 1, 1, 1, time.UTC)),
//				Reason:             "newReason",
//				Message:            "newMessage",
//			},
//			expectedPatchRequests:    1,
//			expectedPatchDataPattern: `{"status":{"conditions":\[{"lastProbeTime":"2020-05-13T01:01:01Z","lastTransitionTime":".*","message":"newMessage","reason":"newReason","status":"newStatus","type":"newType"}]}}`,
//		},
//		{
//			name: "Should make patch request to add a new region condition when there is already one with another type",
//			currentRegionConditions: []corev1.RegionCondition{
//				{
//					Type:               "someOtherType",
//					Status:             "someOtherTypeStatus",
//					LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 11, 0, 0, 0, 0, time.UTC)),
//					LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 10, 0, 0, 0, 0, time.UTC)),
//					Reason:             "someOtherTypeReason",
//					Message:            "someOtherTypeMessage",
//				},
//			},
//			newRegionCondition: &corev1.RegionCondition{
//				Type:               "newType",
//				Status:             "newStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 1, 1, 1, 1, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 1, 1, 1, 1, time.UTC)),
//				Reason:             "newReason",
//				Message:            "newMessage",
//			},
//			expectedPatchRequests:    1,
//			expectedPatchDataPattern: `{"status":{"\$setElementOrder/conditions":\[{"type":"someOtherType"},{"type":"newType"}],"conditions":\[{"lastProbeTime":"2020-05-13T01:01:01Z","lastTransitionTime":".*","message":"newMessage","reason":"newReason","status":"newStatus","type":"newType"}]}}`,
//		},
//		{
//			name: "Should make patch request to update an existing region condition",
//			currentRegionConditions: []corev1.RegionCondition{
//				{
//					Type:               "currentType",
//					Status:             "currentStatus",
//					LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//					LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//					Reason:             "currentReason",
//					Message:            "currentMessage",
//				},
//			},
//			newRegionCondition: &corev1.RegionCondition{
//				Type:               "currentType",
//				Status:             "newStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 1, 1, 1, 1, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 1, 1, 1, 1, time.UTC)),
//				Reason:             "newReason",
//				Message:            "newMessage",
//			},
//			expectedPatchRequests:    1,
//			expectedPatchDataPattern: `{"status":{"\$setElementOrder/conditions":\[{"type":"currentType"}],"conditions":\[{"lastProbeTime":"2020-05-13T01:01:01Z","lastTransitionTime":".*","message":"newMessage","reason":"newReason","status":"newStatus","type":"currentType"}]}}`,
//		},
//		{
//			name: "Should make patch request to update an existing region condition, but the transition time should remain unchanged because the status is the same",
//			currentRegionConditions: []corev1.RegionCondition{
//				{
//					Type:               "currentType",
//					Status:             "currentStatus",
//					LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//					LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//					Reason:             "currentReason",
//					Message:            "currentMessage",
//				},
//			},
//			newRegionCondition: &corev1.RegionCondition{
//				Type:               "currentType",
//				Status:             "currentStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 1, 1, 1, 1, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//				Reason:             "newReason",
//				Message:            "newMessage",
//			},
//			expectedPatchRequests:    1,
//			expectedPatchDataPattern: `{"status":{"\$setElementOrder/conditions":\[{"type":"currentType"}],"conditions":\[{"lastProbeTime":"2020-05-13T01:01:01Z","message":"newMessage","reason":"newReason","type":"currentType"}]}}`,
//		},
//		{
//			name: "Should not make patch request if region condition already exists and is identical and nominated runner name is not set",
//			currentRegionConditions: []corev1.RegionCondition{
//				{
//					Type:               "currentType",
//					Status:             "currentStatus",
//					LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//					LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//					Reason:             "currentReason",
//					Message:            "currentMessage",
//				},
//			},
//			newRegionCondition: &v1.RegionCondition{
//				Type:               "currentType",
//				Status:             "currentStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//				Reason:             "currentReason",
//				Message:            "currentMessage",
//			},
//			currentNominatedRunnerName: "runner1",
//			expectedPatchRequests:      0,
//		},
//		{
//			name: "Should make patch request if region condition already exists and is identical but nominated runner name is set and different",
//			currentRegionConditions: []v1.RegionCondition{
//				{
//					Type:               "currentType",
//					Status:             "currentStatus",
//					LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//					LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//					Reason:             "currentReason",
//					Message:            "currentMessage",
//				},
//			},
//			newRegionCondition: &v1.RegionCondition{
//				Type:               "currentType",
//				Status:             "currentStatus",
//				LastProbeTime:      metav1.NewTime(time.Date(2020, 5, 13, 0, 0, 0, 0, time.UTC)),
//				LastTransitionTime: metav1.NewTime(time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC)),
//				Reason:             "currentReason",
//				Message:            "currentMessage",
//			},
//			newNominatingInfo:        &framework.NominatingInfo{NominatingMode: framework.ModeOverride, NominatedRunnerName: "runner1"},
//			expectedPatchRequests:    1,
//			expectedPatchDataPattern: `{"status":{"nominatedRunnerName":"runner1"}}`,
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			actualPatchRequests := 0
//			var actualPatchData string
//			cs := &clientsetfake.Clientset{}
//			cs.AddReactor("patch", "regions", func(action clienttesting.Action) (bool, runtime.Object, error) {
//				actualPatchRequests++
//				patch := action.(clienttesting.PatchAction)
//				actualPatchData = string(patch.GetPatch())
//				// For this test, we don't care about the result of the patched region, just that we got the expected
//				// patch request, so just returning &v1.Region{} here is OK because scheduler doesn't use the response.
//				return true, &v1.Region{}, nil
//			})
//
//			region := st.MakeRegion().Name("foo").NominatedRunnerName(test.currentNominatedRunnerName).Conditions(test.currentRegionConditions).Obj()
//
//			ctx, cancel := context.WithCancel(context.Background())
//			defer cancel()
//			if err := updateRegion(ctx, cs, region, test.newRegionCondition, test.newNominatingInfo); err != nil {
//				t.Fatalf("Error calling update: %v", err)
//			}
//
//			if actualPatchRequests != test.expectedPatchRequests {
//				t.Fatalf("Actual patch requests (%d) does not equal expected patch requests (%d), actual patch data: %v", actualPatchRequests, test.expectedPatchRequests, actualPatchData)
//			}
//
//			regex, err := regexp.Compile(test.expectedPatchDataPattern)
//			if err != nil {
//				t.Fatalf("Error compiling regexp for %v: %v", test.expectedPatchDataPattern, err)
//			}
//
//			if test.expectedPatchRequests > 0 && !regex.MatchString(actualPatchData) {
//				t.Fatalf("Patch data mismatch: Actual was %v, but expected to match regexp %v", actualPatchData, test.expectedPatchDataPattern)
//			}
//		})
//	}
//}
//
//func Test_SelectHost(t *testing.T) {
//	tests := []struct {
//		name                string
//		list                []framework.RunnerPluginScores
//		topRunnersCnt       int
//		possibleRunners     sets.Set[string]
//		possibleRunnerLists [][]framework.RunnerPluginScores
//		wantError           error
//	}{
//		{
//			name: "unique properly ordered scores",
//			list: []framework.RunnerPluginScores{
//				{Name: "runner1", TotalScore: 1},
//				{Name: "runner2", TotalScore: 2},
//			},
//			topRunnersCnt:   2,
//			possibleRunners: sets.New("runner2"),
//			possibleRunnerLists: [][]framework.RunnerPluginScores{
//				{
//					{Name: "runner2", TotalScore: 2},
//					{Name: "runner1", TotalScore: 1},
//				},
//			},
//		},
//		{
//			name: "numberOfRunnerScoresToReturn > len(list)",
//			list: []framework.RunnerPluginScores{
//				{Name: "runner1", TotalScore: 1},
//				{Name: "runner2", TotalScore: 2},
//			},
//			topRunnersCnt:   100,
//			possibleRunners: sets.New("runner2"),
//			possibleRunnerLists: [][]framework.RunnerPluginScores{
//				{
//					{Name: "runner2", TotalScore: 2},
//					{Name: "runner1", TotalScore: 1},
//				},
//			},
//		},
//		{
//			name: "equal scores",
//			list: []framework.RunnerPluginScores{
//				{Name: "runner2.1", TotalScore: 2},
//				{Name: "runner2.2", TotalScore: 2},
//				{Name: "runner2.3", TotalScore: 2},
//			},
//			topRunnersCnt:   2,
//			possibleRunners: sets.New("runner2.1", "runner2.2", "runner2.3"),
//			possibleRunnerLists: [][]framework.RunnerPluginScores{
//				{
//					{Name: "runner2.1", TotalScore: 2},
//					{Name: "runner2.2", TotalScore: 2},
//				},
//				{
//					{Name: "runner2.1", TotalScore: 2},
//					{Name: "runner2.3", TotalScore: 2},
//				},
//				{
//					{Name: "runner2.2", TotalScore: 2},
//					{Name: "runner2.1", TotalScore: 2},
//				},
//				{
//					{Name: "runner2.2", TotalScore: 2},
//					{Name: "runner2.3", TotalScore: 2},
//				},
//				{
//					{Name: "runner2.3", TotalScore: 2},
//					{Name: "runner2.1", TotalScore: 2},
//				},
//				{
//					{Name: "runner2.3", TotalScore: 2},
//					{Name: "runner2.2", TotalScore: 2},
//				},
//			},
//		},
//		{
//			name: "out of order scores",
//			list: []framework.RunnerPluginScores{
//				{Name: "runner3.1", TotalScore: 3},
//				{Name: "runner2.1", TotalScore: 2},
//				{Name: "runner1.1", TotalScore: 1},
//				{Name: "runner3.2", TotalScore: 3},
//			},
//			topRunnersCnt:   3,
//			possibleRunners: sets.New("runner3.1", "runner3.2"),
//			possibleRunnerLists: [][]framework.RunnerPluginScores{
//				{
//					{Name: "runner3.1", TotalScore: 3},
//					{Name: "runner3.2", TotalScore: 3},
//					{Name: "runner2.1", TotalScore: 2},
//				},
//				{
//					{Name: "runner3.2", TotalScore: 3},
//					{Name: "runner3.1", TotalScore: 3},
//					{Name: "runner2.1", TotalScore: 2},
//				},
//			},
//		},
//		{
//			name:            "empty priority list",
//			list:            []framework.RunnerPluginScores{},
//			possibleRunners: sets.Set[string]{},
//			wantError:       errEmptyPriorityList,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			// increase the randomness
//			for i := 0; i < 10; i++ {
//				got, scoreList, err := selectHost(test.list, test.topRunnersCnt)
//				if err != test.wantError {
//					t.Fatalf("unexpected error is returned from selectHost: got: %v want: %v", err, test.wantError)
//				}
//				if test.possibleRunners.Len() == 0 {
//					if got != "" {
//						t.Fatalf("expected nothing returned as selected Runner, but actually %s is returned from selectHost", got)
//					}
//					return
//				}
//				if !test.possibleRunners.Has(got) {
//					t.Errorf("got %s is not in the possible map %v", got, test.possibleRunners)
//				}
//				if got != scoreList[0].Name {
//					t.Errorf("The head of list should be the selected Runner's score: got: %v, expected: %v", scoreList[0], got)
//				}
//				for _, list := range test.possibleRunnerLists {
//					if cmp.Equal(list, scoreList) {
//						return
//					}
//				}
//				t.Errorf("Unexpected scoreList: %v", scoreList)
//			}
//		})
//	}
//}
//
//func TestFindRunnersThatPassExtenders(t *testing.T) {
//	tests := []struct {
//		name                    string
//		extenders               []tf.FakeExtender
//		runners                 []*v1.Runner
//		filteredRunnersStatuses framework.RunnerToStatusMap
//		expectsErr              bool
//		expectedRunners         []*v1.Runner
//		expectedStatuses        framework.RunnerToStatusMap
//	}{
//		{
//			name: "error",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.ErrorPredicateExtender},
//				},
//			},
//			runners:                 makeRunnerList([]string{"a"}),
//			filteredRunnersStatuses: make(framework.RunnerToStatusMap),
//			expectsErr:              true,
//		},
//		{
//			name: "success",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.TruePredicateExtender},
//				},
//			},
//			runners:                 makeRunnerList([]string{"a"}),
//			filteredRunnersStatuses: make(framework.RunnerToStatusMap),
//			expectsErr:              false,
//			expectedRunners:         makeRunnerList([]string{"a"}),
//			expectedStatuses:        make(framework.RunnerToStatusMap),
//		},
//		{
//			name: "unschedulable",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates: []tf.FitPredicate{func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
//						if runner.Runner().Name == "a" {
//							return framework.NewStatus(framework.Success)
//						}
//						return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//					}},
//				},
//			},
//			runners:                 makeRunnerList([]string{"a", "b"}),
//			filteredRunnersStatuses: make(framework.RunnerToStatusMap),
//			expectsErr:              false,
//			expectedRunners:         makeRunnerList([]string{"a"}),
//			expectedStatuses: framework.RunnerToStatusMap{
//				"b": framework.NewStatus(framework.Unschedulable, fmt.Sprintf("FakeExtender: runner %q failed", "b")),
//			},
//		},
//		{
//			name: "unschedulable and unresolvable",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates: []tf.FitPredicate{func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
//						if runner.Runner().Name == "a" {
//							return framework.NewStatus(framework.Success)
//						}
//						if runner.Runner().Name == "b" {
//							return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//						}
//						return framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//					}},
//				},
//			},
//			runners:                 makeRunnerList([]string{"a", "b", "c"}),
//			filteredRunnersStatuses: make(framework.RunnerToStatusMap),
//			expectsErr:              false,
//			expectedRunners:         makeRunnerList([]string{"a"}),
//			expectedStatuses: framework.RunnerToStatusMap{
//				"b": framework.NewStatus(framework.Unschedulable, fmt.Sprintf("FakeExtender: runner %q failed", "b")),
//				"c": framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("FakeExtender: runner %q failed and unresolvable", "c")),
//			},
//		},
//		{
//			name: "extender may overwrite the statuses",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates: []tf.FitPredicate{func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
//						if runner.Runner().Name == "a" {
//							return framework.NewStatus(framework.Success)
//						}
//						if runner.Runner().Name == "b" {
//							return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//						}
//						return framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//					}},
//				},
//			},
//			runners: makeRunnerList([]string{"a", "b", "c"}),
//			filteredRunnersStatuses: framework.RunnerToStatusMap{
//				"c": framework.NewStatus(framework.Unschedulable, fmt.Sprintf("FakeFilterPlugin: runner %q failed", "c")),
//			},
//			expectsErr:      false,
//			expectedRunners: makeRunnerList([]string{"a"}),
//			expectedStatuses: framework.RunnerToStatusMap{
//				"b": framework.NewStatus(framework.Unschedulable, fmt.Sprintf("FakeExtender: runner %q failed", "b")),
//				"c": framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("FakeFilterPlugin: runner %q failed", "c"), fmt.Sprintf("FakeExtender: runner %q failed and unresolvable", "c")),
//			},
//		},
//		{
//			name: "multiple extenders",
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates: []tf.FitPredicate{func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
//						if runner.Runner().Name == "a" {
//							return framework.NewStatus(framework.Success)
//						}
//						if runner.Runner().Name == "b" {
//							return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//						}
//						return framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//					}},
//				},
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates: []tf.FitPredicate{func(region *corev1.Region, runner *framework.RunnerInfo) *framework.Status {
//						if runner.Runner().Name == "a" {
//							return framework.NewStatus(framework.Success)
//						}
//						return framework.NewStatus(framework.Unschedulable, fmt.Sprintf("runner %q is not allowed", runner.Runner().Name))
//					}},
//				},
//			},
//			runners:                 makeRunnerList([]string{"a", "b", "c"}),
//			filteredRunnersStatuses: make(framework.RunnerToStatusMap),
//			expectsErr:              false,
//			expectedRunners:         makeRunnerList([]string{"a"}),
//			expectedStatuses: framework.RunnerToStatusMap{
//				"b": framework.NewStatus(framework.Unschedulable, fmt.Sprintf("FakeExtender: runner %q failed", "b")),
//				"c": framework.NewStatus(framework.UnschedulableAndUnresolvable, fmt.Sprintf("FakeExtender: runner %q failed and unresolvable", "c")),
//			},
//		},
//	}
//
//	cmpOpts := []cmp.Option{
//		cmp.Comparer(func(s1 framework.Status, s2 framework.Status) bool {
//			return s1.Code() == s2.Code() && reflect.DeepEqual(s1.Reasons(), s2.Reasons())
//		}),
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_, ctx := ktesting.NewTestContext(t)
//			var extenders []framework.Extender
//			for ii := range tt.extenders {
//				extenders = append(extenders, &tt.extenders[ii])
//			}
//
//			region := st.MakeRegion().Name("1").UID("1").Obj()
//			got, err := findRunnersThatPassExtenders(ctx, extenders, region, tf.BuildRunnerInfos(tt.runners), tt.filteredRunnersStatuses)
//			runners := make([]*v1.Runner, len(got))
//			for i := 0; i < len(got); i++ {
//				runners[i] = got[i].Runner()
//			}
//			if tt.expectsErr {
//				if err == nil {
//					t.Error("Unexpected non-error")
//				}
//			} else {
//				if err != nil {
//					t.Errorf("Unexpected error: %v", err)
//				}
//				if diff := cmp.Diff(tt.expectedRunners, runners); diff != "" {
//					t.Errorf("filtered runners (-want,+got):\n%s", diff)
//				}
//				if diff := cmp.Diff(tt.expectedStatuses, tt.filteredRunnersStatuses, cmpOpts...); diff != "" {
//					t.Errorf("filtered statuses (-want,+got):\n%s", diff)
//				}
//			}
//		})
//	}
//}
//
//func TestSchedulerScheduleRegion(t *testing.T) {
//	fts := feature.Features{}
//	tests := []struct {
//		name                 string
//		registerPlugins      []tf.RegisterPluginFunc
//		extenders            []tf.FakeExtender
//		runners              []string
//		pvcs                 []v1.PersistentVolumeClaim
//		region               *corev1.Region
//		regions              []*corev1.Region
//		wantRunners          sets.Set[string]
//		wantEvaluatedRunners *int32
//		wErr                 error
//	}{
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("FalseFilter", tf.NewFalseFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			region:  st.MakeRegion().Name("2").UID("2").Obj(),
//			name:    "test 1",
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("2").UID("2").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("FalseFilter"),
//						"runner2": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("FalseFilter"),
//					},
//					UnschedulablePlugins: sets.New("FalseFilter"),
//				},
//			},
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"runner1", "runner2"},
//			region:      st.MakeRegion().Name("ignore").UID("ignore").Obj(),
//			wantRunners: sets.New("runner1", "runner2"),
//			name:        "test 2",
//			wErr:        nil,
//		},
//		{
//			// Fits on a runner where the region ID matches the runner name
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("MatchFilter", tf.NewMatchFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"runner1", "runner2"},
//			region:      st.MakeRegion().Name("runner2").UID("runner2").Obj(),
//			wantRunners: sets.New("runner2"),
//			name:        "test 3",
//			wErr:        nil,
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"3", "2", "1"},
//			region:      st.MakeRegion().Name("ignore").UID("ignore").Obj(),
//			wantRunners: sets.New("3"),
//			name:        "test 4",
//			wErr:        nil,
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("MatchFilter", tf.NewMatchFilterPlugin),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"3", "2", "1"},
//			region:      st.MakeRegion().Name("2").UID("2").Obj(),
//			wantRunners: sets.New("2"),
//			name:        "test 5",
//			wErr:        nil,
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterScorePlugin("ReverseNumericMap", newReverseNumericMapPlugin(), 2),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"3", "2", "1"},
//			region:      st.MakeRegion().Name("2").UID("2").Obj(),
//			wantRunners: sets.New("1"),
//			name:        "test 6",
//			wErr:        nil,
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterFilterPlugin("FalseFilter", tf.NewFalseFilterPlugin),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"3", "2", "1"},
//			region:  st.MakeRegion().Name("2").UID("2").Obj(),
//			name:    "test 7",
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("2").UID("2").Obj(),
//				NumAllRunners: 3,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"3": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("FalseFilter"),
//						"2": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("FalseFilter"),
//						"1": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("FalseFilter"),
//					},
//					UnschedulablePlugins: sets.New("FalseFilter"),
//				},
//			},
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("NoRegionsFilter", NewNoRegionsFilterPlugin),
//				tf.RegisterFilterPlugin("MatchFilter", tf.NewMatchFilterPlugin),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("2").UID("2").Runner("2").Phase(v1.RegionRunning).Obj(),
//			},
//			region:  st.MakeRegion().Name("2").UID("2").Obj(),
//			runners: []string{"1", "2"},
//			name:    "test 8",
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("2").UID("2").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"1": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("MatchFilter"),
//						"2": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("NoRegionsFilter"),
//					},
//					UnschedulablePlugins: sets.New("MatchFilter", "NoRegionsFilter"),
//				},
//			},
//		},
//		{
//			// Region with existing PVC
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(volumebinding.Name, frameworkruntime.FactoryAdapter(fts, volumebinding.New)),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			pvcs: []v1.PersistentVolumeClaim{
//				{
//					ObjectMeta: metav1.ObjectMeta{Name: "existingPVC", UID: types.UID("existingPVC"), Namespace: v1.NamespaceDefault},
//					Spec:       v1.PersistentVolumeClaimSpec{VolumeName: "existingPV"},
//				},
//			},
//			region:      st.MakeRegion().Name("ignore").UID("ignore").Namespace(v1.NamespaceDefault).PVC("existingPVC").Obj(),
//			wantRunners: sets.New("runner1", "runner2"),
//			name:        "existing PVC",
//			wErr:        nil,
//		},
//		{
//			// Region with non existing PVC
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(volumebinding.Name, frameworkruntime.FactoryAdapter(fts, volumebinding.New)),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			region:  st.MakeRegion().Name("ignore").UID("ignore").PVC("unknownPVC").Obj(),
//			name:    "unknown PVC",
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("ignore").UID("ignore").PVC("unknownPVC").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.UnschedulableAndUnresolvable, `persistentvolumeclaim "unknownPVC" not found`).WithPlugin("VolumeBinding"),
//						"runner2": framework.NewStatus(framework.UnschedulableAndUnresolvable, `persistentvolumeclaim "unknownPVC" not found`).WithPlugin("VolumeBinding"),
//					},
//					PreFilterMsg:         `persistentvolumeclaim "unknownPVC" not found`,
//					UnschedulablePlugins: sets.New(volumebinding.Name),
//				},
//			},
//		},
//		{
//			// Region with deleting PVC
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(volumebinding.Name, frameworkruntime.FactoryAdapter(fts, volumebinding.New)),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			pvcs:    []v1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "existingPVC", UID: types.UID("existingPVC"), Namespace: v1.NamespaceDefault, DeletionTimestamp: &metav1.Time{}}}},
//			region:  st.MakeRegion().Name("ignore").UID("ignore").Namespace(v1.NamespaceDefault).PVC("existingPVC").Obj(),
//			name:    "deleted PVC",
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("ignore").UID("ignore").Namespace(v1.NamespaceDefault).PVC("existingPVC").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.UnschedulableAndUnresolvable, `persistentvolumeclaim "existingPVC" is being deleted`).WithPlugin("VolumeBinding"),
//						"runner2": framework.NewStatus(framework.UnschedulableAndUnresolvable, `persistentvolumeclaim "existingPVC" is being deleted`).WithPlugin("VolumeBinding"),
//					},
//					PreFilterMsg:         `persistentvolumeclaim "existingPVC" is being deleted`,
//					UnschedulablePlugins: sets.New(volumebinding.Name),
//				},
//			},
//		},
//		{
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterScorePlugin("FalseMap", newFalseMapPlugin(), 1),
//				tf.RegisterScorePlugin("TrueMap", newTrueMapPlugin(), 2),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"2", "1"},
//			region:  st.MakeRegion().Name("2").Obj(),
//			name:    "test error with priority map",
//			wErr:    fmt.Errorf("running Score plugins: %w", fmt.Errorf(`plugin "FalseMap" failed with: %w`, errPrioritize)),
//		},
//		{
//			name: "test regiontopologyspread plugin - 2 runners with maxskew=1",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPluginAsExtensions(
//					regiontopologyspread.Name,
//					regionTopologySpreadFunc,
//					"PreFilter",
//					"Filter",
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			region: st.MakeRegion().Name("p").UID("p").Label("foo", "").SpreadConstraint(1, "hostname", v1.DoNotSchedule, &metav1.LabelSelector{
//				MatchExpressions: []metav1.LabelSelectorRequirement{
//					{
//						Key:      "foo",
//						Operator: metav1.LabelSelectorOpExists,
//					},
//				},
//			}, nil, nil, nil, nil).Obj(),
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("region1").UID("region1").Label("foo", "").Runner("runner1").Phase(v1.RegionRunning).Obj(),
//			},
//			wantRunners: sets.New("runner2"),
//			wErr:        nil,
//		},
//		{
//			name: "test regiontopologyspread plugin - 3 runners with maxskew=2",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPluginAsExtensions(
//					regiontopologyspread.Name,
//					regionTopologySpreadFunc,
//					"PreFilter",
//					"Filter",
//				),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2", "runner3"},
//			region: st.MakeRegion().Name("p").UID("p").Label("foo", "").SpreadConstraint(2, "hostname", v1.DoNotSchedule, &metav1.LabelSelector{
//				MatchExpressions: []metav1.LabelSelectorRequirement{
//					{
//						Key:      "foo",
//						Operator: metav1.LabelSelectorOpExists,
//					},
//				},
//			}, nil, nil, nil, nil).Obj(),
//			regions: []*corev1.Region{
//				st.MakeRegion().Name("region1a").UID("region1a").Label("foo", "").Runner("runner1").Phase(v1.RegionRunning).Obj(),
//				st.MakeRegion().Name("region1b").UID("region1b").Label("foo", "").Runner("runner1").Phase(v1.RegionRunning).Obj(),
//				st.MakeRegion().Name("region2").UID("region2").Label("foo", "").Runner("runner2").Phase(v1.RegionRunning).Obj(),
//			},
//			wantRunners: sets.New("runner2", "runner3"),
//			wErr:        nil,
//		},
//		{
//			name: "test with filter plugin returning Unschedulable status",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin(
//					"FakeFilter",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{"3": framework.Unschedulable}),
//				),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"3"},
//			region:      st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//			wantRunners: nil,
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//				NumAllRunners: 1,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"3": framework.NewStatus(framework.Unschedulable, "injecting failure for region test-filter").WithPlugin("FakeFilter"),
//					},
//					UnschedulablePlugins: sets.New("FakeFilter"),
//				},
//			},
//		},
//		{
//			name: "test with extender which filters out some Runners",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin(
//					"FakeFilter",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{"3": framework.Unschedulable}),
//				),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Predicates:   []tf.FitPredicate{tf.FalsePredicateExtender},
//				},
//			},
//			runners:     []string{"1", "2", "3"},
//			region:      st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//			wantRunners: nil,
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//				NumAllRunners: 3,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"1": framework.NewStatus(framework.Unschedulable, `FakeExtender: runner "1" failed`),
//						"2": framework.NewStatus(framework.Unschedulable, `FakeExtender: runner "2" failed`),
//						"3": framework.NewStatus(framework.Unschedulable, "injecting failure for region test-filter").WithPlugin("FakeFilter"),
//					},
//					UnschedulablePlugins: sets.New("FakeFilter", framework.ExtenderName),
//				},
//			},
//		},
//		{
//			name: "test with filter plugin returning UnschedulableAndUnresolvable status",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin(
//					"FakeFilter",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{"3": framework.UnschedulableAndUnresolvable}),
//				),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"3"},
//			region:      st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//			wantRunners: nil,
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//				NumAllRunners: 1,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"3": framework.NewStatus(framework.UnschedulableAndUnresolvable, "injecting failure for region test-filter").WithPlugin("FakeFilter"),
//					},
//					UnschedulablePlugins: sets.New("FakeFilter"),
//				},
//			},
//		},
//		{
//			name: "test with partial failed filter plugin",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin(
//					"FakeFilter",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{"1": framework.Unschedulable}),
//				),
//				tf.RegisterScorePlugin("NumericMap", newNumericMapPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"1", "2"},
//			region:      st.MakeRegion().Name("test-filter").UID("test-filter").Obj(),
//			wantRunners: nil,
//			wErr:        nil,
//		},
//		{
//			name: "test prefilter plugin returning Unschedulable status",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter",
//					tf.NewFakePreFilterPlugin("FakePreFilter", nil, framework.NewStatus(framework.UnschedulableAndUnresolvable, "injected unschedulable status")),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"1", "2"},
//			region:      st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wantRunners: nil,
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"1": framework.NewStatus(framework.UnschedulableAndUnresolvable, "injected unschedulable status").WithPlugin("FakePreFilter"),
//						"2": framework.NewStatus(framework.UnschedulableAndUnresolvable, "injected unschedulable status").WithPlugin("FakePreFilter"),
//					},
//					PreFilterMsg:         "injected unschedulable status",
//					UnschedulablePlugins: sets.New("FakePreFilter"),
//				},
//			},
//		},
//		{
//			name: "test prefilter plugin returning error status",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter",
//					tf.NewFakePreFilterPlugin("FakePreFilter", nil, framework.NewStatus(framework.Error, "injected error status")),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:     []string{"1", "2"},
//			region:      st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wantRunners: nil,
//			wErr:        fmt.Errorf(`running PreFilter plugin "FakePreFilter": %w`, errors.New("injected error status")),
//		},
//		{
//			name: "test prefilter plugin returning runner",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter1",
//					tf.NewFakePreFilterPlugin("FakePreFilter1", nil, nil),
//				),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter2",
//					tf.NewFakePreFilterPlugin("FakePreFilter2", &framework.PreFilterResult{RunnerNames: sets.New("runner2")}, nil),
//				),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter3",
//					tf.NewFakePreFilterPlugin("FakePreFilter3", &framework.PreFilterResult{RunnerNames: sets.New("runner1", "runner2")}, nil),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:              []string{"runner1", "runner2", "runner3"},
//			region:               st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wantRunners:          sets.New("runner2"),
//			wantEvaluatedRunners: ptr.To[int32](3),
//		},
//		{
//			name: "test prefilter plugin returning non-intersecting runners",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter1",
//					tf.NewFakePreFilterPlugin("FakePreFilter1", nil, nil),
//				),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter2",
//					tf.NewFakePreFilterPlugin("FakePreFilter2", &framework.PreFilterResult{RunnerNames: sets.New("runner2")}, nil),
//				),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter3",
//					tf.NewFakePreFilterPlugin("FakePreFilter3", &framework.PreFilterResult{RunnerNames: sets.New("runner1")}, nil),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2", "runner3"},
//			region:  st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//				NumAllRunners: 3,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner(s) didn't satisfy plugin(s) [FakePreFilter2 FakePreFilter3] simultaneously"),
//						"runner2": framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner(s) didn't satisfy plugin(s) [FakePreFilter2 FakePreFilter3] simultaneously"),
//						"runner3": framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner(s) didn't satisfy plugin(s) [FakePreFilter2 FakePreFilter3] simultaneously"),
//					},
//					UnschedulablePlugins: sets.Set[string]{},
//					PreFilterMsg:         "runner(s) didn't satisfy plugin(s) [FakePreFilter2 FakePreFilter3] simultaneously",
//				},
//			},
//		},
//		{
//			name: "test prefilter plugin returning empty runner set",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter1",
//					tf.NewFakePreFilterPlugin("FakePreFilter1", nil, nil),
//				),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter2",
//					tf.NewFakePreFilterPlugin("FakePreFilter2", &framework.PreFilterResult{RunnerNames: sets.New[string]()}, nil),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1"},
//			region:  st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//				NumAllRunners: 1,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner(s) didn't satisfy plugin FakePreFilter2"),
//					},
//					UnschedulablePlugins: sets.Set[string]{},
//					PreFilterMsg:         "runner(s) didn't satisfy plugin FakePreFilter2",
//				},
//			},
//		},
//		{
//			name: "test some runners are filtered out by prefilter plugin and other are filtered out by filter plugin",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter",
//					tf.NewFakePreFilterPlugin("FakePreFilter", &framework.PreFilterResult{RunnerNames: sets.New[string]("runner2")}, nil),
//				),
//				tf.RegisterFilterPlugin(
//					"FakeFilter",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{"runner2": framework.Unschedulable}),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners: []string{"runner1", "runner2"},
//			region:  st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wErr: &framework.FitError{
//				Region:        st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//				NumAllRunners: 2,
//				Diagnosis: framework.Diagnosis{
//					RunnerToStatusMap: framework.RunnerToStatusMap{
//						"runner1": framework.NewStatus(framework.UnschedulableAndUnresolvable, "runner is filtered out by the prefilter result"),
//						"runner2": framework.NewStatus(framework.Unschedulable, "injecting failure for region test-prefilter").WithPlugin("FakeFilter"),
//					},
//					UnschedulablePlugins: sets.New("FakeFilter"),
//					PreFilterMsg:         "",
//				},
//			},
//		},
//		{
//			name: "test prefilter plugin returning skip",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter1",
//					tf.NewFakePreFilterPlugin("FakeFilter1", nil, nil),
//				),
//				tf.RegisterFilterPlugin(
//					"FakeFilter1",
//					tf.NewFakeFilterPlugin(map[string]framework.Code{
//						"runner1": framework.Unschedulable,
//					}),
//				),
//				tf.RegisterPluginAsExtensions("FakeFilter2", func(_ context.Context, configuration runtime.Object, f framework.Handle) (framework.Plugin, error) {
//					return tf.FakePreFilterAndFilterPlugin{
//						FakePreFilterPlugin: &tf.FakePreFilterPlugin{
//							Result: nil,
//							Status: framework.NewStatus(framework.Skip),
//						},
//						FakeFilterPlugin: &tf.FakeFilterPlugin{
//							// This Filter plugin shouldn't be executed in the Filter extension point due to skip.
//							// To confirm that, return the status code Error to all Runners.
//							FailedRunnerReturnCodeMap: map[string]framework.Code{
//								"runner1": framework.Error, "runner2": framework.Error, "runner3": framework.Error,
//							},
//						},
//					}, nil
//				}, "PreFilter", "Filter"),
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:              []string{"runner1", "runner2", "runner3"},
//			region:               st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wantRunners:          sets.New("runner2", "runner3"),
//			wantEvaluatedRunners: ptr.To[int32](3),
//		},
//		{
//			name: "test all prescore plugins return skip",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//				tf.RegisterPluginAsExtensions("FakePreScoreAndScorePlugin", tf.NewFakePreScoreAndScorePlugin("FakePreScoreAndScorePlugin", 0,
//					framework.NewStatus(framework.Skip, "fake skip"),
//					framework.NewStatus(framework.Error, "this score function shouldn't be executed because this plugin returned Skip in the PreScore"),
//				), "PreScore", "Score"),
//			},
//			runners:     []string{"runner1", "runner2"},
//			region:      st.MakeRegion().Name("ignore").UID("ignore").Obj(),
//			wantRunners: sets.New("runner1", "runner2"),
//		},
//		{
//			name: "test without score plugin no extra runners are evaluated",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:              []string{"runner1", "runner2", "runner3"},
//			region:               st.MakeRegion().Name("region1").UID("region1").Obj(),
//			wantRunners:          sets.New("runner1", "runner2", "runner3"),
//			wantEvaluatedRunners: ptr.To[int32](1),
//		},
//		{
//			name: "test no score plugin, prefilter plugin returning 2 runners",
//			registerPlugins: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterPreFilterPlugin(
//					"FakePreFilter",
//					tf.NewFakePreFilterPlugin("FakePreFilter", &framework.PreFilterResult{RunnerNames: sets.New("runner1", "runner2")}, nil),
//				),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			runners:              []string{"runner1", "runner2", "runner3"},
//			region:               st.MakeRegion().Name("test-prefilter").UID("test-prefilter").Obj(),
//			wantRunners:          sets.New("runner1", "runner2"),
//			wantEvaluatedRunners: ptr.To[int32](2),
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			cache := internalcache.New(ctx, time.Duration(0))
//			for _, region := range test.regions {
//				cache.AddRegion(logger, region)
//			}
//			var runners []*v1.Runner
//			for _, name := range test.runners {
//				runner := &v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"hostname": name}}}
//				runners = append(runners, runner)
//				cache.AddRunner(logger, runner)
//			}
//
//			cs := clientsetfake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(cs, 0)
//			for _, pvc := range test.pvcs {
//				metav1.SetMetaDataAnnotation(&pvc.ObjectMeta, volume.AnnBindCompleted, "true")
//				cs.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, &pvc, metav1.CreateOptions{})
//				if pvName := pvc.Spec.VolumeName; pvName != "" {
//					pv := v1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: pvName}}
//					cs.CoreV1().PersistentVolumes().Create(ctx, &pv, metav1.CreateOptions{})
//				}
//			}
//			snapshot := internalcache.NewSnapshot(test.regions, runners)
//			fwk, err := tf.NewFramework(
//				ctx,
//				test.registerPlugins, "",
//				frameworkruntime.WithSnapshotSharedLister(snapshot),
//				frameworkruntime.WithInformerFactory(informerFactory),
//				frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//			)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			var extenders []framework.Extender
//			for ii := range test.extenders {
//				extenders = append(extenders, &test.extenders[ii])
//			}
//			sched := &Scheduler{
//				Cache:                      cache,
//				runnerInfoSnapshot:         snapshot,
//				percentageOfRunnersToScore: schedulerapi.DefaultPercentageOfRunnersToScore,
//				Extenders:                  extenders,
//			}
//			sched.applyDefaultHandlers()
//
//			informerFactory.Start(ctx.Done())
//			informerFactory.WaitForCacheSync(ctx.Done())
//
//			result, err := sched.ScheduleRegion(ctx, fwk, framework.NewCycleState(), test.region)
//			if err != test.wErr {
//				gotFitErr, gotOK := err.(*framework.FitError)
//				wantFitErr, wantOK := test.wErr.(*framework.FitError)
//				if gotOK != wantOK {
//					t.Errorf("Expected err to be FitError: %v, but got %v (error: %v)", wantOK, gotOK, err)
//				} else if gotOK {
//					if diff := cmp.Diff(gotFitErr, wantFitErr); diff != "" {
//						t.Errorf("Unexpected fitErr: (-want, +got): %s", diff)
//					}
//				}
//			}
//			if test.wantRunners != nil && !test.wantRunners.Has(result.SuggestedHost) {
//				t.Errorf("Expected: %s, got: %s", test.wantRunners, result.SuggestedHost)
//			}
//			wantEvaluatedRunners := len(test.runners)
//			if test.wantEvaluatedRunners != nil {
//				wantEvaluatedRunners = int(*test.wantEvaluatedRunners)
//			}
//			if test.wErr == nil && wantEvaluatedRunners != result.EvaluatedRunners {
//				t.Errorf("Expected EvaluatedRunners: %d, got: %d", wantEvaluatedRunners, result.EvaluatedRunners)
//			}
//		})
//	}
//}
//
//func TestFindFitAllError(t *testing.T) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	runners := makeRunnerList([]string{"3", "2", "1"})
//	scheduler := makeScheduler(ctx, runners)
//
//	fwk, err := tf.NewFramework(
//		ctx,
//		[]tf.RegisterPluginFunc{
//			tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//			tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//			tf.RegisterFilterPlugin("MatchFilter", tf.NewMatchFilterPlugin),
//			tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		},
//		"",
//		frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(nil)),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	_, diagnosis, err := scheduler.findRunnersThatFitRegion(ctx, fwk, framework.NewCycleState(), &v1.Region{})
//	if err != nil {
//		t.Errorf("unexpected error: %v", err)
//	}
//
//	expected := framework.Diagnosis{
//		RunnerToStatusMap: framework.RunnerToStatusMap{
//			"1": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("MatchFilter"),
//			"2": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("MatchFilter"),
//			"3": framework.NewStatus(framework.Unschedulable, tf.ErrReasonFake).WithPlugin("MatchFilter"),
//		},
//		UnschedulablePlugins: sets.New("MatchFilter"),
//	}
//	if diff := cmp.Diff(diagnosis, expected); diff != "" {
//		t.Errorf("Unexpected diagnosis: (-want, +got): %s", diff)
//	}
//}
//
//func TestFindFitSomeError(t *testing.T) {
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	runners := makeRunnerList([]string{"3", "2", "1"})
//	scheduler := makeScheduler(ctx, runners)
//
//	fwk, err := tf.NewFramework(
//		ctx,
//		[]tf.RegisterPluginFunc{
//			tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//			tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//			tf.RegisterFilterPlugin("MatchFilter", tf.NewMatchFilterPlugin),
//			tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		},
//		"",
//		frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(nil)),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	region := st.MakeRegion().Name("1").UID("1").Obj()
//	_, diagnosis, err := scheduler.findRunnersThatFitRegion(ctx, fwk, framework.NewCycleState(), region)
//	if err != nil {
//		t.Errorf("unexpected error: %v", err)
//	}
//
//	if len(diagnosis.RunnerToStatusMap) != len(runners)-1 {
//		t.Errorf("unexpected failed status map: %v", diagnosis.RunnerToStatusMap)
//	}
//
//	if diff := cmp.Diff(sets.New("MatchFilter"), diagnosis.UnschedulablePlugins); diff != "" {
//		t.Errorf("Unexpected unschedulablePlugins: (-want, +got): %s", diagnosis.UnschedulablePlugins)
//	}
//
//	for _, runner := range runners {
//		if runner.Name == region.Name {
//			continue
//		}
//		t.Run(runner.Name, func(t *testing.T) {
//			status, found := diagnosis.RunnerToStatusMap[runner.Name]
//			if !found {
//				t.Errorf("failed to find runner %v in %v", runner.Name, diagnosis.RunnerToStatusMap)
//			}
//			reasons := status.Reasons()
//			if len(reasons) != 1 || reasons[0] != tf.ErrReasonFake {
//				t.Errorf("unexpected failures: %v", reasons)
//			}
//		})
//	}
//}
//
//func TestFindFitPredicateCallCounts(t *testing.T) {
//	tests := []struct {
//		name          string
//		region        *corev1.Region
//		expectedCount int32
//	}{
//		{
//			name:          "nominated regions have lower priority, predicate is called once",
//			region:        st.MakeRegion().Name("1").UID("1").Priority(highPriority).Obj(),
//			expectedCount: 1,
//		},
//		{
//			name:          "nominated regions have higher priority, predicate is called twice",
//			region:        st.MakeRegion().Name("1").UID("1").Priority(lowPriority).Obj(),
//			expectedCount: 2,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			runners := makeRunnerList([]string{"1"})
//
//			plugin := tf.FakeFilterPlugin{}
//			registerFakeFilterFunc := tf.RegisterFilterPlugin(
//				"FakeFilter",
//				func(_ context.Context, _ runtime.Object, fh framework.Handle) (framework.Plugin, error) {
//					return &plugin, nil
//				},
//			)
//			registerPlugins := []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				registerFakeFilterFunc,
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			}
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//			fwk, err := tf.NewFramework(
//				ctx,
//				registerPlugins, "",
//				frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(nil)),
//			)
//			if err != nil {
//				t.Fatal(err)
//			}
//
//			scheduler := makeScheduler(ctx, runners)
//			if err := scheduler.Cache.UpdateSnapshot(logger, scheduler.runnerInfoSnapshot); err != nil {
//				t.Fatal(err)
//			}
//			regioninfo, err := framework.NewRegionInfo(st.MakeRegion().UID("nominated").Priority(midPriority).Obj())
//			if err != nil {
//				t.Fatal(err)
//			}
//			fwk.AddNominatedRegion(logger, regioninfo, &framework.NominatingInfo{NominatingMode: framework.ModeOverride, NominatedRunnerName: "1"})
//
//			_, _, err = scheduler.findRunnersThatFitRegion(ctx, fwk, framework.NewCycleState(), test.region)
//			if err != nil {
//				t.Errorf("unexpected error: %v", err)
//			}
//			if test.expectedCount != plugin.NumFilterCalled {
//				t.Errorf("predicate was called %d times, expected is %d", plugin.NumFilterCalled, test.expectedCount)
//			}
//		})
//	}
//}
//
//// The point of this test is to show that you:
////   - get the same priority for a zero-request region as for a region with the defaults requests,
////     both when the zero-request region is already on the runner and when the zero-request region
////     is the one being scheduled.
////   - don't get the same score no matter what we schedule.
//func TestZeroRequest(t *testing.T) {
//	// A region with no resources. We expect spreading to count it as having the default resources.
//	noResources := v1.RegionSpec{
//		Containers: []v1.Container{
//			{},
//		},
//	}
//	noResources1 := noResources
//	noResources1.RunnerName = "runner1"
//	// A region with the same resources as a 0-request region gets by default as its resources (for spreading).
//	small := v1.RegionSpec{
//		Containers: []v1.Container{
//			{
//				Resources: v1.ResourceRequirements{
//					Requests: v1.ResourceList{
//						v1.ResourceCPU: resource.MustParse(
//							strconv.FormatInt(schedutil.DefaultMilliCPURequest, 10) + "m"),
//						v1.ResourceMemory: resource.MustParse(
//							strconv.FormatInt(schedutil.DefaultMemoryRequest, 10)),
//					},
//				},
//			},
//		},
//	}
//	small2 := small
//	small2.RunnerName = "runner2"
//	// A larger region.
//	large := v1.RegionSpec{
//		Containers: []v1.Container{
//			{
//				Resources: v1.ResourceRequirements{
//					Requests: v1.ResourceList{
//						v1.ResourceCPU: resource.MustParse(
//							strconv.FormatInt(schedutil.DefaultMilliCPURequest*3, 10) + "m"),
//						v1.ResourceMemory: resource.MustParse(
//							strconv.FormatInt(schedutil.DefaultMemoryRequest*3, 10)),
//					},
//				},
//			},
//		},
//	}
//	large1 := large
//	large1.RunnerName = "runner1"
//	large2 := large
//	large2.RunnerName = "runner2"
//	tests := []struct {
//		region        *corev1.Region
//		regions       []*corev1.Region
//		runners       []*v1.Runner
//		name          string
//		expectedScore int64
//	}{
//		// The point of these next two tests is to show you get the same priority for a zero-request region
//		// as for a region with the defaults requests, both when the zero-request region is already on the runner
//		// and when the zero-request region is the one being scheduled.
//		{
//			region:  &v1.Region{Spec: noResources},
//			runners: []*v1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			name:    "test priority of zero-request region with runner with zero-request region",
//			regions: []*corev1.Region{
//				{Spec: large1}, {Spec: noResources1},
//				{Spec: large2}, {Spec: small2},
//			},
//			expectedScore: 150,
//		},
//		{
//			region:  &v1.Region{Spec: small},
//			runners: []*v1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			name:    "test priority of nonzero-request region with runner with zero-request region",
//			regions: []*corev1.Region{
//				{Spec: large1}, {Spec: noResources1},
//				{Spec: large2}, {Spec: small2},
//			},
//			expectedScore: 150,
//		},
//		// The point of this test is to verify that we're not just getting the same score no matter what we schedule.
//		{
//			region:  &v1.Region{Spec: large},
//			runners: []*v1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			name:    "test priority of larger region with runner with zero-request region",
//			regions: []*corev1.Region{
//				{Spec: large1}, {Spec: noResources1},
//				{Spec: large2}, {Spec: small2},
//			},
//			expectedScore: 130,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			client := clientsetfake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//
//			snapshot := internalcache.NewSnapshot(test.regions, test.runners)
//			fts := feature.Features{}
//			pluginRegistrations := []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterScorePlugin(runnerresources.Name, frameworkruntime.FactoryAdapter(fts, runnerresources.NewFit), 1),
//				tf.RegisterScorePlugin(runnerresources.BalancedAllocationName, frameworkruntime.FactoryAdapter(fts, runnerresources.NewBalancedAllocation), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			}
//			ctx, cancel := context.WithCancel(context.Background())
//			defer cancel()
//			fwk, err := tf.NewFramework(
//				ctx,
//				pluginRegistrations, "",
//				frameworkruntime.WithInformerFactory(informerFactory),
//				frameworkruntime.WithSnapshotSharedLister(snapshot),
//				frameworkruntime.WithClientSet(client),
//				frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//			)
//			if err != nil {
//				t.Fatalf("error creating framework: %+v", err)
//			}
//
//			sched := &Scheduler{
//				runnerInfoSnapshot:         snapshot,
//				percentageOfRunnersToScore: schedulerapi.DefaultPercentageOfRunnersToScore,
//			}
//			sched.applyDefaultHandlers()
//
//			state := framework.NewCycleState()
//			_, _, err = sched.findRunnersThatFitRegion(ctx, fwk, state, test.region)
//			if err != nil {
//				t.Fatalf("error filtering runners: %+v", err)
//			}
//			fwk.RunPreScorePlugins(ctx, state, test.region, tf.BuildRunnerInfos(test.runners))
//			list, err := prioritizeRunners(ctx, nil, fwk, state, test.region, tf.BuildRunnerInfos(test.runners))
//			if err != nil {
//				t.Errorf("unexpected error: %v", err)
//			}
//			for _, hp := range list {
//				if hp.TotalScore != test.expectedScore {
//					t.Errorf("expected %d for all priorities, got list %#v", test.expectedScore, list)
//				}
//			}
//		})
//	}
//}
//
//func Test_prioritizeRunners(t *testing.T) {
//	imageStatus1 := []v1.ContainerImage{
//		{
//			Names: []string{
//				"gcr.io/40:latest",
//				"gcr.io/40:v1",
//			},
//			SizeBytes: int64(80 * mb),
//		},
//		{
//			Names: []string{
//				"gcr.io/300:latest",
//				"gcr.io/300:v1",
//			},
//			SizeBytes: int64(300 * mb),
//		},
//	}
//
//	imageStatus2 := []v1.ContainerImage{
//		{
//			Names: []string{
//				"gcr.io/300:latest",
//			},
//			SizeBytes: int64(300 * mb),
//		},
//		{
//			Names: []string{
//				"gcr.io/40:latest",
//				"gcr.io/40:v1",
//			},
//			SizeBytes: int64(80 * mb),
//		},
//	}
//
//	imageStatus3 := []v1.ContainerImage{
//		{
//			Names: []string{
//				"gcr.io/600:latest",
//			},
//			SizeBytes: int64(600 * mb),
//		},
//		{
//			Names: []string{
//				"gcr.io/40:latest",
//			},
//			SizeBytes: int64(80 * mb),
//		},
//		{
//			Names: []string{
//				"gcr.io/900:latest",
//			},
//			SizeBytes: int64(900 * mb),
//		},
//	}
//	tests := []struct {
//		name                string
//		region              *corev1.Region
//		regions             []*corev1.Region
//		runners             []*v1.Runner
//		pluginRegistrations []tf.RegisterPluginFunc
//		extenders           []tf.FakeExtender
//		want                []framework.RunnerPluginScores
//	}{
//		{
//			name:    "the score from all plugins should be recorded in PluginToRunnerScores",
//			region:  &v1.Region{},
//			runners: []*v1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterScorePlugin(runnerresources.BalancedAllocationName, frameworkruntime.FactoryAdapter(feature.Features{}, runnerresources.NewBalancedAllocation), 1),
//				tf.RegisterScorePlugin("Runner2Prioritizer", tf.NewRunner2PrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: nil,
//			want: []framework.RunnerPluginScores{
//				{
//					Name: "runner1",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "Runner2Prioritizer",
//							Score: 10,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 110,
//				},
//				{
//					Name: "runner2",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "Runner2Prioritizer",
//							Score: 100,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 200,
//				},
//			},
//		},
//		{
//			name:    "the score from extender should also be recorded in PluginToRunnerScores with plugin scores",
//			region:  &v1.Region{},
//			runners: []*v1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterScorePlugin(runnerresources.BalancedAllocationName, frameworkruntime.FactoryAdapter(feature.Features{}, runnerresources.NewBalancedAllocation), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: []tf.FakeExtender{
//				{
//					ExtenderName: "FakeExtender1",
//					Weight:       1,
//					Prioritizers: []tf.PriorityConfig{
//						{
//							Weight:   3,
//							Function: tf.Runner1PrioritizerExtender,
//						},
//					},
//				},
//				{
//					ExtenderName: "FakeExtender2",
//					Weight:       1,
//					Prioritizers: []tf.PriorityConfig{
//						{
//							Weight:   2,
//							Function: tf.Runner2PrioritizerExtender,
//						},
//					},
//				},
//			},
//			want: []framework.RunnerPluginScores{
//				{
//					Name: "runner1",
//					Scores: []framework.PluginScore{
//
//						{
//							Name:  "FakeExtender1",
//							Score: 300,
//						},
//						{
//							Name:  "FakeExtender2",
//							Score: 20,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 420,
//				},
//				{
//					Name: "runner2",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "FakeExtender1",
//							Score: 30,
//						},
//						{
//							Name:  "FakeExtender2",
//							Score: 200,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 330,
//				},
//			},
//		},
//		{
//			name:    "plugin which returned skip in preScore shouldn't be executed in the score phase",
//			region:  &corev1.Region{},
//			runners: []*corev1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterScorePlugin(runnerresources.BalancedAllocationName, frameworkruntime.FactoryAdapter(feature.Features{}, runnerresources.NewBalancedAllocation), 1),
//				tf.RegisterScorePlugin("Runner2Prioritizer", tf.NewRunner2PrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//				tf.RegisterPluginAsExtensions("FakePreScoreAndScorePlugin", tf.NewFakePreScoreAndScorePlugin("FakePreScoreAndScorePlugin", 0,
//					framework.NewStatus(framework.Skip, "fake skip"),
//					framework.NewStatus(framework.Error, "this score function shouldn't be executed because this plugin returned Skip in the PreScore"),
//				), "PreScore", "Score"),
//			},
//			extenders: nil,
//			want: []framework.RunnerPluginScores{
//				{
//					Name: "runner1",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "Runner2Prioritizer",
//							Score: 10,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 110,
//				},
//				{
//					Name: "runner2",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "Runner2Prioritizer",
//							Score: 100,
//						},
//						{
//							Name:  "RunnerResourcesBalancedAllocation",
//							Score: 100,
//						},
//					},
//					TotalScore: 200,
//				},
//			},
//		},
//		{
//			name:    "all score plugins are skipped",
//			region:  &corev1.Region{},
//			runners: []*corev1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10), makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10)},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//				tf.RegisterPluginAsExtensions("FakePreScoreAndScorePlugin", tf.NewFakePreScoreAndScorePlugin("FakePreScoreAndScorePlugin", 0,
//					framework.NewStatus(framework.Skip, "fake skip"),
//					framework.NewStatus(framework.Error, "this score function shouldn't be executed because this plugin returned Skip in the PreScore"),
//				), "PreScore", "Score"),
//			},
//			extenders: nil,
//			want: []framework.RunnerPluginScores{
//				{Name: "runner1", Scores: []framework.PluginScore{}},
//				{Name: "runner2", Scores: []framework.PluginScore{}},
//			},
//		},
//		{
//			name: "the score from Image Locality plugin with image in all runners",
//			region: &corev1.Region{
//				Spec: corev1.RegionSpec{},
//			},
//			runners: []*corev1.Runner{
//				makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10, imageStatus1...),
//				makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10, imageStatus2...),
//				makeRunner("runner3", 1000, schedutil.DefaultMemoryRequest*10, imageStatus3...),
//			},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: nil,
//			want: []framework.RunnerPluginScores{
//				{
//					Name: "runner1",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 5,
//						},
//					},
//					TotalScore: 5,
//				},
//				{
//					Name: "runner2",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 5,
//						},
//					},
//					TotalScore: 5,
//				},
//				{
//					Name: "runner3",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 5,
//						},
//					},
//					TotalScore: 5,
//				},
//			},
//		},
//		{
//			name: "the score from Image Locality plugin with image in partial runners",
//			region: &corev1.Region{
//				Spec: corev1.RegionSpec{},
//			},
//			runners: []*corev1.Runner{makeRunner("runner1", 1000, schedutil.DefaultMemoryRequest*10, imageStatus1...),
//				makeRunner("runner2", 1000, schedutil.DefaultMemoryRequest*10, imageStatus2...),
//				makeRunner("runner3", 1000, schedutil.DefaultMemoryRequest*10, imageStatus3...),
//			},
//			pluginRegistrations: []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			},
//			extenders: nil,
//			want: []framework.RunnerPluginScores{
//				{
//					Name: "runner1",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 18,
//						},
//					},
//					TotalScore: 18,
//				},
//				{
//					Name: "runner2",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 18,
//						},
//					},
//					TotalScore: 18,
//				},
//				{
//					Name: "runner3",
//					Scores: []framework.PluginScore{
//						{
//							Name:  "ImageLocality",
//							Score: 0,
//						},
//					},
//					TotalScore: 0,
//				},
//			},
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			client := clientsetfake.NewSimpleClientset()
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//
//			ctx, cancel := context.WithCancel(context.Background())
//			defer cancel()
//			cache := internalcache.New(ctx, time.Duration(0))
//			for _, runner := range test.runners {
//				cache.AddRunner(klog.FromContext(ctx), runner)
//			}
//			snapshot := internalcache.NewEmptySnapshot()
//			if err := cache.UpdateSnapshot(klog.FromContext(ctx), snapshot); err != nil {
//				t.Fatal(err)
//			}
//			fwk, err := tf.NewFramework(
//				ctx,
//				test.pluginRegistrations, "",
//				frameworkruntime.WithInformerFactory(informerFactory),
//				frameworkruntime.WithSnapshotSharedLister(snapshot),
//				frameworkruntime.WithClientSet(client),
//				frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//			)
//			if err != nil {
//				t.Fatalf("error creating framework: %+v", err)
//			}
//
//			state := framework.NewCycleState()
//			var extenders []framework.Extender
//			for ii := range test.extenders {
//				extenders = append(extenders, &test.extenders[ii])
//			}
//			runnersscores, err := prioritizeRunners(ctx, extenders, fwk, state, test.region, tf.BuildRunnerInfos(test.runners))
//			if err != nil {
//				t.Errorf("unexpected error: %v", err)
//			}
//			for i := range runnersscores {
//				sort.Slice(runnersscores[i].Scores, func(j, k int) bool {
//					return runnersscores[i].Scores[j].Name < runnersscores[i].Scores[k].Name
//				})
//			}
//
//			if diff := cmp.Diff(test.want, runnersscores); diff != "" {
//				t.Errorf("returned runners scores (-want,+got):\n%s", diff)
//			}
//		})
//	}
//}
//
//var lowPriority, midPriority, highPriority = int32(0), int32(100), int32(1000)
//
//func TestNumFeasibleRunnersToFind(t *testing.T) {
//	tests := []struct {
//		name              string
//		globalPercentage  int32
//		profilePercentage *int32
//		numAllRunners     int32
//		wantNumRunners    int32
//	}{
//		{
//			name:           "not set percentageOfRunnersToScore and runners number not more than 50",
//			numAllRunners:  10,
//			wantNumRunners: 10,
//		},
//		{
//			name:              "set profile percentageOfRunnersToScore and runners number not more than 50",
//			profilePercentage: ptr.To[int32](40),
//			numAllRunners:     10,
//			wantNumRunners:    10,
//		},
//		{
//			name:           "not set percentageOfRunnersToScore and runners number more than 50",
//			numAllRunners:  1000,
//			wantNumRunners: 420,
//		},
//		{
//			name:              "set profile percentageOfRunnersToScore and runners number more than 50",
//			profilePercentage: ptr.To[int32](40),
//			numAllRunners:     1000,
//			wantNumRunners:    400,
//		},
//		{
//			name:              "set global and profile percentageOfRunnersToScore and runners number more than 50",
//			globalPercentage:  100,
//			profilePercentage: ptr.To[int32](40),
//			numAllRunners:     1000,
//			wantNumRunners:    400,
//		},
//		{
//			name:             "set global percentageOfRunnersToScore and runners number more than 50",
//			globalPercentage: 40,
//			numAllRunners:    1000,
//			wantNumRunners:   400,
//		},
//		{
//			name:           "not set profile percentageOfRunnersToScore and runners number more than 50*125",
//			numAllRunners:  6000,
//			wantNumRunners: 300,
//		},
//		{
//			name:              "set profile percentageOfRunnersToScore and runners number more than 50*125",
//			profilePercentage: ptr.To[int32](40),
//			numAllRunners:     6000,
//			wantNumRunners:    2400,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			sched := &Scheduler{
//				percentageOfRunnersToScore: tt.globalPercentage,
//			}
//			if gotNumRunners := sched.numFeasibleRunnersToFind(tt.profilePercentage, tt.numAllRunners); gotNumRunners != tt.wantNumRunners {
//				t.Errorf("Scheduler.numFeasibleRunnersToFind() = %v, want %v", gotNumRunners, tt.wantNumRunners)
//			}
//		})
//	}
//}
//
//func TestFairEvaluationForRunners(t *testing.T) {
//	numAllRunners := 500
//	runnerNames := make([]string, 0, numAllRunners)
//	for i := 0; i < numAllRunners; i++ {
//		runnerNames = append(runnerNames, strconv.Itoa(i))
//	}
//	runners := makeRunnerList(runnerNames)
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//	sched := makeScheduler(ctx, runners)
//
//	fwk, err := tf.NewFramework(
//		ctx,
//		[]tf.RegisterPluginFunc{
//			tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//			tf.RegisterFilterPlugin("TrueFilter", tf.NewTrueFilterPlugin),
//			tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//			tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		},
//		"",
//		frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(nil)),
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	// To make numAllRunners % runnersToFind != 0
//	sched.percentageOfRunnersToScore = 30
//	runnersToFind := int(sched.numFeasibleRunnersToFind(fwk.PercentageOfRunnersToScore(), int32(numAllRunners)))
//
//	// Iterating over all runners more than twice
//	for i := 0; i < 2*(numAllRunners/runnersToFind+1); i++ {
//		runnersThatFit, _, err := sched.findRunnersThatFitRegion(ctx, fwk, framework.NewCycleState(), &v1.Region{})
//		if err != nil {
//			t.Errorf("unexpected error: %v", err)
//		}
//		if len(runnersThatFit) != runnersToFind {
//			t.Errorf("got %d runners filtered, want %d", len(runnersThatFit), runnersToFind)
//		}
//		if sched.nextStartRunnerIndex != (i+1)*runnersToFind%numAllRunners {
//			t.Errorf("got %d lastProcessedRunnerIndex, want %d", sched.nextStartRunnerIndex, (i+1)*runnersToFind%numAllRunners)
//		}
//	}
//}
//
//func TestPreferNominatedRunnerFilterCallCounts(t *testing.T) {
//	tests := []struct {
//		name                  string
//		region                *corev1.Region
//		runnerReturnCodeMap   map[string]framework.Code
//		expectedCount         int32
//		expectedPatchRequests int
//	}{
//		{
//			name:          "region has the nominated runner set, filter is called only once",
//			region:        st.MakeRegion().Name("p_with_nominated_runner").UID("p").Priority(highPriority).NominatedRunnerName("runner1").Obj(),
//			expectedCount: 1,
//		},
//		{
//			name:          "region without the nominated region, filter is called for each runner",
//			region:        st.MakeRegion().Name("p_without_nominated_runner").UID("p").Priority(highPriority).Obj(),
//			expectedCount: 3,
//		},
//		{
//			name:                "nominated region cannot pass the filter, filter is called for each runner",
//			region:              st.MakeRegion().Name("p_with_nominated_runner").UID("p").Priority(highPriority).NominatedRunnerName("runner1").Obj(),
//			runnerReturnCodeMap: map[string]framework.Code{"runner1": framework.Unschedulable},
//			expectedCount:       4,
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			logger, ctx := ktesting.NewTestContext(t)
//			ctx, cancel := context.WithCancel(ctx)
//			defer cancel()
//
//			// create three runners in the cluster.
//			runners := makeRunnerList([]string{"runner1", "runner2", "runner3"})
//			client := clientsetfake.NewSimpleClientset(test.region)
//			informerFactory := informers.NewSharedInformerFactory(client, 0)
//			cache := internalcache.New(ctx, time.Duration(0))
//			for _, n := range runners {
//				cache.AddRunner(logger, n)
//			}
//			plugin := tf.FakeFilterPlugin{FailedRunnerReturnCodeMap: test.runnerReturnCodeMap}
//			registerFakeFilterFunc := tf.RegisterFilterPlugin(
//				"FakeFilter",
//				func(_ context.Context, _ runtime.Object, fh framework.Handle) (framework.Plugin, error) {
//					return &plugin, nil
//				},
//			)
//			registerPlugins := []tf.RegisterPluginFunc{
//				tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//				registerFakeFilterFunc,
//				tf.RegisterScorePlugin("EqualPrioritizerPlugin", tf.NewEqualPrioritizerPlugin(), 1),
//				tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//			}
//			fwk, err := tf.NewFramework(
//				ctx,
//				registerPlugins, "",
//				frameworkruntime.WithClientSet(client),
//				frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//			)
//			if err != nil {
//				t.Fatal(err)
//			}
//			snapshot := internalcache.NewSnapshot(nil, runners)
//
//			sched := &Scheduler{
//				Cache:                      cache,
//				runnerInfoSnapshot:         snapshot,
//				percentageOfRunnersToScore: schedulerapi.DefaultPercentageOfRunnersToScore,
//			}
//			sched.applyDefaultHandlers()
//
//			_, _, err = sched.findRunnersThatFitRegion(ctx, fwk, framework.NewCycleState(), test.region)
//			if err != nil {
//				t.Errorf("unexpected error: %v", err)
//			}
//			if test.expectedCount != plugin.NumFilterCalled {
//				t.Errorf("predicate was called %d times, expected is %d", plugin.NumFilterCalled, test.expectedCount)
//			}
//		})
//	}
//}
//
//func regionWithID(id, desiredHost string) *corev1.Region {
//	return st.MakeRegion().Name(id).UID(id).Runners(desiredHost).SchedulerName(testSchedulerName).Obj()
//}
//
//func deletingRegion(id string) *corev1.Region {
//	return st.MakeRegion().Name(id).UID(id).Terminating().Runners("").SchedulerName(testSchedulerName).Obj()
//}
//
//func regionWithPort(id, desiredHost string, port int) *corev1.Region {
//	region := regionWithID(id, desiredHost)
//	region.Spec.Containers = []v1.Container{
//		{Name: "ctr", Ports: []v1.ContainerPort{{HostPort: int32(port)}}},
//	}
//	return region
//}
//
//func regionWithResources(id, desiredHost string, limits v1.ResourceList, requests v1.ResourceList) *corev1.Region {
//	region := regionWithID(id, desiredHost)
//	region.Spec.Containers = []v1.Container{
//		{Name: "ctr", Resources: v1.ResourceRequirements{Limits: limits, Requests: requests}},
//	}
//	return region
//}
//
//func makeRunnerList(runnerNames []string) []*v1.Runner {
//	result := make([]*v1.Runner, 0, len(runnerNames))
//	for _, runnerName := range runnerNames {
//		result = append(result, &v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: runnerName}})
//	}
//	return result
//}
//
//// makeScheduler makes a simple Scheduler for testing.
//func makeScheduler(ctx context.Context, runners []*v1.Runner) *Scheduler {
//	logger := klog.FromContext(ctx)
//	cache := internalcache.New(ctx, time.Duration(0))
//	for _, n := range runners {
//		cache.AddRunner(logger, n)
//	}
//
//	sched := &Scheduler{
//		Cache:                      cache,
//		runnerInfoSnapshot:         emptySnapshot,
//		percentageOfRunnersToScore: schedulerapi.DefaultPercentageOfRunnersToScore,
//	}
//	sched.applyDefaultHandlers()
//	cache.UpdateSnapshot(logger, sched.runnerInfoSnapshot)
//	return sched
//}
//
//func makeRunner(runner string, milliCPU, memory int64, images ...v1.ContainerImage) *v1.Runner {
//	return &v1.Runner{
//		ObjectMeta: metav1.ObjectMeta{Name: runner},
//		Status: v1.RunnerStatus{
//			Capacity: v1.ResourceList{
//				v1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
//				v1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
//				"regions":         *resource.NewQuantity(100, resource.DecimalSI),
//			},
//			Allocatable: v1.ResourceList{
//
//				v1.ResourceCPU:    *resource.NewMilliQuantity(milliCPU, resource.DecimalSI),
//				v1.ResourceMemory: *resource.NewQuantity(memory, resource.BinarySI),
//				"regions":         *resource.NewQuantity(100, resource.DecimalSI),
//			},
//			Images: images,
//		},
//	}
//}
//
//// queuedRegionStore: regions queued before processing.
//// cache: scheduler cache that might contain assumed regions.
//func setupTestSchedulerWithOneRegionOnRunner(ctx context.Context, t *testing.T, queuedRegionStore *clientcache.FIFO, scache internalcache.Cache,
//	region *corev1.Region, runner *v1.Runner, fns ...tf.RegisterPluginFunc) (*Scheduler, chan *v1.Binding, chan error) {
//	scheduler, bindingChan, errChan := setupTestScheduler(ctx, t, queuedRegionStore, scache, nil, nil, fns...)
//
//	queuedRegionStore.Add(region)
//	// queuedRegionStore: [foo:8080]
//	// cache: []
//
//	scheduler.ScheduleOne(ctx)
//	// queuedRegionStore: []
//	// cache: [(assumed)foo:8080]
//
//	select {
//	case b := <-bindingChan:
//		expectBinding := &v1.Binding{
//			ObjectMeta: metav1.ObjectMeta{Name: region.Name, UID: types.UID(region.Name)},
//			Target:     v1.ObjectReference{Kind: "Runner", Name: runner.Name},
//		}
//		if !reflect.DeepEqual(expectBinding, b) {
//			t.Errorf("binding want=%v, get=%v", expectBinding, b)
//		}
//	case <-time.After(wait.ForeverTestTimeout):
//		t.Fatalf("timeout after %v", wait.ForeverTestTimeout)
//	}
//	return scheduler, bindingChan, errChan
//}
//
//// queuedRegionStore: regions queued before processing.
//// scache: scheduler cache that might contain assumed regions.
//func setupTestScheduler(ctx context.Context, t *testing.T, queuedRegionStore *clientcache.FIFO, cache internalcache.Cache, informerFactory informers.SharedInformerFactory, broadcaster events.EventBroadcaster, fns ...tf.RegisterPluginFunc) (*Scheduler, chan *v1.Binding, chan error) {
//	bindingChan := make(chan *v1.Binding, 1)
//	client := clientsetfake.NewSimpleClientset()
//	client.PrependReactor("create", "regions", func(action clienttesting.Action) (bool, runtime.Object, error) {
//		var b *v1.Binding
//		if action.GetSubresource() == "binding" {
//			b := action.(clienttesting.CreateAction).GetObject().(*v1.Binding)
//			bindingChan <- b
//		}
//		return true, b, nil
//	})
//
//	var recorder events.EventRecorder
//	if broadcaster != nil {
//		recorder = broadcaster.NewRecorder(scheme.Scheme, testSchedulerName)
//	} else {
//		recorder = &events.FakeRecorder{}
//	}
//
//	if informerFactory == nil {
//		informerFactory = informers.NewSharedInformerFactory(clientsetfake.NewSimpleClientset(), 0)
//	}
//	schedulingQueue := internalqueue.NewTestQueueWithInformerFactory(ctx, nil, informerFactory)
//
//	fwk, _ := tf.NewFramework(
//		ctx,
//		fns,
//		testSchedulerName,
//		frameworkruntime.WithClientSet(client),
//		frameworkruntime.WithEventRecorder(recorder),
//		frameworkruntime.WithInformerFactory(informerFactory),
//		frameworkruntime.WithRegionNominator(internalqueue.NewRegionNominator(informerFactory.Core().V1().Regions().Lister())),
//	)
//
//	errChan := make(chan error, 1)
//	sched := &Scheduler{
//		Cache:                      cache,
//		client:                     client,
//		runnerInfoSnapshot:         internalcache.NewEmptySnapshot(),
//		percentageOfRunnersToScore: schedulerapi.DefaultPercentageOfRunnersToScore,
//		NextRegion: func(logger klog.Logger) (*framework.QueuedRegionInfo, error) {
//			return &framework.QueuedRegionInfo{RegionInfo: mustNewRegionInfo(t, clientcache.Pop(queuedRegionStore).(*corev1.Region))}, nil
//		},
//		SchedulingQueue: schedulingQueue,
//		Profiles:        profile.Map{testSchedulerName: fwk},
//	}
//
//	sched.ScheduleRegion = sched.scheduleRegion
//	sched.FailureHandler = func(_ context.Context, _ framework.Framework, p *framework.QueuedRegionInfo, status *framework.Status, _ *framework.NominatingInfo, _ time.Time) {
//		err := status.AsError()
//		errChan <- err
//
//		msg := truncateMessage(err.Error())
//		fwk.EventRecorder().Eventf(p.Region, nil, v1.EventTypeWarning, "FailedScheduling", "Scheduling", msg)
//	}
//	return sched, bindingChan, errChan
//}
//
//func setupTestSchedulerWithVolumeBinding(ctx context.Context, t *testing.T, volumeBinder volumebinding.SchedulerVolumeBinder, broadcaster events.EventBroadcaster) (*Scheduler, chan *v1.Binding, chan error) {
//	logger := klog.FromContext(ctx)
//	testRunner := v1.Runner{ObjectMeta: metav1.ObjectMeta{Name: "runner1", UID: types.UID("runner1")}}
//	queuedRegionStore := clientcache.NewFIFO(clientcache.MetaNamespaceKeyFunc)
//	region := regionWithID("foo", "")
//	region.Namespace = "foo-ns"
//	region.Spec.Volumes = append(region.Spec.Volumes, v1.Volume{Name: "testVol",
//		VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: "testPVC"}}})
//	queuedRegionStore.Add(region)
//	scache := internalcache.New(ctx, 10*time.Minute)
//	scache.AddRunner(logger, &testRunner)
//	testPVC := v1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "testPVC", Namespace: region.Namespace, UID: types.UID("testPVC")}}
//	client := clientsetfake.NewSimpleClientset(&testRunner, &testPVC)
//	informerFactory := informers.NewSharedInformerFactory(client, 0)
//	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()
//	pvcInformer.Informer().GetStore().Add(&testPVC)
//
//	fns := []tf.RegisterPluginFunc{
//		tf.RegisterQueueSortPlugin(queuesort.Name, queuesort.New),
//		tf.RegisterBindPlugin(defaultbinder.Name, defaultbinder.New),
//		tf.RegisterPluginAsExtensions(volumebinding.Name, func(ctx context.Context, plArgs runtime.Object, handle framework.Handle) (framework.Plugin, error) {
//			return &volumebinding.VolumeBinding{Binder: volumeBinder, PVCLister: pvcInformer.Lister()}, nil
//		}, "PreFilter", "Filter", "Reserve", "PreBind"),
//	}
//	s, bindingChan, errChan := setupTestScheduler(ctx, t, queuedRegionStore, scache, informerFactory, broadcaster, fns...)
//	return s, bindingChan, errChan
//}
//
//// This is a workaround because golint complains that errors cannot
//// end with punctuation.  However, the real predicate error message does
//// end with a period.
//func makePredicateError(failReason string) error {
//	s := fmt.Sprintf("0/1 runners are available: %v.", failReason)
//	return fmt.Errorf(s)
//}
//
//func mustNewRegionInfo(t *testing.T, region *corev1.Region) *framework.RegionInfo {
//	regionInfo, err := framework.NewRegionInfo(region)
//	if err != nil {
//		t.Fatal(err)
//	}
//	return regionInfo
//}
