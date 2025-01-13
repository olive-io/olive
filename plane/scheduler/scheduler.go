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
	internalrunner "github.com/olive-io/olive/plane/scheduler/internal/runner"
)

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc

	options *Options

	leaderNotifier monleader.Notifier

	clientSet       clientset.Interface
	informerFactory informers.SharedInformerFactory

	runnerQ internalrunner.SchedulingQueue

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

	runnerQ := internalrunner.NewSchedulingQueue()

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
			s.runnerQ.Update(newObj.(*corev1.Runner))
		},
		DeleteFunc: func(obj interface{}) {
			s.runnerQ.Remove(obj.(*corev1.Runner))
		},
	})
	if err != nil {
		return err
	}
	informerSynced = append(informerSynced, runnerRegistration.HasSynced)

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

func (s *Scheduler) putAction(msg imessage) {
	s.messageC <- msg
}

func (s *Scheduler) handleAction(msg imessage) {
	//body := msg.payload()
	//switch req := body.(type) {
	//default:
	//}
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
