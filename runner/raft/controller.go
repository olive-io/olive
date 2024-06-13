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
	urlpkg "net/url"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/coreos/go-semver/semver"
	json "github.com/json-iterator/go"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/olive-io/bpmn/tracing"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	pb "github.com/olive-io/olive/apis/pb/olive"
	clientgo "github.com/olive-io/olive/client-go"
	informers "github.com/olive-io/olive/client-go/generated/informers/externalversions"
	"github.com/olive-io/olive/pkg/jsonpatch"
	"github.com/olive-io/olive/pkg/proxy"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
)

type Controller struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg Config
	nh  *dragonboat.NodeHost

	be      backend.IBackend
	regionW wait.Wait

	tracer tracing.ITracer
	proxy  proxy.IProxy

	client *clientgo.Client

	reqId *idutil.Generator
	reqW  wait.Wait

	pr *corev1.Runner

	smu    sync.RWMutex
	shards map[uint64]*Shard

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

	traces := tracer.SubscribeChannel(make(chan tracing.ITrace, 128))
	controller := &Controller{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
		nh:     nh,
		tracer: tracer,
		//proxy:           py,
		client:  client,
		be:      be,
		regionW: wait.New(),
		reqId:   idutil.NewGenerator(0, time.Now()),
		reqW:    wait.New(),
		pr:      runner,
		shards:  make(map[uint64]*Shard),
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
			_ = c.SyncShard(c.ctx, newObj.(*corev1.Region))
		},
		DeleteFunc: func(obj interface{}) {
			_ = c.RemoveShard(obj.(*corev1.Region))
		},
	})
	if err != nil {
		return err
	}
	syncs = append(syncs, regionRegistration.HasSynced)

	informerFactory.Start(stopping)

	if ok := cache.WaitForNamedCacheSync("runner-raft-controller", c.stopping, syncs...); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	regions, _ := informerFactory.Core().V1().Regions().Lister().List(labels.Everything())
	for i := range regions {
		region := regions[i]
		err = c.SyncShard(c.ctx, region)
		if err != nil {
			return err
		}
	}

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

				shard, ok := c.popShard(id)
				if !ok {
					continue
				}
				region := shard.getInfo()
				region.Status.Stat = *stat

				ctx := c.ctx
				var err error
				region, err = c.client.CoreV1().Regions().UpdateStatus(ctx, region, metav1.UpdateOptions{})
				if err != nil {

				} else {
					shard.updateInfo(region)
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

	regions := make([]int64, 0)
	leaders := make([]string, 0)
	for _, region := range c.shards {
		regions = append(regions, int64(region.id))
		lead := region.getLeader()
		if lead == 0 {
			continue
		}

		replicas := region.getInfo().Spec.Replicas
		if len(replicas) == 0 {
			continue
		}
		for _, replica := range replicas {
			if int64(lead) == replica.Id && replica.Runner == c.pr.Name {
				sv := semver.Version{
					Major: int64(region.id),
					Minor: replica.Id,
				}
				leaders = append(leaders, sv.String())
			}
		}
	}
	RegionCounter.Set(float64(len(regions)))
	LeaderCounter.Set(float64(len(leaders)))
	return regions, leaders
}

func (c *Controller) SubscribeTrace() <-chan tracing.ITrace {
	traceChannel := make(chan tracing.ITrace, 10)
	return c.tracer.SubscribeChannel(traceChannel)
}

func (c *Controller) CreateShard(ctx context.Context, region *corev1.Region) error {
	if !c.isLocalRegion(region) {
		return nil
	}

	_, err := c.startRaftShard(ctx, region)
	if err != nil {
		return err
	}

	key := region.Name
	data, _ := region.Marshal()
	tx := c.be.BatchTx()
	tx.Lock()
	_ = tx.UnsafePut(buckets.Region, []byte(key), data)
	tx.Unlock()
	_ = tx.Commit()
	c.be.ForceCommit()

	return nil
}

