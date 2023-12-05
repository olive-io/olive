// Copyright 2023 The olive Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/pkg/jsonpatch"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/buckets"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

type Controller struct {
	Config

	ctx    context.Context
	cancel context.CancelFunc

	nh *dragonboat.NodeHost

	leaderCh chan raftio.LeaderInfo

	be      backend.IBackend
	regionW wait.Wait

	rsw *regionStatWatcher

	reqId *idutil.Generator
	reqW  wait.Wait
	reqCh chan chunk

	pr *pb.Runner

	rmu     sync.RWMutex
	regions map[uint64]*Region

	stopping <-chan struct{}
	done     chan struct{}
}

func NewController(ctx context.Context, cfg Config, be backend.IBackend, pr *pb.Runner) (*Controller, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewExample()
	}
	lg := cfg.Logger

	leaderCh := make(chan raftio.LeaderInfo, 100)
	el := newEventListener(leaderCh)

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

	// deep copy *pb.Runner
	runner := pr.Clone()

	ctx, cancel := context.WithCancel(ctx)
	controller := &Controller{
		Config:   cfg,
		ctx:      ctx,
		cancel:   cancel,
		nh:       nh,
		leaderCh: leaderCh,
		be:       be,
		regionW:  wait.New(),
		rsw:      newRegionStatWatcher(),
		reqId:    idutil.NewGenerator(0, time.Now()),
		reqW:     wait.New(),
		reqCh:    make(chan chunk, 128),
		pr:       runner,
		regions:  make(map[uint64]*Region),
	}

	go controller.listening()
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

func (c *Controller) listening() {
	for {
		select {
		case <-c.stopping:
			return
		case leaderInfo := <-c.leaderCh:
			region, ok := c.popRegion(leaderInfo.ShardID)
			if !ok {
				break
			}
			lead := leaderInfo.LeaderID
			term := leaderInfo.Term
			region.setLeader(lead)
			region.setTerm(term)
			region.notifyAboutChange()
			if lead != 0 {
				region.notifyAboutReady()
			}
			c.setRegion(region)
		}
	}
}

func (c *Controller) run() {
	defer c.Stop()
	for {
		select {
		case <-c.stopping:
			return
		case ch := <-c.reqCh:
			switch tt := ch.(type) {
			case *regionStatC:
				_, value := tt.unwrap()
				body := value.(*pb.RegionStat)
				c.rsw.Trigger(body)
				tt.write(struct{}{})
			case *raftReadC:
				ctx, value := tt.unwrap()
				body := value.(*raftQuery)
				result, err := c.nh.SyncRead(ctx, body.region, body.query)
				if err != nil {
					tt.failure(err)
				} else {
					tt.write(result)
				}
			case *raftProposeC:
				ctx, value := tt.unwrap()
				body := value.(*raftPropose)
				session := c.nh.GetNoOPSession(body.region)
				result, err := c.nh.SyncPropose(ctx, session, body.cmd)
				if err != nil {
					tt.failure(err)
				} else {
					tt.write(result)
				}
			}
		}
	}
}

func (c *Controller) Stop() {
	c.nh.Close()
	_ = c.rsw.close()

	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.cancel()
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
		if replica := replicas[lead]; replica.Runner == c.pr.Id {
			sv := semver.Version{
				Major: int64(region.id),
				Minor: int64(replica.Id),
			}
			leaders = append(leaders, sv.String())
		}
	}
	RegionCounter.Set(float64(len(regions)))
	LeaderCounter.Set(float64(len(leaders)))
	return regions, leaders
}

func (c *Controller) GetRegionWatcher() RegionStatWatcher {
	return c.rsw
}

func (c *Controller) CreateRegion(ctx context.Context, region *pb.Region) error {
	_, err := c.startRaftRegion(ctx, region)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%d", region.Id)
	data, _ := region.Marshal()
	tx := c.be.BatchTx()
	tx.Lock()
	tx.UnsafePut(buckets.Region, []byte(key), data)
	tx.Unlock()
	tx.Commit()

	return nil
}

func (c *Controller) SyncRegion(ctx context.Context, region *pb.Region) error {
	local, ok := c.getRegion(region.Id)
	if !ok {
		return c.CreateRegion(ctx, region)
	}

	regionInfo := local.getInfo()
	patch, err := jsonpatch.CreateJSONPatch(region, regionInfo)
	if err != nil {
		return err
	}
	if patch.Len() == 0 {
		return nil
	}

	return c.patchRegion(ctx, patch)
}

// prepareRegions loads regions from backend.IBackend and start raft regions
func (c *Controller) prepareRegions() error {
	regions := make([]*pb.Region, 0)
	readTx := c.be.ReadTx()
	readTx.RLock()
	err := readTx.UnsafeForEach(buckets.Region, func(k, v []byte) error {
		region := &pb.Region{}
		err := region.Unmarshal(v)
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
	lg := c.Logger
	for i := range regions {
		region := regions[i]
		_, err = c.startRaftRegion(ctx, region)
		if err != nil {
			return err
		}
		DefinitionsCounter.Add(float64(region.Definitions))

		lg.Info("start raft region",
			zap.Uint64("id", region.Id))
	}

	return nil
}

func (c *Controller) patchRegion(ctx context.Context, patch *jsonpatch.Patch) error {
	return nil
}

func (c *Controller) startRaftRegion(ctx context.Context, ri *pb.Region) (*Region, error) {
	members := map[uint64]string{}
	join := false

	replicaId := uint64(0)
	for id, replica := range ri.Replicas {
		peerURL, err := url.Parse(replica.RaftAddress)
		if err != nil {
			return nil, err
		}
		members[id] = peerURL.Host
		if replica.Runner == c.pr.Id {
			replicaId = id
		}
	}

	if replicaId == 0 {
		return nil, fmt.Errorf("missing local replica")
	}

	if ri.Replicas[replicaId].IsJoin {
		join = true
		members = map[uint64]string{}
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	err := c.nh.StartOnDiskReplica(members, join, c.InitDiskStateMachine, cfg)
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
	c.setRegion(region)

	return region, nil
}

func (c *Controller) DeployDefinition(ctx context.Context, definition *pb.Definition) error {
	if definition.Header == nil || definition.Header.Region == 0 {
		c.Logger.Warn("definition missing region",
			zap.String("id", definition.Id),
			zap.Uint64("version", definition.Version))
		return nil
	}

	regionId := definition.Header.Region
	region, ok := c.getRegion(regionId)
	if !ok {
		c.Logger.Info("definition deploys others",
			zap.String("id", definition.Id),
			zap.Uint64("version", definition.Version))
		return nil
	}

	c.Logger.Info("definition deploy",
		zap.String("id", definition.Id),
		zap.Uint64("version", definition.Version))
	if err := region.deployDefinition(definition); err != nil {
		return err
	}

	return nil
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
