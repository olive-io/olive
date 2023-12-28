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

package raft_test

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	pb "github.com/olive-io/olive/api/olivepb"
	dsy "github.com/olive-io/olive/pkg/discovery"
	"github.com/olive-io/olive/pkg/discovery/testdata"
	"github.com/olive-io/olive/runner/backend"
	"github.com/olive-io/olive/runner/raft"
)

const (
	Peer1 = "http://localhost:60001"
	Peer2 = "http://localhost:60002"
	Peer3 = "http://localhost:60003"
)

var peerId = map[string]uint64{
	Peer1: 1,
	Peer2: 2,
	Peer3: 3,
}

func newController(t *testing.T, discovery dsy.IDiscovery, peer string) (*raft.Controller, func()) {
	ctx := context.Background()
	cfg := raft.NewConfig()
	id := peerId[peer]
	root := fmt.Sprintf(".testbe%d", id)
	cfg.DataDir = root
	url, _ := urlpkg.Parse(peer)
	cfg.RaftAddress = url.Host
	pr := &pb.Runner{
		Id:            id,
		ListenPeerURL: peer,
		HeartbeatMs:   2000,
		Hostname:      fmt.Sprintf("n%d", id),
		Cpu:           32280,
		Memory:        16 * 1024 * 1024 * 1024,
		Version:       "1.0",
	}

	bcfg := backend.DefaultBackendConfig()
	bcfg.Dir = root + "/db"
	bcfg.WAL = root + "/wal"
	_ = os.MkdirAll(root, os.ModePerm)
	be := backend.New(bcfg)

	destroy := func() {
		_ = be.Close()
		_ = os.RemoveAll(root)
	}

	controller, err := raft.NewController(ctx, cfg, be, discovery, pr)
	if !assert.NoError(t, err) {
		destroy()
		return nil, func() {}
	}

	stop := make(chan struct{}, 1)
	err = controller.Start(stop)
	if !assert.NoError(t, err) {
		destroy()
		return nil, func() {}
	}

	destroy1 := func() {
		destroy()
		close(stop)
	}

	return controller, destroy1
}

func TestNewController(t *testing.T) {
	discovery, cancel := testdata.TestDiscovery(t)
	defer cancel()

	c, destroy := newController(t, discovery, Peer1)
	defer destroy()

	rc, lc := c.RunnerStat()
	t.Logf("rc=%v, lc=%v\n", rc, lc)
}

func TestController_SyncRegion(t *testing.T) {
	discovery, cancel := testdata.TestDiscovery(t)
	defer cancel()

	c1, destroy := newController(t, discovery, Peer1)
	defer destroy()

	region := &pb.Region{
		Id:           1,
		Name:         "r1",
		DeploymentId: 0,
		Replicas: map[uint64]*pb.RegionReplica{
			1: &pb.RegionReplica{
				Id:          1,
				Runner:      1,
				Region:      1,
				RaftAddress: Peer1,
				IsJoin:      false,
				Initial:     map[uint64]string{1: Peer1},
			},
		},
		ElectionRTT:  10,
		HeartbeatRTT: 1,
		Leader:       1,
	}

	ctx := context.TODO()
	err := c1.SyncRegion(ctx, region)
	if !assert.NoError(t, err) {
		return
	}

	c2, destroy2 := newController(t, discovery, Peer2)
	defer destroy2()

	region1 := deepCopyRegion(region)

	region1.Replicas[2] = &pb.RegionReplica{
		Id:          2,
		Runner:      2,
		Region:      1,
		RaftAddress: Peer2,
		IsNonVoting: false,
		IsWitness:   false,
		IsJoin:      true,
		Initial:     map[uint64]string{},
	}

	ech := make(chan error, 1)
	go func() {
		err = c1.SyncRegion(ctx, region1)
		if !assert.NoError(t, err) {
			ech <- err
			return
		}
		ech <- nil
	}()

	err = c2.SyncRegion(ctx, region1)
	if !assert.NoError(t, err) {
		return
	}

	select {
	case err = <-ech:
		if err != nil {
			t.Fatal(err)
		}
	}

	region1 = deepCopyRegion(region1)
	region1.Leader = 2

	err = c1.SyncRegion(ctx, region1)
	if !assert.NoError(t, err) {
		return
	}
}

func deepCopyRegion(region *pb.Region) *pb.Region {
	r := new(pb.Region)
	data, _ := region.Marshal()
	_ = r.Unmarshal(data)
	return r
}
