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

	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	monleader "github.com/olive-io/olive/plane/leader"
	internalregion "github.com/olive-io/olive/plane/scheduler/internal/region"
	internalrunner "github.com/olive-io/olive/plane/scheduler/internal/runner"
)

type Scheduler struct {
	options *Options

	ctx    context.Context
	cancel context.CancelFunc

	leaderNotifier monleader.Notifier

	clientSet       clientset.Interface
	informerFactory informers.SharedInformerFactory

	runnerQ internalrunner.SchedulingQueue
	regionQ internalregion.SchedulingQueue

	// definitionQ is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens.
	definitionQ workqueue.RateLimitingInterface
	processQ    workqueue.RateLimitingInterface

	messageC chan imessage
}

func NewScheduler(
	ctx context.Context,
	leaderNotifier monleader.Notifier,
	clientSet clientset.Interface,
	informerFactory informers.SharedInformerFactory,
	opts ...Option) (*Scheduler, error) {

	options := NewOptions(opts...)
	if err := options.Validate(); err != nil {
		return nil, err
	}

	runnerQ := internalrunner.NewSchedulingQueue(options.RegionLimit)
	regionQ := internalregion.NewSchedulingQueue(options.DefinitionLimit, options.RegionReplicas)

	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)

	definitionQ := workqueue.NewRateLimitingQueueWithConfig(ratelimiter, workqueue.RateLimitingQueueConfig{})
	processQ := workqueue.NewRateLimitingQueueWithConfig(ratelimiter, workqueue.RateLimitingQueueConfig{})

	ctx, cancel := context.WithCancel(ctx)
	scheduler := &Scheduler{
		options:        options,
		ctx:            ctx,
		cancel:         cancel,
		leaderNotifier: leaderNotifier,

		clientSet:       clientSet,
		informerFactory: informerFactory,

		runnerQ: runnerQ,
		regionQ: regionQ,

		definitionQ: definitionQ,
		processQ:    processQ,

		messageC: make(chan imessage, 20),
	}

	return scheduler, nil
}

func (s *Scheduler) Start() error {
	defer utilruntime.HandleCrash()

	informerSynced := []cache.InformerSynced{}

	runnerInformer := s.informerFactory.Core().V1().Runners()
	runnerRegistration, err := runnerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.runnerQ.Add(obj.(*corev1.Runner))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRunner := oldObj.(*corev1.Runner)
			newRunner := newObj.(*corev1.Runner)
			if oldRunner.ResourceVersion == newRunner.ResourceVersion {
				return
			}
			s.runnerQ.Update(newObj.(*corev1.Runner), s.options.RegionLimit)
		},
		DeleteFunc: func(obj interface{}) {
			s.runnerQ.Remove(obj.(*corev1.Runner))
		},
	})
	if err != nil {
		return err
	}
	informerSynced = append(informerSynced, runnerRegistration.HasSynced)

	regionInformer := s.informerFactory.Core().V1().Regions()
	regionRegistration, err := regionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.regionQ.Add(obj.(*corev1.Region))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldRegion := oldObj.(*corev1.Region)
			newRegion := newObj.(*corev1.Region)
			if oldRegion.ResourceVersion == newRegion.ResourceVersion {
				return
			}
			s.regionQ.Update(newRegion, s.options.RegionLimit)
		},
		DeleteFunc: func(obj interface{}) {
			s.regionQ.Remove(obj.(*corev1.Region))
		},
	})
	if err != nil {
		return err
	}
	informerSynced = append(informerSynced, regionRegistration.HasSynced)

	definitionRegistration, err := s.informerFactory.Core().V1().Definitions().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: s.enqueueDefinition,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDef := oldObj.(*corev1.Definition)
			newDef := newObj.(*corev1.Definition)
			if oldDef.ResourceVersion == newDef.ResourceVersion {
				return
			}
			if newDef.Status.Phase != corev1.DefPending {
				return
			}
			s.enqueueDefinition(newObj)
		},
	})
	if err != nil {
		return err
	}
	informerSynced = append(informerSynced, definitionRegistration.HasSynced)

	processRegistration, err := s.informerFactory.Core().V1().Processes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: s.enqueueProcess,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPr := oldObj.(*corev1.Process)
			newPr := newObj.(*corev1.Process)
			if oldPr.ResourceVersion == newPr.ResourceVersion {
				return
			}
			if newPr.Status.Phase != corev1.ProcessPending {
				return
			}
			s.enqueueProcess(newObj)
		},
	})
	if err != nil {
		return err
	}
	informerSynced = append(informerSynced, processRegistration.HasSynced)

	ctx := s.ctx

	if ok := cache.WaitForNamedCacheSync("monitor-scheduler", ctx.Done(), informerSynced...); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	runnerListener := runnerInformer.Lister()
	runners, err := runnerListener.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, runner := range runners {
		s.runnerQ.Add(runner)
	}

	regionListener := regionInformer.Lister()
	regions, err := regionListener.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, region := range regions {
		s.regionQ.Add(region)
	}

	klog.Infof("start region worker")
	go wait.UntilWithContext(ctx, s.runnerRegionWorker, time.Second*3)
	go wait.UntilWithContext(ctx, s.definitionWorker, time.Second)
	go wait.UntilWithContext(ctx, s.processWorker, time.Second)
	go s.process()

	return nil
}

