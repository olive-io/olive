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

package runner

import (
	"sync"

	"github.com/coreos/go-semver/semver"
	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/raftio"
	pb "github.com/olive-io/olive/api/olivepb"
	"github.com/olive-io/olive/runner/backend"
	"go.etcd.io/etcd/pkg/v3/wait"
	"go.uber.org/zap"
)

type MultiRaftGroup struct {
	nh *dragonboat.NodeHost

	leaderCh chan raftio.LeaderInfo

	be backend.IBackend
	w  wait.Wait

	lg *zap.Logger

	pr *pb.Runner

	rmu     sync.RWMutex
	regions map[uint64]*pb.Region

	stopping <-chan struct{}
	done     chan struct{}
}

func (r *Runner) newMultiRaftGroup() (*MultiRaftGroup, error) {
	cfg := r.Config
	lg := cfg.Logger
	if lg == nil {
		lg = zap.NewExample()
	}

	leaderCh := make(chan raftio.LeaderInfo, 10)
	el := newEventListener(leaderCh)

	sl := newSystemListener()

	dir := cfg.RegionDir()
	nhConfig := config.NodeHostConfig{
		NodeHostDir:         dir,
		RTTMillisecond:      cfg.RaftRTTMillisecond,
		RaftAddress:         cfg.PeerListen,
		EnableMetrics:       true,
		RaftEventListener:   el,
		SystemEventListener: sl,
		NotifyCommit:        true,
	}

	lg.Debug("start multi raft group",
		zap.String("module", "dragonboat"),
		zap.String("dir", dir),
		zap.String("listen", cfg.PeerListen))

	nh, err := dragonboat.NewNodeHost(nhConfig)
	if err != nil {
		return nil, err
	}

	be := r.be
	mrg := &MultiRaftGroup{
		nh:       nh,
		leaderCh: leaderCh,
		be:       be,
		w:        wait.New(),
		pr:       r.pr,
		lg:       lg,
		regions:  make(map[uint64]*pb.Region),
		stopping: r.StoppingNotify(),
		done:     make(chan struct{}, 1),
	}

	go mrg.run()
	return mrg, nil
}

func (mrg *MultiRaftGroup) run() {
	defer mrg.Stop()
	for {
		select {
		case <-mrg.stopping:
			return
		case leaderInfo := <-mrg.leaderCh:
			_ = leaderInfo
		}
	}
}

func (mrg *MultiRaftGroup) Stop() {
	mrg.nh.Close()

	select {
	case <-mrg.done:
	default:
		close(mrg.done)
	}
}

func (mrg *MultiRaftGroup) runnerStat() ([]uint64, []string) {
	mrg.rmu.RLock()
	defer mrg.rmu.RUnlock()

	regions := make([]uint64, 0)
	leaders := make([]string, 0)
	for _, region := range mrg.regions {
		regions = append(regions, region.Id)
		if region.Leader != 0 {
			replica, ok := region.Replicas[region.Leader]
			if ok && replica.Runner == mrg.pr.Id {
				sv := semver.Version{
					Major: int64(replica.Runner),
					Minor: int64(replica.Id),
				}
				leaders = append(leaders, sv.String())
			}
		}
	}

	return regions, leaders
}

func (mrg *MultiRaftGroup) CreateRegion() error {
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
	err := mrg.nh.StartOnDiskReplica(members, join, mrg.InitDiskStateMachine, cfg)
	if err != nil {
		return err
	}

	ech := mrg.w.Register(cfg.ShardID)

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