func (c *Controller) SyncShard(ctx context.Context, region *corev1.Region) error {
	if !c.isLocalRegion(region) {
		return nil
	}

	regionId := uint64(region.Spec.Id)
	if _, ok := c.shards[regionId]; ok {
		return nil
	}

	local, ok := c.getShard(regionId)
	if !ok {
		return c.CreateShard(ctx, region)
	}

	oldRegion := local.getInfo()
	patch, err := jsonpatch.CreateJSONPatch(oldRegion, region)
	if err != nil {
		return err
	}
	if patch.Len() == 0 {
		return nil
	}

	var patchErr error
	patch.Each(func(op string, path string, value any) {
		data, _ := json.Marshal(value)
		if op == "add" && strings.HasPrefix(path, "/replicas") {
			replica := new(pb.RegionReplica)
			if err = json.Unmarshal(data, replica); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
			if err = c.requestAddRegionReplica(ctx, uint64(oldRegion.Spec.Id), replica); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
		}
		if op == "replace" && strings.HasPrefix(path, "/leader") {
			var leader uint64
			_ = json.Unmarshal(data, &leader)
			if err = c.requestLeaderTransferRegion(ctx, uint64(oldRegion.Spec.Id), leader); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
		}
	})

	if patchErr != nil {
		return patchErr
	}

	local.updateInfo(region)
	return nil
}

func (c *Controller) RemoveShard(ri *corev1.Region) error {
	if !c.isLocalRegion(ri) {
		return nil
	}

	return nil
}

// prepareShards loads regions from backend.IBackend and start raft shard
func (c *Controller) prepareShards() error {
	regions := make([]*corev1.Region, 0)
	readTx := c.be.ReadTx()
	readTx.RLock()
	err := readTx.UnsafeForEach(buckets.Region, func(k, value []byte) error {
		region := &corev1.Region{}
		err := region.Unmarshal(value)
		if err != nil {
			return err
		}
		regions = append(regions, region)

		return nil
	})
	readTx.RUnlock()

	if err != nil {
		return err
	}

	ctx := c.ctx
	lg := c.cfg.Logger
	for i := range regions {
		region := regions[i]
		go func() {
			_, err = c.startRaftShard(ctx, region)
			if err != nil {
				lg.Error("start raft region",
					zap.Int64("id", region.Spec.Id),
					zap.Error(err))
				return
			}
			DefinitionsCounter.Add(float64(region.Status.Definitions))

			lg.Info("start raft region",
				zap.Int64("id", region.Spec.Id))
		}()
	}

	return nil
}

func (c *Controller) startRaftShard(ctx context.Context, ri *corev1.Region) (*Shard, error) {
	replicaId := uint64(0)
	var join bool
	for _, replica := range ri.Spec.Replicas {
		if replica.Runner == c.pr.Name {
			replicaId = uint64(replica.Id)
			join = replica.IsJoin
		}
	}

	if replicaId == 0 {
		return nil, fmt.Errorf("missing local replica")
	}

	members := map[uint64]string{}
	for id, urlText := range ri.InitialURL() {
		url, err := urlpkg.Parse(urlText)
		if err != nil {
			return nil, errors.Wrap(ErrRaftAddress, err.Error())
		}
		members[uint64(id)] = url.Host
	}

	regionId := ri.Spec.Id
	electRTT := ri.Spec.ElectionRTT
	heartbeatRTT := ri.Spec.HeartbeatRTT
	snapshotEntries := uint64(10000)
	compactionOverhead := uint64(1000)
	maxInMemLogSize := uint64(1 * 1024 * 1024 * 1024) // 1GB

	cfg := config.Config{
		ReplicaID:           replicaId,
		ShardID:             uint64(regionId),
		CheckQuorum:         true,
		PreVote:             true,
		ElectionRTT:         uint64(electRTT),
		HeartbeatRTT:        uint64(heartbeatRTT),
		SnapshotEntries:     snapshotEntries,
		CompactionOverhead:  compactionOverhead,
		OrderedConfigChange: true,
		MaxInMemLogSize:     maxInMemLogSize,
		WaitReady:           false,
	}

	rcfg := NewRegionConfig()
	rcfg.RaftRTTMillisecond = c.cfg.RaftRTTMillisecond
	rcfg.ElectionRTT = uint64(electRTT)
	rcfg.HeartbeatRTT = uint64(heartbeatRTT)
	rcfg.StatHeartBeatMs = c.cfg.HeartbeatMs

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	err := c.nh.StartOnDiskReplica(members, join, c.initDiskStateMachine, cfg)
	if err != nil {
		return nil, err
	}

	var shard *Shard
	ch := c.regionW.Register(cfg.ShardID)
	select {
	case <-c.stopping:
		return nil, fmt.Errorf("controller Stopping")
	case <-ctx.Done():
		return nil, ctx.Err()
	case x := <-ch:
		switch result := x.(type) {
		case error:
			err = result
		case *Shard:
			shard = result
		}
	}

	if err != nil {
		return nil, err
	}

	<-shard.ReadyNotify()
	if shard.leader != shard.getMember() && shard.isLeader() {
		if err = c.nh.RequestLeaderTransfer(shard.id, shard.leader); err != nil {
			return nil, err
		}
		<-shard.ReadyNotify()
	}

	shard.updateInfo(ri)
	shard.updateConfig(rcfg)
	shard.Start()
	c.setShard(shard)

	return shard, nil
}