func (s *Scheduler) process() {
	defer s.cancel()

LOOP:
	for {
		if !s.waitUtilLeader() {
			select {
			case <-s.ctx.Done():
				break LOOP
			case <-s.messageC:
				// to nothing, when monitor is not leader
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
				go s.handleAction(msg)
			}
		}
	}

	klog.Infof("Scheduler Done")
}

func (s *Scheduler) runnerRegionWorker(ctx context.Context) {
	if !s.leaderReady() {
		return
	}

	remind := s.options.InitRegionNum - s.regionQ.Len()
	if remind > 0 {
		s.putAction(newAllocRegionAction(remind))
		return
	}

	needScaled, ok := s.regionQ.RegionToScale()
	if ok {
		replicas := s.options.RegionReplicas - int(needScaled.Status.Replicas)
		s.putAction(newScaleRegionAction(needScaled.Name, replicas))
		return
	}
}

func (s *Scheduler) putAction(msg imessage) {
	s.messageC <- msg
}

func (s *Scheduler) handleAction(msg imessage) {
	body := msg.payload()
	switch req := body.(type) {
	case *AllocRegionRequest:
		s.AllocRegion(req)
	case *ScaleRegionRequest:
		s.ScaleRegion(req)
	}
}

func (s *Scheduler) AllocRegion(req *AllocRegionRequest) {
	for i := 0; i < req.Count; i++ {
		s.allocRegion()
	}
}

func (s *Scheduler) allocRegion() {

	opts := make([]internalrunner.NextOption, 0)
	runners := make([]*corev1.Runner, 0)
	for i := 0; i < s.options.RegionReplicas; i++ {
		snapshot, ok := s.runnerQ.Pop(opts...)
		if !ok {
			break
		}
		runners = append(runners, snapshot.Get())
	}
	defer s.runnerQ.Free()

	if len(runners) == 0 {
		return
	}

	region := &corev1.Region{
		Spec: corev1.RegionSpec{
			InitialReplicas:  []corev1.RegionReplica{},
			ElectionRTT:      s.options.RegionElectionTTL,
			HeartbeatRTT:     s.options.RegionHeartbeatTTL,
			DefinitionsLimit: int64(s.options.DefinitionLimit),
		},
		Status: corev1.RegionStatus{
			Phase: corev1.RegionPending,
		},
	}
	region.Name = "region"
	region.Spec.Leader = runners[0].Spec.ID
	for i, runner := range runners {
		replica := corev1.RegionReplica{
			Id:          int64(i) + 1,
			Runner:      runner.Name,
			RaftAddress: runner.Spec.PeerURL,
		}
		region.Spec.InitialReplicas = append(region.Spec.InitialReplicas, replica)
	}

	var err error
	region, err = s.clientSet.CoreV1().
		Regions().
		Create(s.ctx, region, metav1.CreateOptions{})
	if err != nil {
		return
	}
	klog.Infof("create new region %s, replica %d", region.Name, len(region.Spec.InitialReplicas))
	s.regionQ.Add(region)
}

func (s *Scheduler) ScaleRegion(req *ScaleRegionRequest) {
	rname := req.region
	scale := req.scale

	region, err := s.clientSet.CoreV1().Regions().Get(s.ctx, rname, metav1.GetOptions{})
	if err != nil {
		return
	}

	opts := make([]internalrunner.NextOption, 0)

	ignores := []string{}
	for _, replica := range region.Spec.InitialReplicas {
		ignores = append(ignores, replica.Runner)
	}
	opts = append(opts, internalrunner.WithIgnores(ignores...))

	runners := make([]*corev1.Runner, 0)
	for i := 0; i < scale; i++ {
		snapshot, ok := s.runnerQ.Pop(opts...)
		if !ok {
			break
		}
		runners = append(runners, snapshot.Get())
	}
	defer s.runnerQ.Free()

	if len(runners) == 0 {
		return
	}

	from := len(region.Spec.InitialReplicas)
	for _, runner := range runners {
		replicaNum := len(region.Spec.InitialReplicas)

		isJoin := true
		if replicaNum == 0 {
			isJoin = false
		}
		replicaId := replicaNum + 1
		newReplica := corev1.RegionReplica{
			Id:          int64(replicaId),
			Runner:      runner.Name,
			RaftAddress: runner.Spec.PeerURL,
			IsJoin:      isJoin,
		}
		region.Spec.InitialReplicas = append(region.Spec.InitialReplicas, newReplica)
	}

	klog.Infof("scale region %s: %d -> %d", region.Name, from, len(region.Spec.InitialReplicas))
	region, err = s.clientSet.CoreV1().
		Regions().
		Update(s.ctx, region, metav1.UpdateOptions{})
	if err != nil {
		return
	}
	s.regionQ.Update(region, s.options.DefinitionLimit)
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

func (s *Scheduler) leaderReady() bool {
	if s.leaderNotifier.IsLeader() {
		<-s.leaderNotifier.ReadyNotify()
		return true
	}
	return false
}
