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

package raft

import (
	"context"
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/lni/dragonboat/v4/config"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	corev1 "github.com/olive-io/olive/apis/core/v1"
	"github.com/olive-io/olive/pkg/jsonpatch"
	"github.com/olive-io/olive/runner/buckets"
)

func (c *Controller) CreateShard(ctx context.Context, region *corev1.Region) error {
	if !c.isLocalRegion(region) {
		return nil
	}

	return c.persistStartShard(ctx, region)
}

func (c *Controller) SyncShard(ctx context.Context, region *corev1.Region, patch *jsonpatch.Patch) error {
	if !c.isLocalRegion(region) {
		return nil
	}

	regionId := uint64(region.Spec.Id)

	_, ok := c.getShard(regionId)
	if !ok {
		return c.persistStartShard(ctx, region)
	}

	if region.Status.Stat.Leader != region.Spec.Leader &&
		region.Spec.Leader == c.pr.Spec.ID {
		if err := c.requestLeaderTransferShard(ctx, uint64(region.Spec.Id), uint64(region.Spec.Leader)); err != nil {
			return err
		}
	}

	if patch == nil || patch.Len() == 0 {
		return nil
	}

	var err, patchErr error
	patch.Each(func(op string, path string, value any) {
		data, _ := json.Marshal(value)
		if op == "add" && strings.HasPrefix(path, "/spec/initialReplicas") {
			replica := new(corev1.RegionReplica)
			if err = json.Unmarshal(data, replica); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
			if err = c.requestAddShardReplica(ctx, uint64(region.Spec.Id), replica); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
		}
		if op == "replace" && strings.HasPrefix(path, "/spec/leader") {
			var leader uint64
			_ = json.Unmarshal(data, &leader)
			if err = c.requestLeaderTransferShard(ctx, uint64(region.Spec.Id), leader); err != nil {
				patchErr = multierr.Append(patchErr, err)
				return
			}
		}
	})

	if patchErr != nil {
		return patchErr
	}

	return nil
}

func (c *Controller) RemoveShard(ri *corev1.Region) error {
	if !c.isLocalRegion(ri) {
		return nil
	}

	return nil
}

func (c *Controller) persistStartShard(ctx context.Context, region *corev1.Region) error {

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
	for _, replica := range ri.Spec.InitialReplicas {
		if replica.Runner == c.pr.Name {
			replicaId = uint64(replica.Id)
			join = replica.IsJoin
		}
	}

	if replicaId == 0 {
		return nil, fmt.Errorf("missing local replica")
	}

	members := map[uint64]string{}
	if !join {
		for id, urlText := range ri.InitialURL() {
			url, err := urlpkg.Parse(urlText)
			if err != nil {
				return nil, errors.Wrap(ErrRaftAddress, err.Error())
			}
			members[uint64(id)] = url.Host
		}
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

	shard.updateConfig(rcfg)
	shard.Start()
	c.setShard(shard)

	return shard, nil
}

func (c *Controller) requestAddShardReplica(ctx context.Context, id uint64, replica *corev1.RegionReplica) error {
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

	replicaId := uint64(replica.Id)
	if _, ok = ms.Nodes[replicaId]; ok {
		return errors.Wrapf(ErrRegionReplicaAdded, "add replica (%d) to region (%d)", replica.Id, id)
	}

	cc := ms.ConfigChangeID
	url, err := urlpkg.Parse(replica.RaftAddress)
	if err != nil {
		return errors.Wrap(ErrRaftAddress, err.Error())
	}
	target := url.Host

	err = c.nh.SyncRequestAddReplica(ctx, id, replicaId, target, cc)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) requestLeaderTransferShard(ctx context.Context, id, leader uint64) error {
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
	for _, replica := range region.Spec.InitialReplicas {
		if replica.Runner == c.pr.Name {
			return true
		}
	}
	return false
}