func (c *Controller) requestAddRegionReplica(ctx context.Context, id uint64, replica *pb.RegionReplica) error {
	region, ok := c.getShard(id)
	if !ok {
		return errors.Wrapf(ErrNoRegion, "id is %d", id)
	}

	if _, ok = ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, region.cfg.ReqTimeout())
		defer cancel()
	}

	ms, err := c.nh.SyncGetShardMembership(ctx, id)
	if err != nil {
		return err
	}

	if _, ok = ms.Nodes[replica.Id]; ok {
		return errors.Wrapf(ErrRegionReplicaAdded, "add replica (%d) to region (%d)", replica.Id, id)
	}

	cc := ms.ConfigChangeID
	url, err := urlpkg.Parse(replica.RaftAddress)
	if err != nil {
		return errors.Wrap(ErrRaftAddress, err.Error())
	}
	target := url.Host

	err = c.nh.SyncRequestAddReplica(ctx, id, replica.Id, target, cc)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) requestLeaderTransferRegion(ctx context.Context, id, leader uint64) error {
	region, ok := c.getShard(id)
	if !ok {
		return errors.Wrapf(ErrNoRegion, "id is %d", id)
	}

	if region.getLeader() == leader || leader == 0 {
		return nil
	}

	err := c.nh.RequestLeaderTransfer(id, leader)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) DeployDefinition(ctx context.Context, definition *corev1.Definition) error {
	lg := c.cfg.Logger
	if definition.Spec.Region == 0 {
		lg.Warn("definition missing region",
			zap.String("name", definition.Name),
			zap.Int64("version", definition.Spec.Version))
		return nil
	}

	shardId := definition.Spec.Region
	shard, ok := c.getShard(uint64(shardId))
	if !ok {
		lg.Info("region running others",
			zap.Int64("id", shardId))
		return nil
	}

	lg.Info("definition deploy",
		zap.String("name", definition.Name),
		zap.Int64("version", definition.Spec.Version))
	req := &pb.ShardDeployDefinitionRequest{Definition: definition}
	if _, err := shard.DeployDefinition(ctx, req); err != nil {
		return err
	}

	return nil
}

func (c *Controller) ExecuteDefinition(ctx context.Context, process *corev1.Process) error {
	lg := c.cfg.Logger
	if process.Spec.Definition == "" || process.Status.Region == 0 {
		lg.Warn("invalid process instance")
		return nil
	}

	shardId := process.Status.Region
	shard, ok := c.getShard(uint64(shardId))
	if !ok {
		lg.Info("shard running others", zap.Int64("id", shardId))
		return nil
	}

	lg.Info("definition executed",
		zap.String("id", process.Spec.Definition),
		zap.Int64("version", process.Spec.Version))

	req := &pb.ShardExecuteDefinitionRequest{Process: process}
	resp, err := shard.ExecuteDefinition(ctx, req)
	if err != nil {
		return err
	}
	resp.Process.DeepCopyInto(process)

	return nil
}

func (c *Controller) GetProcess(ctx context.Context, req *pb.GetProcessRequest) (*pb.GetProcessResponse, error) {
	resp := &pb.GetProcessResponse{}
	region, ok := c.getShard(req.Region)
	if !ok {
		return nil, ErrNoRegion
	}

	instance, err := region.GetProcess(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	if err != nil {
		return nil, err
	}
	resp.Process = instance

	return resp, nil
}

func (c *Controller) getShard(id uint64) (*Shard, bool) {
	c.smu.RLock()
	region, ok := c.shards[id]
	c.smu.RUnlock()
	return region, ok
}

func (c *Controller) popShard(id uint64) (*Shard, bool) {
	c.smu.Lock()
	region, ok := c.shards[id]
	if !ok {
		c.smu.Unlock()
		return nil, false
	}
	delete(c.shards, id)
	c.smu.Unlock()
	return region, ok
}

func (c *Controller) setShard(shard *Shard) {
	c.smu.Lock()
	c.shards[shard.id] = shard
	c.smu.Unlock()
}

func (c *Controller) isLocalRegion(region *corev1.Region) bool {
	for _, replica := range region.Spec.Replicas {
		if replica.Runner == c.pr.Name {
			return true
		}
	}
	return false
}
