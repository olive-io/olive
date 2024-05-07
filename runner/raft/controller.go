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
	"google.golang.org/protobuf/proto"

	pb "github.com/olive-io/olive/api/olivepb"
	dsy "github.com/olive-io/olive/pkg/discovery"
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

	reqId *idutil.Generator
	reqW  wait.Wait

	pr *pb.Runner

	rmu     sync.RWMutex
	regions map[uint64]*Region

	stopping <-chan struct{}
	done     chan struct{}
}

func NewController(ctx context.Context, cfg Config, be backend.IBackend, discovery dsy.IDiscovery, pr *pb.Runner) (*Controller, error) {
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

	gwCfg := proxy.Config{
		Logger:    lg,
		Discovery: discovery,
	}
	py, err := proxy.NewProxy(gwCfg)
	if err != nil {
		return nil, err
	}

	// deep copy *pb.Runner
	runner := pr.Clone()
	ctx, cancel := context.WithCancel(ctx)

	traces := tracer.SubscribeChannel(make(chan tracing.ITrace, 128))
	controller := &Controller{
		ctx:     ctx,
		cancel:  cancel,
		cfg:     cfg,
		nh:      nh,
		tracer:  tracer,
		proxy:   py,
		be:      be,
		regionW: wait.New(),
		reqId:   idutil.NewGenerator(0, time.Now()),
		reqW:    wait.New(),
		pr:      runner,
		regions: make(map[uint64]*Region),
	}

	go controller.watchTrace(traces)
	return controller, nil
}

