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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	clientset "github.com/olive-io/olive/client-go/generated/clientset/versioned"
	coreInformers "github.com/olive-io/olive/client-go/generated/informers/externalversions/core/v1"
	"github.com/olive-io/olive/mon/leader"
	internalregion "github.com/olive-io/olive/mon/scheduler/internal/region"
	internalrunner "github.com/olive-io/olive/mon/scheduler/internal/runner"
)

type Scheduler struct {
	options *Options

	ctx    context.Context
	cancel context.CancelFunc

	leaderNotifier leader.Notifier

	clientSet      clientset.Interface
	runnerInformer coreInformers.RunnerInformer
	regionInformer coreInformers.RegionInformer

	runnerQ internalrunner.SchedulingQueue
	regionQ internalregion.SchedulingQueue

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

	runnerQ := internalrunner.NewSchedulingQueue(options.RegionLimit)
	regionQ := internalregion.NewSchedulingQueue(options.DefinitionLimit, options.RegionReplicas)

	ctx, cancel := context.WithCancel(ctx)
	scheduler := &Scheduler{
		options:        options,
		ctx:            ctx,
		cancel:         cancel,
		leaderNotifier: leaderNotifier,

		clientSet:      clientSet,
		runnerInformer: runnerInformer,
		regionInformer: regionInformer,

		runnerQ: runnerQ,
		regionQ: regionQ,

		messageC: make(chan imessage, 20),
	}

	return scheduler, nil
}

func (s *Scheduler) Start() error {
	defer utilruntime.HandleCrash()

	runnerListener := s.runnerInformer.Lister()
	runners, err := runnerListener.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, runner := range runners {
		s.runnerQ.Add(runner)
	}

	regionListener := s.regionInformer.Lister()
	regions, err := regionListener.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, region := range regions {
		s.regionQ.Add(region)
	}

	_, err = s.runnerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.runnerQ.Add(obj.(*corev1.Runner))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.runnerQ.Update(newObj.(*corev1.Runner), s.options.RegionLimit)
		},
		DeleteFunc: func(obj interface{}) {
			s.runnerQ.Remove(obj.(*corev1.Runner))
		},
	})
	if err != nil {
		return err
	}

	_, err = s.regionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.regionQ.Add(obj.(*corev1.Region))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			s.regionQ.Update(newObj.(*corev1.Region), s.options.RegionLimit)
		},
		DeleteFunc: func(obj interface{}) {
			s.regionQ.Remove(obj.(*corev1.Region))
		},
	})
	if err != nil {
		return err
	}

	ctx := s.ctx
	informerSynced := []cache.InformerSynced{
		s.runnerInformer.Informer().HasSynced,
		s.regionInformer.Informer().HasSynced,
	}

	if ok := cache.WaitForCacheSync(ctx.Done(), informerSynced...); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	defer s.cancel()

	go wait.UntilWithContext(ctx, s.runnerRegionWorker, time.Second*3)

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

func (s *Scheduler) runnerRegionWorker(ctx context.Context) {
	if !s.leaderReady() {
		return
	}

	remind := s.options.InitRegionNum - s.regionQ.Len()
	if remind > 0 {
		s.putMessage(newAllocRegionMsg(remind))
		return
	}

	needScaled, ok := s.regionQ.RegionToScale()
	if ok {
		replicas := s.options.RegionReplicas - int(needScaled.Status.Replicas)
		s.putMessage(newScaleRegionMsg(needScaled, replicas))
		return
	}
}

func (s *Scheduler) putMessage(msg imessage) {
	s.messageC <- msg
}

func (s *Scheduler) handleMessage(msg imessage) {
	body := msg.payload()
	switch req := body.(type) {
	case *AllocRegionRequest:
		s.AllocRegion(req)
	case *ScaleRegionRequest:
		s.ScaleRegion(req)
	}
}

func (s *Scheduler) AllocRegion(req *AllocRegionRequest) {

	opts := make([]internalrunner.NextOption, 0)
	runners := make([]*corev1.Runner, 0)
	for i := 0; i < s.options.RegionReplicas; i++ {
		snapshot, ok := s.runnerQ.Pop(opts...)
		if !ok {
			break
		}
		runners = append(runners, snapshot.Get())
	}
	defer s.runnerQ.Recycle()

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
		Status: corev1.RegionStatus{
			Phase: corev1.RegionPending,
			Stat:  &corev1.RegionStat{},
		},
	}
	region.Name = "region"
	region.Spec.Leader = runners[0].Spec.ID
	for i, runner := range runners {
		replica := corev1.RegionReplica{
			Id:          int64(i),
			Runner:      runner.Name,
			RaftAddress: runner.Spec.PeerURL,
		}
		region.Spec.Replicas = append(region.Spec.Replicas, replica)
	}

	region, err := s.clientSet.CoreV1().Regions().Create(context.TODO(), region, metav1.CreateOptions{})
	if err != nil {

	}
}

func (s *Scheduler) ScaleRegion(req *ScaleRegionRequest) {
	region := req.region
	scale := req.scale

	opts := make([]internalrunner.NextOption, 0)

	ignores := []string{}
	for _, replica := range region.Spec.Replicas {
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
	defer s.runnerQ.Recycle()

	if len(runners) == 0 {
		return
	}

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
