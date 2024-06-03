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

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	coreInformers "github.com/olive-io/olive/client-go/generated/informers/externalversions/core/v1"
	lister "github.com/olive-io/olive/client-go/generated/listers/core/v1"
	"github.com/olive-io/olive/mon/leader"
	"github.com/olive-io/olive/pkg/queue"
)

type Scheduler struct {
	options *Options

	leaderNotifier leader.Notifier

	clientSet clientset.Interface

	runnerLister    lister.RunnerLister
	runnerSynced    cache.InformerSynced
	runnerMap       *RunnerMap
	runnerWorkQueue *queue.SyncPriorityQueue

	regionLister    lister.RegionLister
	regionSynced    cache.InformerSynced
	regionMap       *RegionMap
	regionWorkQueue *queue.SyncPriorityQueue
}

func NewScheduler(
	leaderNotifier leader.Notifier,
	clientSet clientset.Interface,
	runnerInformer coreInformers.RunnerInformer,
	regionInformer coreInformers.RegionInformer,
	opts ...Option) (*Scheduler, error) {

	options := NewOptions(opts...)
	if err := options.Validate(); err != nil {
		return nil, err
	}

	runnerLister := runnerInformer.Lister()
	runners, err := runnerLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	runnerMap := NewRunnerMap()
	for _, runner := range runners {
		runnerMap.Put(runner)
	}

	regionLister := regionInformer.Lister()
	regions, err := regionLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	regionMap := NewRegionMap()
	for _, region := range regions {
		regionMap.Put(region)
	}

	scheduler := &Scheduler{
		options:         options,
		leaderNotifier:  leaderNotifier,
		clientSet:       clientSet,
		runnerLister:    runnerLister,
		runnerSynced:    runnerInformer.Informer().HasSynced,
		runnerMap:       runnerMap,
		runnerWorkQueue: queue.NewSync(),
		regionLister:    regionLister,
		regionSynced:    regionInformer.Informer().HasSynced,
		regionMap:       regionMap,
		regionWorkQueue: queue.NewSync(),
	}

	_, err = runnerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			scheduler.enqueueRunner(obj.(*corev1.Runner))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			scheduler.enqueueRunner(newObj.(*corev1.Runner))
		},
		DeleteFunc: func(obj interface{}) {
			scheduler.dequeueRunner(obj.(*corev1.Runner))
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = regionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			scheduler.enqueueRegion(obj.(*corev1.Region))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			scheduler.enqueueRegion(newObj.(*corev1.Region))
		},
		DeleteFunc: func(obj interface{}) {
			scheduler.dequeueRegion(obj.(*corev1.Region))
		},
	})
	if err != nil {
		return nil, err
	}

	return scheduler, nil
}

func (s *Scheduler) Start(ctx context.Context, workers int) error {
	defer utilruntime.HandleCrash()

	if ok := cache.WaitForCacheSync(ctx.Done(), s.runnerSynced, s.regionSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	period := time.Second

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, s.runRegionWorker, period)
	}

	<-ctx.Done()

	return nil
}

func (s *Scheduler) runRegionWorker(ctx context.Context) {
	for s.processNextRegionWorkItem(ctx) {
	}
}

func (s *Scheduler) enqueueRunner(runner *corev1.Runner) {
	s.runnerMap.Put(runner)

	// filter runner
	// TODO: add runner filter
	if runner.Status.Phase == corev1.RunnerActive {
		s.runnerWorkQueue.Push(&RunnerInfo{
			Runner:      runner,
			RegionLimit: s.options.RegionLimit,
		})
	}

	remind := s.options.InitRegionNum - s.regionMap.Len()
	if remind > 0 {
		s.allocRegion(remind)
	}
}

func (s *Scheduler) dequeueRunner(runner *corev1.Runner) {
	uid := string(runner.UID)
	s.runnerMap.Del(uid)
	s.runnerWorkQueue.Remove(uid)
}

func (s *Scheduler) enqueueRegion(region *corev1.Region) {
	s.regionMap.Put(region)
	// filter region
	// TODO: add region filter
	if int(region.Status.Replicas) < s.options.RegionReplicas {
		s.regionWorkQueue.Push(&RegionInfo{
			Region:          region,
			DefinitionLimit: s.options.DefinitionLimit,
		})
	}
}

func (s *Scheduler) dequeueRegion(region *corev1.Region) {
	uid := string(region.UID)
	s.regionMap.Del(uid)
	s.runnerWorkQueue.Remove(uid)
}

func (s *Scheduler) allocRegion(n int) {

}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (s *Scheduler) processNextRegionWorkItem(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	return true
}
