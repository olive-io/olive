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

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
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

	ctx    context.Context
	cancel context.CancelFunc

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

	messageC chan imessage
}

func NewScheduler(
	ctx context.Context,
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

	ctx, cancel := context.WithCancel(ctx)
	scheduler := &Scheduler{
		options:         options,
		ctx:             ctx,
		cancel:          cancel,
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

		messageC: make(chan imessage, 20),
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

func (s *Scheduler) Start() error {
	defer utilruntime.HandleCrash()
	ctx := s.ctx

	if ok := cache.WaitForCacheSync(ctx.Done(), s.runnerSynced, s.regionSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	defer s.cancel()

LOOP:
	for {
		if !s.waitUtilLeader() {
			select {
			case <-s.ctx.Done():
				break LOOP
			case <-s.messageC:
				// to nothing
			default:
			}
			continue
		}

		for {
			select {
			case <-s.ctx.Done():
				break LOOP
			case <-s.leaderNotifier.ChangeNotify():
				goto LOOP
			case msg := <-s.messageC:
				go s.handleMessage(msg)
			}
		}
	}

	return nil
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

	if remind := s.options.InitRegionNum - s.regionMap.Len(); remind > 0 {
		s.putMessage(newAllocRegionMsg(remind))
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
	if region.Status.Phase == corev1.RegionActive {
		s.regionWorkQueue.Push(&RegionInfo{
			Region:          region,
			DefinitionLimit: s.options.DefinitionLimit,
		})
	}

	if scale := s.options.RegionReplicas - int(region.Status.Replicas); scale != 0 {
		s.putMessage(newScaleRegionMsg(region.DeepCopy(), scale))
	}
}

func (s *Scheduler) dequeueRegion(region *corev1.Region) {
	uid := string(region.UID)
	s.regionMap.Del(uid)
	s.runnerWorkQueue.Remove(uid)
}

func (s *Scheduler) putMessage(msg imessage) {
	s.messageC <- msg
}

func (s *Scheduler) handleMessage(msg imessage) {
	body := msg.payload()
	switch box := body.(type) {
	case *allocRegionBox:
		for i := 0; i < box.num; i++ {
			s.allocRegion()
		}
	case *scaleRegionBox:
		s.scaleRegion(box)
	}
}

func (s *Scheduler) allocRegion() {
	runners := make([]*corev1.Runner, 0)
	for i := 0; i < s.options.RegionReplicas; i++ {
		runner, ok := s.runnerPop()
		if !ok {
			break
		}
		runners = append(runners, runner)
	}

	if len(runners) == 0 {
		return
	}

	region := &corev1.Region{
		Spec: corev1.RegionSpec{
			Replicas:         []corev1.RegionReplica{},
			ElectionRTT:      s.options.RegionElectionTTL,
			HeartbeatRTT:     s.options.RegionHeartbeatTTL,
			DefinitionsLimit: int64(s.options.DefinitionLimit),
		},
	}
	region.Name = "region"
	region.SetUID(types.UID(uuid.New().String()))

	region.Spec.Leader = runners[0].Spec.ID
	for i, runner := range runners {
		replica := corev1.RegionReplica{
			Id:          int64(i),
			Runner:      runner.Spec.ID,
			RaftAddress: runner.Spec.PeerURL,
		}
		region.Spec.Replicas = append(region.Spec.Replicas, replica)
	}

	region, err := s.clientSet.CoreV1().Regions().Create(context.TODO(), region, metav1.CreateOptions{})
	if err != nil {

	} else {
		s.regionMap.Put(region)
	}

	for _, runner := range runners {
		s.runnerWorkQueue.Set(&RunnerInfo{
			Runner:      runner,
			RegionLimit: s.options.RegionLimit,
		})
	}
}

func (s *Scheduler) scaleRegion(box *scaleRegionBox) {
	region := box.region
	binded := sets.New[int64]()
	for _, replica := range region.Spec.Replicas {
		binded.Insert(replica.Runner)
	}
	learners := make([]*corev1.Runner, 0)
	pops := make([]*RunnerInfo, 0)
	need := box.scale
	for {
		x, ok := s.runnerWorkQueue.Pop()
		if !ok {
			break
		}
		ri := x.(*RunnerInfo)
		pops = append(pops, ri)
	}
}

func (s *Scheduler) runnerPop() (*corev1.Runner, bool) {
	x, ok := s.runnerWorkQueue.Pop()
	if !ok {
		return nil, false
	}
	runnerInfo := x.(*RunnerInfo)
	return runnerInfo.Runner, true
}

func (s *Scheduler) regionPop() (*corev1.Region, bool) {
	x, ok := s.regionWorkQueue.Pop()
	if !ok {
		return nil, false
	}
	regionInfo := x.(*RegionInfo)
	return regionInfo.Region, true
}

func (s *Scheduler) waitUtilLeader() bool {
	for {
		if s.leaderNotifier.IsLeader() {
			<-s.leaderNotifier.ReadyNotify()
			return true
		}

		select {
		case <-s.ctx.Done():
			return false
		case <-s.leaderNotifier.ChangeNotify():
		}
	}
}