func (c *Controller) Start(stopping <-chan struct{}) error {
	c.stopping = stopping
	c.done = make(chan struct{}, 1)

	var err error
	if err = c.prepareRegions(); err != nil {
		return err
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
				region, ok := c.popRegion(tt.ShardID)
				if !ok {
					break
				}
				lead := tt.LeaderID
				term := tt.Term
				region.setTerm(term)
				oldLead := region.getLeader()
				newLeader := oldLead != lead && lead != 0
				region.setLeader(lead, newLeader)
				if lead != 0 {
					region.notifyAboutReady()
				}
				region.notifyAboutChange()
				c.setRegion(region)

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

func (c *Controller) RunnerStat() ([]uint64, []string) {
	c.rmu.RLock()
	defer c.rmu.RUnlock()

	regions := make([]uint64, 0)
	leaders := make([]string, 0)
	for _, region := range c.regions {
		regions = append(regions, region.id)
		lead := region.getLeader()
		if lead == 0 {
			continue
		}

		replicas := region.getInfo().Replicas
		if len(replicas) == 0 {
			continue
		}
		for _, replica := range replicas {
			if lead == replica.Id && replica.Runner == c.pr.Id {
				sv := semver.Version{
					Major: int64(region.id),
					Minor: int64(replica.Id),
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

func (c *Controller) CreateRegion(ctx context.Context, region *pb.Region) error {
	_, err := c.startRaftRegion(ctx, region)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%d", region.Id)
	data, _ := proto.Marshal(region)
	tx := c.be.BatchTx()
	tx.Lock()
	_ = tx.UnsafePut(buckets.Region, []byte(key), data)
	tx.Unlock()
	_ = tx.Commit()
	c.be.ForceCommit()

	return nil
}

func (c *Controller) SyncRegion(ctx context.Context, region *pb.Region) error {
	local, ok := c.getRegion(region.Id)
	if !ok {
		return c.CreateRegion(ctx, region)
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
			if err = c.requestAddRegionReplica(ctx, oldRegion.Id, replica); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
		}
		if op == "replace" && strings.HasPrefix(path, "/leader") {
			var leader uint64
			_ = json.Unmarshal(data, &leader)
			if err = c.requestLeaderTransferRegion(ctx, oldRegion.Id, leader); err != nil {
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

// prepareRegions loads regions from backend.IBackend and start raft regions
func (c *Controller) prepareRegions() error {
	regions := make([]*pb.Region, 0)
	readTx := c.be.ReadTx()
	readTx.RLock()
	err := readTx.UnsafeForEach(buckets.Region, func(k, v []byte) error {
		region := &pb.Region{}
		err := proto.Unmarshal(v, region)
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
			_, err = c.startRaftRegion(ctx, region)
			if err != nil {
				lg.Error("start raft region",
					zap.Uint64("id", region.Id),
					zap.Error(err))
				return
			}
			DefinitionsCounter.Add(float64(region.Definitions))

			lg.Info("start raft region",
				zap.Uint64("id", region.Id))
		}()
	}

	return nil
}

func (c *Controller) startRaftRegion(ctx context.Context, ri *pb.Region) (*Region, error) {
	replicaId := uint64(0)
	var join bool
	for _, replica := range ri.Replicas {
		if replica.Runner == c.pr.Id {
			replicaId = replica.Id
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
		members[id] = url.Host
	}

	regionId := ri.Id
	electRTT := ri.ElectionRTT
	heartbeatRTT := ri.HeartbeatRTT
	snapshotEntries := uint64(10000)
	compactionOverhead := uint64(1000)
	maxInMemLogSize := uint64(1 * 1024 * 1024 * 1024) // 1GB

	cfg := config.Config{
		ReplicaID:           replicaId,
		ShardID:             regionId,
		CheckQuorum:         true,
		PreVote:             true,
		ElectionRTT:         electRTT,
		HeartbeatRTT:        heartbeatRTT,
		SnapshotEntries:     snapshotEntries,
		CompactionOverhead:  compactionOverhead,
		OrderedConfigChange: true,
		MaxInMemLogSize:     maxInMemLogSize,
		WaitReady:           false,
	}

	rcfg := NewRegionConfig()
	rcfg.RaftRTTMillisecond = c.cfg.RaftRTTMillisecond
	rcfg.ElectionRTT = electRTT
	rcfg.HeartbeatRTT = heartbeatRTT
	rcfg.StatHeartBeatMs = c.cfg.HeartbeatMs

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	err := c.nh.StartOnDiskReplica(members, join, c.initDiskStateMachine, cfg)
	if err != nil {
		return nil, err
	}

	var region *Region
	ch := c.regionW.Register(cfg.ShardID)
	select {
	case <-c.stopping:
		return nil, fmt.Errorf("controller Stopping")
	case <-ctx.Done():
		return nil, ctx.Err()
	case x := <-ch:
		switch tt := x.(type) {
		case error:
			err = tt
		case *Region:
			region = tt
		}
	}

	if err != nil {
		return nil, err
	}

	<-region.ReadyNotify()
	if region.leader != region.getMember() && region.isLeader() {
		if err = c.nh.RequestLeaderTransfer(region.id, region.leader); err != nil {
			return nil, err
		}
	}

	region.updateInfo(ri)
	region.updateConfig(rcfg)
	region.Start()
	c.setRegion(region)

	return region, nil
}

func (c *Controller) requestAddRegionReplica(ctx context.Context, id uint64, replica *pb.RegionReplica) error {
	region, ok := c.getRegion(id)
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
	region, ok := c.getRegion(id)
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

func (c *Controller) DeployDefinition(ctx context.Context, definition *pb.Definition) error {
	lg := c.cfg.Logger
	if definition.Region == 0 {
		lg.Warn("definition missing region",
			zap.String("id", definition.Id),
			zap.Uint64("version", definition.Version))
		return nil
	}

	regionId := definition.Region
	region, ok := c.getRegion(regionId)
	if !ok {
		lg.Info("region running others",
			zap.Uint64("id", regionId))
		return nil
	}

	lg.Info("definition deploy",
		zap.String("id", definition.Id),
		zap.Uint64("version", definition.Version))
	req := &pb.RegionDeployDefinitionRequest{Definition: definition}
	if _, err := region.DeployDefinition(ctx, req); err != nil {
		return err
	}

	return nil
}

func (c *Controller) ExecuteDefinition(ctx context.Context, instance *pb.ProcessInstance) error {
	lg := c.cfg.Logger
	if instance.DefinitionsId == "" || instance.Region == 0 {
		lg.Warn("invalid process instance")
		return nil
	}

	regionId := instance.Region
	region, ok := c.getRegion(regionId)
	if !ok {
		lg.Info("region running others", zap.Uint64("id", regionId))
		return nil
	}

	lg.Info("definition executed",
		zap.String("id", instance.DefinitionsId),
		zap.Uint64("version", instance.DefinitionsVersion))

	req := &pb.RegionExecuteDefinitionRequest{ProcessInstance: instance}
	resp, err := region.ExecuteDefinition(ctx, req)
	if err != nil {
		return err
	}
	*instance = *resp.ProcessInstance

	return nil
}

func (c *Controller) GetProcessInstance(ctx context.Context, req *pb.GetProcessInstanceRequest) (*pb.GetProcessInstanceResponse, error) {
	resp := &pb.GetProcessInstanceResponse{}
	region, ok := c.getRegion(req.Region)
	if !ok {
		return nil, ErrNoRegion
	}

	instance, err := region.GetProcessInstance(ctx, req.DefinitionId, req.DefinitionVersion, req.Id)
	if err != nil {
		return nil, err
	}
	resp.Instance = instance

	return resp, nil
}

func (c *Controller) getRegion(id uint64) (*Region, bool) {
	c.rmu.RLock()
	region, ok := c.regions[id]
	c.rmu.RUnlock()
	return region, ok
}

func (c *Controller) popRegion(id uint64) (*Region, bool) {
	c.rmu.Lock()
	region, ok := c.regions[id]
	if !ok {
		c.rmu.Unlock()
		return nil, false
	}
	delete(c.regions, id)
	c.rmu.Unlock()
	return region, ok
}

func (c *Controller) setRegion(region *Region) {
	c.rmu.Lock()
	c.regions[region.id] = region
	c.rmu.Unlock()
}
