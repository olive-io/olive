// Copyright 2023 Lack (xingyys@gmail.com).
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
	"sync"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/client"
	"github.com/olive-io/olive/runner/backend"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

type Controller struct {
	Config

	nh *dragonboat.NodeHost

	leaderCh chan raftio.LeaderInfo

	be backend.IBackend
	w  wait.Wait

	oct *client.Client

	lg *zap.Logger

	pr *pb.Runner

	rmu     sync.RWMutex
	regions map[uint64]*Region

	stopping <-chan struct{}
	done     chan struct{}
}

func NewController(cfg Config, oct *client.Client, be backend.IBackend, pr *pb.Runner, stopping <-chan struct{}) (*Controller, error) {
	lg := cfg.Logger
	if lg == nil {
		lg = zap.NewExample()
	}

	leaderCh := make(chan raftio.LeaderInfo, 10)
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
	runner := new(pb.Runner)
	*runner = *pr

	controller := &Controller{
		nh:       nh,
		leaderCh: leaderCh,
		be:       be,
		w:        wait.New(),
		oct:      oct,
		pr:       runner,
		lg:       lg,
		regions:  make(map[uint64]*Region),
		stopping: stopping,
		done:     make(chan struct{}, 1),
	}

	go controller.run()
	return controller, nil
}

func (c *Controller) run() {
	defer c.Stop()
	for {
		select {
		case <-c.stopping:
			return
		case leaderInfo := <-c.leaderCh:
			_ = leaderInfo
		}
	}
}

func (c *Controller) Stop() {
	c.nh.Close()

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
		lead := region.getLeader()
		regions = append(regions, lead)
		//if lead != 0 {
		//	replica, ok := region.Replicas[region.Leader]
		//	if ok && replica.Runner == mrg.pr.Id {
		//		sv := semver.Version{
		//			Major: int64(replica.Runner),
		//			Minor: int64(replica.Id),
		//		}
		//		leaders = append(leaders, sv.String())
		//	}
		//}
	}

	return regions, leaders
}

func (c *Controller) CreateRegion(region *pb.Region) error {
	members := map[uint64]string{}
	join := false
	cfg := config.Config{
		ReplicaID:               0,
		ShardID:                 0,
		CheckQuorum:             true,
		PreVote:                 true,
		ElectionRTT:             0,
		HeartbeatRTT:            0,
		SnapshotEntries:         0,
		CompactionOverhead:      0,
		OrderedConfigChange:     false,
		MaxInMemLogSize:         0,
		SnapshotCompressionType: 0,
		EntryCompressionType:    0,
		DisableAutoCompactions:  false,
		IsNonVoting:             false,
		IsWitness:               false,
		Quiesce:                 false,
		WaitReady:               true,
	}
	err := c.nh.StartOnDiskReplica(members, join, c.InitDiskStateMachine, cfg)
	if err != nil {
		return err
	}

	ech := c.w.Register(cfg.ShardID)

	select {
	case ch := <-ech:
		err = ch.(error)
	}

	if err != nil {
		// handle error
	}

	//ms, _ := mrg.nh.SyncGetShardMembership(context.TODO(), cfg.ShardID)
	//info := mrg.nh.GetNodeHostInfo(dragonboat.DefaultNodeHostInfoOption)

	return nil
}
