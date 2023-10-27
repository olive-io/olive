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

package server

import (
	"hash/fnv"
	"strings"

	"github.com/lni/dragonboat/v4/config"
)

type ClusterState string

const (
	NewCluster      ClusterState = "new"
	ExistingCluster ClusterState = "existing"
)

type Option struct {
	Name string

	RaftAddress string

	ListenAddress string

	InitialAdvertisePeer string

	InitialCluster []string

	InitialClusterState ClusterState

	// Node config
	Node config.Config
	// NodeHost config
	NodeHost config.NodeHostConfig
}

func (opt *Option) Validate() []error {
	errs := make([]error, 0)
	return errs
}

func (opt *Option) Apply() error {
	if opt.Node.ShardID == 0 {
		opt.Node.ShardID = 1
	}

	id := digitize(opt.Name)
	opt.Node.ReplicaID = uint64(id)

	opt.NodeHost.RaftAddress = opt.RaftAddress

	return nil
}

func (opt *Option) discoveryMembers() map[uint64]string {
	initialMembers := make(map[uint64]string)
	if len(opt.InitialCluster) == 0 {
		opt.InitialCluster = []string{opt.Name + "=" + opt.RaftAddress}
	}

	for _, initial := range opt.InitialCluster {
		parts := strings.Split(initial, "=")
		if len(parts) != 2 {
			continue
		}
		id := uint64(digitize(strings.TrimSpace(parts[0])))
		initialMembers[id] = strings.TrimSpace(parts[1])
	}

	return initialMembers
}

// digitize converts string to uint32
func digitize(name string) uint32 {
	h := fnv.New32()
	h.Write([]byte(name))
	return h.Sum32()
}
