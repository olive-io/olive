/*
Copyright 2023 The olive Authors

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

package raft

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/olive-io/bpmn/tracing"
	"go.etcd.io/etcd/pkg/v3/idutil"
	v3wait "go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
	clientgo "github.com/olive-io/olive/client-go"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/pkg/jsonpatch"
	"github.com/olive-io/olive/pkg/proxy"
	"github.com/olive-io/olive/runner/backend"
)

type Controller struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg Config
	nh  *dragonboat.NodeHost

	be      backend.IBackend
	regionW v3wait.Wait

	tracer tracing.ITracer
	proxy  proxy.IProxy

	client *clientgo.Client

	reqId *idutil.Generator
	reqW  v3wait.Wait

	pr *corev1.Runner

	smu    sync.RWMutex
	shards map[uint64]*Shard

	definitionQ workqueue.RateLimitingInterface
	processQ    workqueue.RateLimitingInterface

	stopping <-chan struct{}
	done     chan struct{}
}

func NewController(ctx context.Context, cfg Config, be backend.IBackend, client *clientgo.Client, pr *corev1.Runner) (*Controller, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewExample()
	}
	lg := cfg.Logger

	tracer := tracing.NewTracer(ctx)
	el := newEventListener(tracer)
	sl := newSystemListener()

	dir := cfg.DataDir
	peerAddr := cfg.RaftAddress
	raftRTTMs := cfg.RaftRTTMillisecond

	nhConfig := config.NodeHostConfig{
		NodeHostDir:         dir,
		RTTMillisecond:      raftRTTMs,
		RaftAddress:         peerAddr,
		EnableMetrics:       true,
		RaftEventListener:   el,
		SystemEventListener: sl,
		NotifyCommit:        true,
	}

	lg.Debug("start multi raft group",
		zap.String("module", "dragonboat"),
		zap.String("dir", dir),
		zap.String("listen", peerAddr),
		zap.Duration("raft-rtt", time.Millisecond*time.Duration(int64(raftRTTMs))))

	nh, err := dragonboat.NewNodeHost(nhConfig)
	if err != nil {
		return nil, err
	}

	//gwCfg := proxy.Config{
	//	Logger: lg,
	//	//Discovery: discovery,
	//}
	//py, err := proxy.NewProxy(gwCfg)
	//if err != nil {
	//	return nil, err
	//}

	// deep copy *corev1.Runner
	runner := pr.DeepCopy()
	ctx, cancel := context.WithCancel(ctx)

	ratelimiter := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 1000*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(50), 300)},
	)
	definitionQ := workqueue.NewRateLimitingQueueWithConfig(ratelimiter, workqueue.RateLimitingQueueConfig{})
	processQ := workqueue.NewRateLimitingQueueWithConfig(ratelimiter, workqueue.RateLimitingQueueConfig{})

	traces := tracer.SubscribeChannel(make(chan tracing.ITrace, 128))
	controller := &Controller{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		nh:     nh,
		tracer: tracer,

		client:  client,
		be:      be,
		regionW: v3wait.New(),
		reqId:   idutil.NewGenerator(0, time.Now()),
		reqW:    v3wait.New(),
		pr:      runner,
		shards:  make(map[uint64]*Shard),

		definitionQ: definitionQ,
		processQ:    processQ,
	}

	go controller.watchTrace(traces)
	return controller, nil
}

func (c *Controller) Start(stopping <-chan struct{}) error {
	c.stopping = stopping
	c.done = make(chan struct{}, 1)

	var err error
	if err = c.prepareShards(); err != nil {
		return err
	}

	informerFactory := informers.NewSharedInformerFactory(c.client.Clientset, time.Second*15)

	syncs := make([]cache.InformerSynced, 0)

	regionRegistration, err := informerFactory.Core().V1().Regions().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			_ = c.CreateShard(c.ctx, obj.(*corev1.Region))
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			patch, _ := jsonpatch.CreateJSONPatch(oldObj, newObj)
			_ = c.SyncShard(c.ctx, newObj.(*corev1.Region), patch)
		},
		DeleteFunc: func(obj interface{}) {
			_ = c.RemoveShard(obj.(*corev1.Region))
		},
	})
	if err != nil {
		return err
	}
	syncs = append(syncs, regionRegistration.HasSynced)

	definitionRegistration, err := informerFactory.Core().V1().Definitions().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueDef,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldDef := oldObj.(*corev1.Definition)
			newDef := newObj.(*corev1.Definition)
			if oldDef.ResourceVersion == newDef.ResourceVersion {
				return
			}
			c.enqueueDef(newDef)
		},
		DeleteFunc: nil,
	})
	if err != nil {
		return err
	}
	syncs = append(syncs, definitionRegistration.HasSynced)

	processRegistration, err := informerFactory.Core().V1().Processes().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueProcess,
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPr := oldObj.(*corev1.Process)
			newPr := newObj.(*corev1.Process)
			if oldPr.ResourceVersion == newPr.ResourceVersion {
				return
			}
			c.enqueueProcess(newPr)
		},
		DeleteFunc: nil,
	})
	if err != nil {
		return err
	}
	syncs = append(syncs, processRegistration.HasSynced)

	informerFactory.Start(stopping)

	if ok := cache.WaitForNamedCacheSync("runner-raft-controller", c.stopping, syncs...); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	regions, _ := informerFactory.Core().V1().Regions().Lister().List(labels.Everything())
	for i := range regions {
		region := regions[i]
		err = c.SyncShard(c.ctx, region, nil)
		if err != nil {
			return err
		}
	}

	go wait.UntilWithContext(c.ctx, c.runDefinitionWorker, time.Second)
	go wait.UntilWithContext(c.ctx, c.runProcessWorker, time.Second)
	go c.run()
	return nil
}

func (c *Controller) watchTrace(traces <-chan tracing.ITrace) {
	for {
		select {
		case <-c.stopping:
			return
		case trace := <-traces:
			switch tt := trace.(type) {
			case leaderTrace:
				shard, ok := c.popShard(tt.ShardID)
				if !ok {
					break
				}
				lead := tt.LeaderID
				term := tt.Term
				shard.setTerm(term)
				oldLead := shard.getLeader()
				newLeader := oldLead != lead && lead != 0
				shard.setLeader(lead, newLeader)
				if lead != 0 {
					shard.notifyAboutReady()
				}
				shard.notifyAboutChange()
				c.setShard(shard)

			case *readTrace:
				ctx, region, query := tt.ctx, tt.region, tt.query
				result, err := c.nh.SyncRead(ctx, region, query)

				var ar *applyResult
				if result != nil {
					ar, _ = result.(*applyResult)
				}
				tt.Write(ar, err)

			case *proposeTrace:
				ctx, region, cmd := tt.ctx, tt.region, tt.data
				session := c.nh.GetNoOPSession(region)
				_, err := c.nh.SyncPropose(ctx, session, cmd)
				tt.Trigger(err)

			case *regionStatTrace:
				id, stat := tt.Id, tt.stat

				ctx := c.ctx
				name := fmt.Sprintf("rn%d", id)
				region, err := c.client.CoreV1().Regions().Get(ctx, name, metav1.GetOptions{})
				if err != nil {
					continue
				}
				region.Status.Stat = *stat
				region.Status.Phase = corev1.RegionActive
				region, err = c.client.CoreV1().Regions().UpdateStatus(ctx, region, metav1.UpdateOptions{})
				if err != nil {

				}

			case *processStatTrace:
				stat := tt.stat

				ctx := c.ctx
				process, err := c.client.CoreV1().Processes(stat.Namespace).Get(ctx, stat.Spec.ProcessName, metav1.GetOptions{})
				if err != nil {

				} else {
					process.Status.Phase = stat.Status.Phase
					process.Status.Message = stat.Status.Message
					process, err = c.client.CoreV1().Processes(stat.Namespace).UpdateStatus(ctx, process, metav1.UpdateOptions{})
					if err != nil {

					}

					klog.Infof("update process %s/%s", process.Namespace, process.Name)
				}
			}
		}
	}
}

func (c *Controller) run() {

	defer c.Stop()
	for {
		select {
		case <-c.stopping:
			return
		}
	}
}

func (c *Controller) Stop() {
	c.nh.Close()
	c.cancel()
	c.tracer.Done()

	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *Controller) RunnerStat() ([]int64, []string) {
	c.smu.RLock()
	defer c.smu.RUnlock()

	shards := make([]int64, 0)
	leaders := make([]string, 0)
	for _, shard := range c.shards {
		shards = append(shards, int64(shard.id))
		lead := shard.getLeader()
		if lead == 0 {
			continue
		}

		//replicas := shard.getInfo().Spec.InitialReplicas
		//if len(replicas) == 0 {
		//	continue
		//}
		//for _, replica := range replicas {
		//	if int64(lead) == replica.Id && replica.Runner == c.pr.Name {
		//		sv := semver.Version{
		//			Major: int64(shard.id),
		//			Minor: replica.Id,
		//		}
		//		leaders = append(leaders, sv.String())
		//	}
		//}
	}
	RegionCounter.Set(float64(len(shards)))
	LeaderCounter.Set(float64(len(leaders)))
	return shards, leaders
}

func (c *Controller) SubscribeTrace() <-chan tracing.ITrace {
	traceChannel := make(chan tracing.ITrace, 10)
	return c.tracer.SubscribeChannel(traceChannel)
}

func (c *Controller) GetDefinitionArchive(ctx context.Context, req *pb.GetDefinitionArchiveRequest) (*pb.GetDefinitionArchiveResponse, error) {
	definition, err := c.client.CoreV1().Definitions(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if definition.Spec.Region == 0 {
		return nil, fmt.Errorf("definition not be binded")
	}

	resp := &pb.GetDefinitionArchiveResponse{}
	region, ok := c.getShard(uint64(definition.Spec.Region))
	if !ok {
		return nil, ErrNoRegion
	}

	definitions, err := region.GetDefinitionArchive(ctx, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	resp.Definitions = definitions

	return resp, nil
}

func (c *Controller) GetProcessStat(ctx context.Context, req *pb.GetProcessStatRequest) (*pb.GetProcessStatResponse, error) {
	resp := &pb.GetProcessStatResponse{}
	region, ok := c.getShard(req.Region)
	if !ok {
		return nil, ErrNoRegion
	}

	stat, err := region.GetProcessStat(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	if err != nil {
		return nil, err
	}
	resp.Stat = stat

	return resp, nil
}
